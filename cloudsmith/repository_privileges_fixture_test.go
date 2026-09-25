// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testAccRepositoryPrivilegesCaller(t *testing.T) (string, string) {
	t.Helper()

	// Config strings are constructed before resource.Test checks TF_ACC.
	// Apply the same gate before performing any account discovery.
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
	}
	testAccPreCheck(t)

	kind, slug, err := testRepositoryPrivilegesCaller(testAccAPIClient(t), testAccNamespace())
	if err != nil {
		t.Fatalf("error resolving authenticated account for privilege fixture: %v", err)
	}
	return kind, slug
}

func testRepositoryPrivilegesCaller(pc *providerConfig, organization string) (string, string, error) {
	req := pc.APIClient.UserApi.UserSelf(pc.Auth)
	self, _, err := pc.APIClient.UserApi.UserSelfExecute(req)
	if err != nil {
		return "", "", fmt.Errorf("error retrieving authenticated account: %w", err)
	}
	slug := self.GetSlug()
	if slug == "" {
		return "", "", fmt.Errorf("authenticated account has no slug")
	}

	privilege, err := authenticatedAccountAdminPrivilege(pc, organization, slug, nil)
	if err != nil {
		return "", "", fmt.Errorf("error resolving authenticated account privilege: %w", err)
	}
	if privilege.HasService() {
		return "service", privilege.GetService(), nil
	}
	return "user", privilege.GetUser(), nil
}

func testAccRepositoryPrivilegesCallerBlock(kind, slug string) string {
	return fmt.Sprintf(`
	%s {
		privilege = "Admin"
		slug      = %q
	}
`, kind, slug)
}

func testAccRepositoryPrivilegeForResource(resourceName, kind, identityResourceName, privilege string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if s == nil || s.RootModule() == nil {
			return fmt.Errorf("missing root module state")
		}
		identity, ok := s.RootModule().Resources[identityResourceName]
		if !ok || identity == nil || identity.Primary == nil {
			return fmt.Errorf("missing identity resource state: %s", identityResourceName)
		}
		slug := identity.Primary.Attributes["slug"]
		if slug == "" {
			return fmt.Errorf("missing slug for identity resource: %s", identityResourceName)
		}
		return resource.TestCheckTypeSetElemNestedAttrs(resourceName, kind+".*", map[string]string{
			"slug":      slug,
			"privilege": privilege,
		})(s)
	}
}

