// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package cloudsmith

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	cloudsmithv2 "github.com/cloudsmith-io/cloudsmith-go-v2"
	"github.com/cloudsmith-io/cloudsmith-go-v2/models/components"
	"github.com/cloudsmith-io/cloudsmith-go-v2/models/operations"
	v2retry "github.com/cloudsmith-io/cloudsmith-go-v2/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const policyListFixtureWorkspace = "fixture-workspace"

type policyListFixtureAPI struct {
	mu         sync.Mutex
	policy     components.Policy
	exists     bool
	visible    bool
	lists      int
	creates    int
	deletes    int
	delayLists int
	listStatus int
}

func newPolicyListFixtureAPI(t *testing.T, name, sort string) (*policyListFixtureAPI, *schema.Provider) {
	t.Helper()
	if sort == "" {
		sort = "-created_at"
	}
	api := &policyListFixtureAPI{policy: components.Policy{
		Name: name, SlugPerm: "owned-policy", Version: 1,
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		defer api.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimSuffix(r.URL.Path, "/")
		base := "/v2/workspaces/" + policyListFixtureWorkspace + "/policies"
		var response interface{}
		switch {
		case r.Method == http.MethodPost && path == base:
			var body components.PolicyRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode policy: %v", err)
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			if body.Name != name || api.exists {
				t.Errorf("unexpected seed creation: name=%q exists=%t", body.Name, api.exists)
			}
			api.policy.Rego = body.Rego
			api.policy.Enabled = body.Enabled
			api.policy.IsTerminal = body.IsTerminal
			api.policy.Precedence = body.Precedence
			api.exists = true
			api.creates++
			w.WriteHeader(http.StatusCreated)
			response = api.policy
		case r.Method == http.MethodGet && path == base+"/owned-policy":
			if !api.exists {
				http.Error(w, `{"detail":"Not found"}`, http.StatusNotFound)
				return
			}
			response = api.policy
		case r.Method == http.MethodGet && path == base:
			if !api.exists {
				t.Error("list read before seed creation")
			}
			if got := r.URL.Query().Get("query"); got != fmt.Sprintf("name:%q", name) {
				t.Errorf("query=%q, want exact owned seed query", got)
			}
			if got := r.URL.Query().Get("sort"); got != sort {
				t.Errorf("sort=%q, want %q", got, sort)
			}
			api.lists++
			if api.listStatus != 0 {
				http.Error(w, `{"detail":"fixture list failed"}`, api.listStatus)
				return
			}
			if api.delayLists > 0 && api.lists > api.delayLists {
				api.visible = true
			}
			results := []components.Policy{}
			if api.visible {
				results = append(results, api.policy)
			}
			w.Header().Set("X-Pagination-Count", fmt.Sprint(len(results)))
			w.Header().Set("X-Pagination-Page", "1")
			w.Header().Set("X-Pagination-PageTotal", "1")
			w.Header().Set("X-Pagination-PageSize", "100")
			response = map[string]interface{}{"results": results, "pagetotal": 1}
		case r.Method == http.MethodDelete && path == base+"/owned-policy":
			api.exists = false
			api.deletes++
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode policy response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		api.mu.Lock()
		defer api.mu.Unlock()
		if api.exists || api.creates != 1 || api.deletes != 1 {
			t.Errorf("incomplete owned cleanup: exists=%t creates=%d deletes=%d", api.exists, api.creates, api.deletes)
		}
	})
	pc := &providerConfig{V2ApiClient: cloudsmithv2.New(
		cloudsmithv2.WithServerURL(server.URL),
		cloudsmithv2.WithRetryConfig(v2retry.Config{Strategy: "none"}),
	)}
	provider := Provider()
	provider.ConfigureContextFunc = func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return pc, nil
	}
	return api, provider
}

