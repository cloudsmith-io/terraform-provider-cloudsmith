//nolint:testpackage
package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cloudsmith "github.com/cloudsmith-io/cloudsmith-api-go"
)

func testPrivilegesProviderConfig(server *httptest.Server) *providerConfig {
	config := cloudsmith.NewConfiguration()
	config.Servers = cloudsmith.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	return &providerConfig{
		APIClient: cloudsmith.NewAPIClient(config),
		Auth: context.WithValue(
			context.Background(),
			cloudsmith.ContextAPIKeys,
			map[string]cloudsmith.APIKey{
				"apikey": {Key: "test-api-key"},
			},
		),
	}
}

// These cover the team-only fallback in authenticatedAccountAdminPrivilege:
// the authenticated account has no direct (service/user) grant in the
// existing privileges, only team-based access, so the function must resolve
// the account's kind via the API and construct a new direct grant.
func TestAuthenticatedAccountAdminPrivilege_TeamOnlyFallback(t *testing.T) {
	t.Parallel()

	const (
		organization = "example-org"
		accountSlug  = "provisioning-sa"
	)

	teamOnlyPrivileges := []cloudsmith.RepositoryPrivilegeDict{func() cloudsmith.RepositoryPrivilegeDict {
		p := cloudsmith.RepositoryPrivilegeDict{}
		p.SetTeam("some-team")
		p.SetPrivilege("Admin")
		return p
	}()}

	t.Run("resolves to a service grant", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == fmt.Sprintf("/orgs/%s/services/%s/", organization, accountSlug):
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"name":"provisioning service","slug":%q,"slug_perm":%q}`, accountSlug, accountSlug)
			default:
				http.Error(w, "unexpected request", http.StatusNotFound)
			}
		}))
		defer server.Close()

		pc := testPrivilegesProviderConfig(server)
		got, err := authenticatedAccountAdminPrivilege(pc, organization, accountSlug, teamOnlyPrivileges)
		if err != nil {
			t.Fatalf("authenticatedAccountAdminPrivilege() error = %v", err)
		}
		if !got.HasService() || got.GetService() != accountSlug {
			t.Fatalf("expected service grant for %q, got %+v", accountSlug, got)
		}
		if got.GetPrivilege() != "Admin" {
			t.Fatalf("expected Admin privilege, got %q", got.GetPrivilege())
		}
	})

	t.Run("resolves to a user grant", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == fmt.Sprintf("/orgs/%s/services/%s/", organization, accountSlug):
				http.Error(w, "not found", http.StatusNotFound)
			case r.URL.Path == fmt.Sprintf("/orgs/%s/members/%s/", organization, accountSlug):
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"user":%q}`, accountSlug)
			default:
				http.Error(w, "unexpected request", http.StatusNotFound)
			}
		}))
		defer server.Close()

		pc := testPrivilegesProviderConfig(server)
		got, err := authenticatedAccountAdminPrivilege(pc, organization, accountSlug, teamOnlyPrivileges)
		if err != nil {
			t.Fatalf("authenticatedAccountAdminPrivilege() error = %v", err)
		}
		if !got.HasUser() || got.GetUser() != accountSlug {
			t.Fatalf("expected user grant for %q, got %+v", accountSlug, got)
		}
		if got.GetPrivilege() != "Admin" {
			t.Fatalf("expected Admin privilege, got %q", got.GetPrivilege())
		}
	})

	t.Run("errors when the account is neither a service nor a member", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}))
		defer server.Close()

		pc := testPrivilegesProviderConfig(server)
		_, err := authenticatedAccountAdminPrivilege(pc, organization, accountSlug, teamOnlyPrivileges)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "neither a service nor an organization member") {
			t.Fatalf("unexpected error %q", err)
		}
	})

	t.Run("errors when service resolution fails", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))
		defer server.Close()

		pc := testPrivilegesProviderConfig(server)
		_, err := authenticatedAccountAdminPrivilege(pc, organization, accountSlug, teamOnlyPrivileges)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "error determining whether authenticated account is a service") {
			t.Fatalf("unexpected error %q", err)
		}
	})

	t.Run("errors when member resolution fails", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == fmt.Sprintf("/orgs/%s/services/%s/", organization, accountSlug):
				http.Error(w, "not found", http.StatusNotFound)
			case r.URL.Path == fmt.Sprintf("/orgs/%s/members/%s/", organization, accountSlug):
				http.Error(w, "internal error", http.StatusInternalServerError)
			default:
				http.Error(w, "unexpected request", http.StatusNotFound)
			}
		}))
		defer server.Close()

		pc := testPrivilegesProviderConfig(server)
		_, err := authenticatedAccountAdminPrivilege(pc, organization, accountSlug, teamOnlyPrivileges)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "error determining whether authenticated account is an organization member") {
			t.Fatalf("unexpected error %q", err)
		}
	})
}