func TestRepositoryPrivilegesCaller(t *testing.T) {
	t.Parallel()

	const (
		organization = "example-org"
		accountSlug  = "caller-account"
		servicePath  = "/orgs/" + organization + "/services/" + accountSlug + "/"
		memberPath   = "/orgs/" + organization + "/members/" + accountSlug + "/"
	)
	tests := []struct {
		name          string
		selfStatus    int
		selfSlug      string
		serviceStatus int
		memberStatus  int
		wantKind      string
		wantError     string
		wantPaths     []string
	}{
		{
			name: "service", selfStatus: http.StatusOK, selfSlug: accountSlug,
			serviceStatus: http.StatusOK, wantKind: "service",
			wantPaths: []string{"/user/self/", servicePath},
		},
		{
			name: "human", selfStatus: http.StatusOK, selfSlug: accountSlug,
			serviceStatus: http.StatusNotFound, memberStatus: http.StatusOK, wantKind: "user",
			wantPaths: []string{"/user/self/", servicePath, memberPath},
		},
		{
			name: "self lookup fails", selfStatus: http.StatusUnauthorized,
			wantError: "error retrieving authenticated account",
			wantPaths: []string{"/user/self/"},
		},
		{
			name: "missing slug", selfStatus: http.StatusOK,
			wantError: "authenticated account has no slug",
			wantPaths: []string{"/user/self/"},
		},
		{
			name: "service lookup forbidden", selfStatus: http.StatusOK, selfSlug: accountSlug,
			serviceStatus: http.StatusForbidden,
			wantError:     "error determining whether authenticated account is a service",
			wantPaths:     []string{"/user/self/", servicePath},
		},
		{
			name: "member lookup fails", selfStatus: http.StatusOK, selfSlug: accountSlug,
			serviceStatus: http.StatusNotFound, memberStatus: http.StatusInternalServerError,
			wantError: "error determining whether authenticated account is an organization member",
			wantPaths: []string{"/user/self/", servicePath, memberPath},
		},
		{
			name: "unknown account", selfStatus: http.StatusOK, selfSlug: accountSlug,
			serviceStatus: http.StatusNotFound, memberStatus: http.StatusNotFound,
			wantError: "neither a service nor an organization member",
			wantPaths: []string{"/user/self/", servicePath, memberPath},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				if r.Method != http.MethodGet {
					t.Errorf("unexpected method: %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/user/self/":
					w.WriteHeader(tt.selfStatus)
					fmt.Fprintf(w, `{"slug":%q}`, tt.selfSlug)
				case servicePath:
					w.WriteHeader(tt.serviceStatus)
					fmt.Fprintf(w, `{"name":"Caller service","slug":%q,"slug_perm":%q}`, accountSlug, accountSlug)
				case memberPath:
					w.WriteHeader(tt.memberStatus)
					fmt.Fprintf(w, `{"user":%q}`, accountSlug)
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					http.Error(w, "unexpected request", http.StatusNotFound)
				}
			}))
			kind, slug, err := testRepositoryPrivilegesCaller(testPrivilegesProviderConfig(server), organization)
			server.Close()

			if !slices.Equal(paths, tt.wantPaths) {
				t.Errorf("request paths = %v, want %v", paths, tt.wantPaths)
			}
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want %q", err, tt.wantError)
				}
				if kind != "" || slug != "" {
					t.Fatalf("failed lookup returned identity: %s %q", kind, slug)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if kind != tt.wantKind || slug != accountSlug {
				t.Fatalf("identity = %s %q, want %s %q", kind, slug, tt.wantKind, accountSlug)
			}
		})
	}
}

func TestRepositoryPrivilegesConfigsPreserveCaller(t *testing.T) {
	t.Parallel()

	configs := []struct {
		name   string
		config func(string, string) string
	}{
		{"basic", testAccRepositoryPrivilegesConfigBasic},
		{"update", testAccRepositoryPrivilegesConfigBasicUpdatePrivilege},
		{"team", testAccRepositoryPrivilegesConfigBasicAddTeam},
		{"another team", testAccRepositoryPrivilegesConfigBasicAddAnotherTeam},
		{"data source", testAccDataSourceRepositoryPrivilegesConfigBasic},
	}
	for _, kind := range []string{"service", "user"} {
		for _, tt := range configs {
			t.Run(kind+"/"+tt.name, func(t *testing.T) {
				t.Parallel()

				block := testAccRepositoryPrivilegesCallerBlock(kind, "caller-account")
				wantBlock := fmt.Sprintf("\n\t%s {\n\t\tprivilege = \"Admin\"\n\t\tslug      = \"caller-account\"\n\t}\n", kind)
				if block != wantBlock {
					t.Fatalf("caller block = %q, want %q", block, wantBlock)
				}
				config := tt.config("test-repository", block)
				if strings.Count(config, block) != 1 {
					t.Fatalf("config must include exactly one caller Admin block:\n%s", config)
				}
				if strings.Contains(config, "cloudsmith_user_self") {
					t.Fatal("config must use the resolved account kind, not assume user_self is a user")
				}
			})
		}
	}
}