func TestPolicyListFixtureFrozenState(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, address, sort string
		config              func(string, string) string
	}{
		{"basic", "data.cloudsmith_policy_list.all", "", testAccPolicyListDataSourceConfigBasic},
		{"filter", "data.cloudsmith_policy_list.filtered", "-created_at", testAccPolicyListDataSourceConfigFilter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			name := "Fixture Seed " + tc.name
			api, provider := newPolicyListFixtureAPI(t, name, tc.sort)
			resource.UnitTest(t, resource.TestCase{
				Providers: map[string]*schema.Provider{"cloudsmith": provider},
				Steps: []resource.TestStep{{
					Config: tc.config(name, policyListFixtureWorkspace),
					Check: func(s *terraform.State) error {
						api.mu.Lock()
						api.visible = true
						api.mu.Unlock()
						query := fmt.Sprintf("name:%q", name)
						ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
						defer cancel()
						pc := provider.Meta().(*providerConfig)
						var sort *string
						if tc.sort != "" {
							sort = &tc.sort
						}
						resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesList(ctx, operations.WorkspacesPoliciesListRequest{
							Workspace: policyListFixtureWorkspace, Query: &query, Sort: sort, Page: 1,
						})
						if err != nil {
							return err
						}
						if resp.PaginatedPolicyList == nil || len(resp.PaginatedPolicyList.Results) != 1 {
							return fmt.Errorf("mock seed did not become visible")
						}
						t.Log("live API sees owned seed; retrying the Terraform snapshot still sees zero")
						api.mu.Lock()
						lists := api.lists
						api.mu.Unlock()
						checks := 0
						check := resource.TestCheckResourceAttr(tc.address, "policies.#", "1")
						err = testAccRetry(100*time.Millisecond, time.Millisecond, func(snapshot *terraform.State) error {
							checks++
							return check(snapshot)
						})(s)
						if err == nil || !strings.Contains(err.Error(), `expected "1", got "0"`) || checks < 2 {
							return fmt.Errorf("expected frozen-state failure after multiple checks, got %v (%d checks)", err, checks)
						}
						api.mu.Lock()
						defer api.mu.Unlock()
						if api.lists != lists {
							return fmt.Errorf("state retry unexpectedly refreshed the live API")
						}
						return nil
					},
				}},
			})
		})
	}
}

func TestPolicyListFixtureDelayedVisibility(t *testing.T) {
	t.Parallel()
	for _, filtered := range []bool{false, true} {
		t.Run(fmt.Sprintf("filtered=%t", filtered), func(t *testing.T) {
			t.Parallel()
			name, sort := "Fixture Delayed Seed", ""
			if filtered {
				sort = "-created_at"
			}
			api, provider := newPolicyListFixtureAPI(t, name, sort)
			api.delayLists = 2
			steps := testAccPolicyListDataSourceSteps(t.Context(), name, policyListFixtureWorkspace, filtered, provider)
			ready := false
			check := steps[0].Check
			steps[0].Check = func(s *terraform.State) error {
				if len(s.RootModule().Resources) != 1 {
					return fmt.Errorf("seed step must not contain a data source")
				}
				if err := check(s); err != nil {
					return err
				}
				api.mu.Lock()
				defer api.mu.Unlock()
				if api.lists != 3 {
					return fmt.Errorf("expected two empty live queries then visibility, got %d queries", api.lists)
				}
				ready = true
				return nil
			}
			dataSource := provider.DataSourcesMap["cloudsmith_policy_list"]
			read := dataSource.ReadContext
			reads := 0
			dataSource.ReadContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
				api.mu.Lock()
				if !ready {
					t.Error("Terraform read the data source before the query visibility barrier")
				}
				reads++
				api.mu.Unlock()
				return read(ctx, d, meta)
			}
			resource.UnitTest(t, resource.TestCase{
				Providers: map[string]*schema.Provider{"cloudsmith": provider},
				Steps:     steps,
			})
			api.mu.Lock()
			defer api.mu.Unlock()
			if reads == 0 {
				t.Error("Terraform never read the data source")
			}
		})
	}
}

func TestPolicyListFixtureBarrierFailureCleanup(t *testing.T) {
	t.Parallel()
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprintf("canceled=%t", canceled), func(t *testing.T) {
			t.Parallel()
			const name = "Fixture Failure Seed"
			api, provider := newPolicyListFixtureAPI(t, name, "")
			api.listStatus = http.StatusForbidden
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			steps := testAccPolicyListDataSourceSteps(ctx, name, policyListFixtureWorkspace, false, provider)
			check := steps[0].Check
			steps[0].Check = func(s *terraform.State) error {
				if canceled {
					cancel()
				}
				return check(s)
			}
			// Only exercise the failing seed step; ErrorCheck records the expected
			// failure while the SDK still runs its real Terraform destroy lifecycle.
			var failure error
			resource.UnitTest(t, resource.TestCase{
				Providers: map[string]*schema.Provider{"cloudsmith": provider},
				Steps:     steps[:1],
				ErrorCheck: func(err error) error {
					failure = err
					if !strings.Contains(err.Error(), "owned policy") {
						return err
					}
					return nil
				},
			})
			if failure == nil {
				t.Fatal("expected visibility barrier failure")
			}
			api.mu.Lock()
			defer api.mu.Unlock()
			if canceled {
				if !strings.Contains(failure.Error(), context.Canceled.Error()) || api.lists != 0 {
					t.Errorf("canceled barrier: failure=%v lists=%d", failure, api.lists)
				}
			} else if !strings.Contains(failure.Error(), "fixture list failed") || api.lists != 1 {
				t.Errorf("API failure was not returned immediately: failure=%v lists=%d", failure, api.lists)
			}
		})
	}
}

func policyListFixtureState(name string) *terraform.State {
	s := terraform.NewState()
	s.RootModule().Resources["cloudsmith_policy.seed"] = &terraform.ResourceState{
		Primary: &terraform.InstanceState{
			ID:         "owned-policy",
			Attributes: map[string]string{"workspace": policyListFixtureWorkspace, "name": name},
		},
	}
	return s
}

func TestPolicyListSeedVisible(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                    string
		status                  int
		body                    string
		cancelInFlight, timeout bool
		wantError               string
	}{
		{name: "empty then visible", body: `{"results":[]}`},
		{name: "wrong identity then visible", body: `{"results":[{"name":"Fixture Seed","slug_perm":"unowned-policy"}]}`},
		{name: "wrong name then visible", body: `{"results":[{"name":"other","slug_perm":"owned-policy"}]}`},
		{name: "multiple then visible", body: `{"results":[{"name":"Fixture Seed","slug_perm":"owned-policy"},{"name":"Fixture Seed","slug_perm":"other"}]}`},
		{name: "forbidden", status: http.StatusForbidden, body: `{"detail":"forbidden"}`, wantError: "forbidden"},
		{name: "server error", status: http.StatusInternalServerError, body: `{"detail":"server failure"}`, wantError: "server failure"},
		{name: "malformed response", body: `{`, wantError: "querying owned policy"},
		{name: "missing response", status: http.StatusNoContent, wantError: "querying owned policy"},
		{name: "canceled in flight", cancelInFlight: true, wantError: context.Canceled.Error()},
		{name: "timeout", timeout: true, body: `{"results":[]}`, wantError: context.DeadlineExceeded.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			release := make(chan struct{})
			var mu sync.Mutex
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				requests++
				request := requests
				mu.Unlock()
				if r.Method != http.MethodGet || r.URL.Path != "/v2/workspaces/"+policyListFixtureWorkspace+"/policies/" ||
					r.URL.Query().Get("query") != `name:"Fixture Seed"` || r.URL.Query().Get("sort") != "-created_at" ||
					r.URL.Query().Get("page") != "1" || r.URL.Query().Get("page_size") != fmt.Sprint(DefaultPageSize) {
					t.Errorf("unexpected exact visibility query: %s %s", r.Method, r.URL)
				}
				if tc.cancelInFlight {
					cancel()
					// Do not return an implicit 200 before cancellation reaches the
					// client, or wait on a server request context to unblock teardown.
					<-release
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if tc.status != 0 {
					w.WriteHeader(tc.status)
				}
				body := tc.body
				if request > 1 && tc.wantError == "" {
					body = `{"results":[{"name":"Fixture Seed","slug_perm":"owned-policy"}]}`
				}
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			defer close(release)
			pc := &providerConfig{V2ApiClient: cloudsmithv2.New(
				cloudsmithv2.WithServerURL(server.URL),
				cloudsmithv2.WithRetryConfig(v2retry.Config{Strategy: "none"}),
			)}
			interval := time.Millisecond
			if tc.timeout {
				var timeoutCancel context.CancelFunc
				ctx, timeoutCancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer timeoutCancel()
				interval = time.Second
			}
			err := testAccPolicyListSeedVisible(ctx, pc, policyListFixtureState("Fixture Seed"), policyListFixtureWorkspace, "Fixture Seed", "-created_at", interval)
			if tc.wantError == "" && err != nil {
				t.Fatal(err)
			}
			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Fatalf("error=%v, want %q", err, tc.wantError)
			}
			if tc.cancelInFlight && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation cause lost: %v", err)
			}
			if tc.timeout && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("deadline cause lost: %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if tc.wantError == "" && requests != 2 || tc.wantError != "" && !tc.timeout && requests != 1 {
				t.Errorf("unexpected query attempts: %d", requests)
			}
		})
	}
}

func TestPolicyListSeedVisibleRejectsUnownedState(t *testing.T) {
	t.Parallel()
	for _, field := range []string{"missing", "primary", "id", "workspace", "name"} {
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			s := policyListFixtureState("Fixture Seed")
			seed := s.RootModule().Resources["cloudsmith_policy.seed"]
			switch field {
			case "missing":
				delete(s.RootModule().Resources, "cloudsmith_policy.seed")
			case "primary":
				seed.Primary = nil
			case "id":
				seed.Primary.ID = ""
			default:
				seed.Primary.Attributes[field] = "unexpected"
			}
			if err := testAccPolicyListSeedVisible(t.Context(), nil, s, policyListFixtureWorkspace, "Fixture Seed", "", time.Millisecond); err == nil {
				t.Fatal("expected rejection before any API access")
			}
		})
	}
}