func TestRepositoryPrivilegesAcceptanceGate(t *testing.T) {
	t.Setenv(resource.EnvTfAcc, "")
	t.Setenv("CLOUDSMITH_API_KEY", "test-api-key")
	t.Setenv("CLOUDSMITH_NAMESPACE", "example-org")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "account discovery must remain gated", http.StatusInternalServerError)
	}))
	defer server.Close()
	t.Setenv("CLOUDSMITH_API_HOST", server.URL)

	t.Run("resource", TestAccRepositoryPrivileges_basic)
	t.Run("data source", TestAccDataSourceRepositoryPrivileges_basic)
	if got := requests.Load(); got != 0 {
		t.Fatalf("acceptance fixtures made %d requests with TF_ACC unset", got)
	}
}

func TestRepositoryPrivilegeForResource(t *testing.T) {
	t.Parallel()

	const (
		resourceName = "cloudsmith_repository_privileges.test"
		identityName = "cloudsmith_service.test"
		testSlug     = "test-identity"
		callerSlug   = "caller-account"
	)
	tests := []struct {
		name            string
		testIndex       string
		callerIndex     string
		testPrivilege   string
		callerPrivilege string
		wantPrivilege   string
		wantError       bool
	}{
		{"read after caller", "1", "0", "Read", "Admin", "Read", false},
		{"write before caller", "0", "1", "Write", "Admin", "Write", false},
		{"unordered set hashes", "99813", "44201", "Read", "Admin", "Read", false},
		{"wrong test privilege", "0", "1", "Read", "Admin", "Write", true},
		{"matching privilege belongs to caller", "1", "0", "Read", "Admin", "Admin", true},
		{"caller no longer admin", "1", "0", "Read", "Write", "Read", true},
	}

	for _, kind := range []string{"service", "team", "user"} {
		for _, tt := range tests {
			t.Run(kind+"/"+tt.name, func(t *testing.T) {
				t.Parallel()

				s := terraform.NewState()
				s.RootModule().Resources[identityName] = &terraform.ResourceState{
					Primary: &terraform.InstanceState{Attributes: map[string]string{"slug": testSlug}},
				}
				s.RootModule().Resources[resourceName] = &terraform.ResourceState{
					Primary: &terraform.InstanceState{Attributes: map[string]string{
						kind + ".#":                                "2",
						kind + "." + tt.testIndex + ".slug":        testSlug,
						kind + "." + tt.testIndex + ".privilege":   tt.testPrivilege,
						kind + "." + tt.callerIndex + ".slug":      callerSlug,
						kind + "." + tt.callerIndex + ".privilege": tt.callerPrivilege,
					}},
				}
				check := resource.ComposeTestCheckFunc(
					testAccRepositoryPrivilegeForResource(resourceName, kind, identityName, tt.wantPrivilege),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, kind+".*", map[string]string{
						"slug": callerSlug, "privilege": "Admin",
					}),
				)
				if err := check(s); (err != nil) != tt.wantError {
					t.Fatalf("check error = %v, want error: %t", err, tt.wantError)
				}
			})
		}
	}
}

func TestRepositoryPrivilegeForResourceMissingState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		identity *terraform.ResourceState
		want     string
	}{
		{"missing identity", nil, "missing identity resource state"},
		{"missing primary", &terraform.ResourceState{}, "missing identity resource state"},
		{"missing slug", &terraform.ResourceState{Primary: &terraform.InstanceState{}}, "missing slug"},
		{
			"missing privilege resource",
			&terraform.ResourceState{Primary: &terraform.InstanceState{Attributes: map[string]string{"slug": "test-identity"}}},
			"Not found",
		},
	}
	check := testAccRepositoryPrivilegeForResource("cloudsmith_repository_privileges.test", "service", "cloudsmith_service.test", "Read")
	if err := check(nil); err == nil || !strings.Contains(err.Error(), "missing root module state") {
		t.Fatalf("check(nil) = %v, want missing root module state error", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := terraform.NewState()
			if tt.identity != nil {
				s.RootModule().Resources["cloudsmith_service.test"] = tt.identity
			}
			if err := check(s); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("check error = %v, want %q", err, tt.want)
			}
		})
	}
}
