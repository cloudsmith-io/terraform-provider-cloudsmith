package cloudsmith

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const testDefaultEntitlement = `{"name":"Default","default":true,"slug_perm":"token-permanent","identifier":42,"is_active":true}`

func testEntitlementPaginationHeaders(w http.ResponseWriter, count int) {
	w.Header().Set("X-Pagination-Count", fmt.Sprint(count))
	w.Header().Set("X-Pagination-Page", "1")
	w.Header().Set("X-Pagination-PageTotal", "1")
	w.Header().Set("X-Pagination-PageSize", "100")
}

func testEntitlementFixtureProvider(server *httptest.Server) *providerConfig {
	cfg := cloudsmith.NewConfiguration()
	cfg.Servers = cloudsmith.ServerConfigurations{{URL: server.URL}}
	cfg.HTTPClient = server.Client()
	return &providerConfig{
		APIClient: cloudsmith.NewAPIClient(cfg),
		Auth: context.WithValue(context.Background(), cloudsmith.ContextAPIKeys, map[string]cloudsmith.APIKey{
			"apikey": {Key: "fixture-key"},
		}),
	}
}

func testEntitlementListRequest(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Method != http.MethodGet || r.URL.Path != "/entitlements/fixture-org/repo-permanent/" {
		t.Errorf("unexpected list request: %s %s", r.Method, r.URL.Path)
	}
	if r.Header.Get("X-Api-Key") != "fixture-key" {
		t.Error("list request lost SDK authentication")
	}
	want := "active=false&page=1&page_size=100&query=name%3ADefault+&show_tokens=false"
	if r.URL.RawQuery != want {
		t.Errorf("query = %q, want %q", r.URL.RawQuery, want)
	}
	if body, err := io.ReadAll(r.Body); err != nil || len(body) != 0 {
		t.Errorf("list request body = %q, error = %v", body, err)
	}
}

func TestEntitlementControlFixtureReadiness(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		statuses []int
		bodies   []string
		wantErr  string
	}{
		{"ready", []int{200}, []string{"[" + testDefaultEntitlement + "]"}, ""},
		{"endpoint then token delayed", []int{404, 200, 200}, []string{`{"detail":"Not found."}`, "[]", "[" + testDefaultEntitlement + "]"}, ""},
		{"unauthorized", []int{401}, []string{`{"detail":"unauthorized"}`}, "401"},
		{"forbidden after 404", []int{404, 403}, []string{`{"detail":"Not found."}`, `{"detail":"forbidden"}`}, "403"},
		{"server failure", []int{500}, []string{`{"detail":"failure"}`}, "500"},
		{"malformed JSON", []int{200}, []string{"{"}, "unexpected end of JSON input"},
		{"wrong token", []int{200}, []string{`[{"name":"Default","slug_perm":"other"}]`}, "unexpected Default entitlement"},
		{"missing token identity", []int{200}, []string{`[{"name":"Default","default":true}]`}, "unexpected Default entitlement"},
		{"ambiguous tokens", []int{200}, []string{"[" + testDefaultEntitlement + "," + testDefaultEntitlement + "]"}, "unexpected Default entitlement"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				testEntitlementListRequest(t, r)
				i := int(calls.Add(1)) - 1
				if i >= len(tc.statuses) {
					t.Error("unexpected retry")
					i = len(tc.statuses) - 1
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statuses[i])
				fmt.Fprint(w, tc.bodies[i])
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			err := testAccWaitForDefaultEntitlement(ctx, testEntitlementFixtureProvider(server), "fixture-org", "repo-permanent", time.Millisecond)
			if tc.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("error = %v, want %q", err, tc.wantErr)
			}
			if got := int(calls.Load()); got != len(tc.statuses) {
				t.Fatalf("requests = %d, want %d", got, len(tc.statuses))
			}
		})
	}
}

func TestEntitlementControlFixtureCancellation(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"before request", "in request", "between requests", "deadline empty", "deadline 404"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var calls atomic.Int32
			started := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				close(started)
				if phase == "in request" {
					<-release
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				if phase == "between requests" {
					cancel()
				}
				w.Header().Set("Content-Type", "application/json")
				if phase == "deadline 404" {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprint(w, `{"detail":"Not found."}`)
				} else {
					fmt.Fprint(w, "[]")
				}
			}))
			defer server.Close()
			defer close(release)
			wantErr := error(context.Canceled)
			if strings.HasPrefix(phase, "deadline") {
				ctx, cancel = context.WithTimeout(t.Context(), 500*time.Millisecond)
				defer cancel()
				wantErr = context.DeadlineExceeded
			}
			if phase == "before request" {
				cancel()
			}
			result := make(chan error, 1)
			go func() {
				result <- testAccWaitForDefaultEntitlement(ctx, testEntitlementFixtureProvider(server), "fixture-org", "repo-permanent", time.Hour)
			}()
			if phase == "in request" {
				select {
				case <-started:
					cancel()
				case <-time.After(10 * time.Second):
					t.Fatal("request did not start")
				}
			}
			select {
			case err := <-result:
				if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "fixture-org/repo-permanent") {
					t.Fatalf("error = %v, want %v with fixture identity", err, wantErr)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("readiness ignored cancellation")
			}
			wantCalls := int32(1)
			if phase == "before request" {
				wantCalls = 0
			}
			if got := calls.Load(); got != wantCalls {
				t.Errorf("requests = %d, want %d", got, wantCalls)
			}
		})
	}
}

func TestEntitlementListReadPreservesErrorsAndEmptyResults(t *testing.T) {
	t.Parallel()
	for _, status := range []int{200, 401, 403, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				testEntitlementListRequest(t, r)
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				testEntitlementPaginationHeaders(w, 0)
				w.WriteHeader(status)
				if status == http.StatusOK {
					fmt.Fprint(w, "[]")
				} else {
					fmt.Fprint(w, `{"detail":"Not found or not permitted."}`)
				}
			}))
			defer server.Close()
			d := schema.TestResourceDataRaw(t, dataSourceEntitlementList().Schema, map[string]interface{}{
				"namespace": "fixture-org", "repository": "repo-permanent", "query": []interface{}{"name:Default"},
			})
			err := dataSourceEntitlementRead(d, testEntitlementFixtureProvider(server))
			if status == http.StatusOK {
				if err != nil || d.Id() == "" || len(d.Get("entitlement_tokens").([]interface{})) != 0 {
					t.Fatalf("legitimate empty list was not preserved: id=%q error=%v", d.Id(), err)
				}
			} else if err == nil || !strings.Contains(err.Error(), fmt.Sprint(status)) || d.Id() != "" {
				t.Fatalf("error/state not preserved: id=%q error=%v", d.Id(), err)
			}
			if calls.Load() != 1 {
				t.Fatal("production data source must not retry")
			}
		})
	}
}

func TestEntitlementControlFixtureCleanup(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		prepare        bool
		controlChecked bool
		status         int
		wantError      string
		wantReads      int32
	}{
		{"failed readiness cleaned up", true, false, 404, "", 1},
		{"failed readiness leaked repository", true, false, 200, "still exists", 1},
		{"cleanup forbidden", true, false, 403, "unable to verify", 1},
		{"cleanup server error", true, false, 500, "unable to verify", 1},
		{"unexpected missing state", false, false, 404, "resource not found", 0},
		{"missing control after creation check", true, true, 404, "resource not found", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var repositoryReads atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/entitlements/fixture-org/repo-permanent/" {
					testEntitlementListRequest(t, r)
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprint(w, `{"detail":"fixture readiness denied"}`)
					return
				}
				if r.URL.Path != "/repos/fixture-org/repo-permanent/" || r.Method != http.MethodGet {
					t.Errorf("unexpected cleanup request: %s %s", r.Method, r.URL.Path)
				}
				repositoryReads.Add(1)
				w.WriteHeader(tc.status)
				if tc.status == http.StatusOK {
					fmt.Fprint(w, `{"name":"fixture-repository","slug_perm":"repo-permanent"}`)
				} else {
					fmt.Fprint(w, `{"detail":"fixture cleanup response"}`)
				}
			}))
			defer server.Close()
			provider := Provider()
			provider.SetMeta(testEntitlementFixtureProvider(server))
			fixture := testAccEntitlementControlTestCase(t.Context(), "fixture-repository", provider)
			state := terraform.NewState()
			state.RootModule().Resources["cloudsmith_repository.test"] = &terraform.ResourceState{
				Primary: &terraform.InstanceState{ID: "repo-permanent", Attributes: map[string]string{
					"namespace": "fixture-org", "slug_perm": "repo-permanent", "name": "fixture-repository",
				}},
			}
			if tc.prepare {
				if err := fixture.Steps[0].Check(state); err == nil || !strings.Contains(err.Error(), "403") {
					t.Fatalf("expected readiness failure: %v", err)
				}
			}
			if tc.controlChecked {
				if err := fixture.Steps[1].Check(state); err == nil {
					t.Fatal("unexpected success checking missing control")
				}
			}
			err := fixture.CheckDestroy(state)
			if tc.wantError == "" && err != nil {
				t.Fatal(err)
			}
			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Fatalf("cleanup error = %v, want %q", err, tc.wantError)
			}
			if got := repositoryReads.Load(); got != tc.wantReads {
				t.Fatalf("cleanup reads = %d, want %d", got, tc.wantReads)
			}
		})
	}
}

func TestEntitlementControlFixtureRequiresOwnedRepository(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "must not request an unknown fixture", http.StatusInternalServerError)
	}))
	defer server.Close()
	for _, attrs := range []map[string]string{
		{},
		{"namespace": "fixture-org", "slug_perm": "another-repository", "name": "fixture-repository"},
		{"namespace": "fixture-org", "slug_perm": "repo-permanent", "name": "another-repository"},
	} {
		provider := Provider()
		provider.SetMeta(testEntitlementFixtureProvider(server))
		tc := testAccEntitlementControlTestCase(t.Context(), "fixture-repository", provider)
		state := terraform.NewState()
		state.RootModule().Resources["cloudsmith_repository.test"] = &terraform.ResourceState{
			Primary: &terraform.InstanceState{ID: "repo-permanent", Attributes: attrs},
		}
		if err := tc.Steps[0].Check(state); err == nil {
			t.Fatal("unexpectedly accepted invalid fixture identity")
		}
		if err := tc.CheckDestroy(state); err == nil {
			t.Fatal("unexpectedly ignored missing control state")
		}
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("invalid fixture made %d requests", got)
	}
}

func TestEntitlementControlFixtureAcceptanceGate(t *testing.T) {
	t.Setenv(resource.EnvTfAcc, "")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "acceptance must remain gated", http.StatusInternalServerError)
	}))
	defer server.Close()
	provider := Provider()
	provider.SetMeta(testEntitlementFixtureProvider(server))
	t.Run("acceptance disabled", func(t *testing.T) {
		tc := testAccEntitlementControlTestCase(t.Context(), "fixture-repository", provider)
		tc.PreCheck = func() { t.Fatal("PreCheck must remain behind TF_ACC") }
		resource.Test(t, tc)
	})
	if got := requests.Load(); got != 0 {
		t.Fatalf("fixture made %d requests with TF_ACC unset", got)
	}
}

// Run the same HCL through Terraform, not just resource callbacks. No real
// credentials or TF_ACC are required; use the SDK's Terraform discovery.
func TestEntitlementControlFixtureTerraformLifecycle(t *testing.T) {
	t.Setenv("CLOUDSMITH_NAMESPACE", "fixture-org")
	for _, initialStatus := range []int{http.StatusNotFound, http.StatusOK} {
		for _, readyPhase := range []bool{false, true} {
			t.Run(fmt.Sprintf("initial_%d/readiness_%t", initialStatus, readyPhase), func(t *testing.T) {
				var mu sync.Mutex
				var events []string
				var repo map[string]interface{}
				var created, deleted, active bool
				var lists int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					event := r.Method + " " + r.URL.Path
					events = append(events, event)
					switch event {
					case "POST /repos/fixture-org/":
						if created {
							t.Error("repository was created twice")
						}
						created, active = true, true
						if err := json.NewDecoder(r.Body).Decode(&repo); err != nil {
							t.Errorf("decode repository: %v", err)
						}
						if repo["name"] != "fixture-repository" {
							t.Errorf("repository create name = %v", repo["name"])
						}
						repo["slug"], repo["slug_perm"] = "human-readable-repository", "repo-permanent"
						w.WriteHeader(http.StatusCreated)
						if err := json.NewEncoder(w).Encode(repo); err != nil {
							t.Errorf("encode repository: %v", err)
						}
					case "GET /repos/fixture-org/repo-permanent/":
						if !created {
							t.Error("repository read before creation")
						}
						if deleted {
							w.WriteHeader(http.StatusNotFound)
							fmt.Fprint(w, `{"detail":"Not found."}`)
							return
						}
						if err := json.NewEncoder(w).Encode(repo); err != nil {
							t.Errorf("encode repository: %v", err)
						}
					case "GET /entitlements/fixture-org/repo-permanent/":
						testEntitlementListRequest(t, r)
						testEntitlementPaginationHeaders(w, 1)
						if !created || deleted {
							t.Error("list outside repository lifetime")
						}
						lists++
						// A repository 200 does not imply that its entitlement
						// endpoint, then default token, are already visible.
						if lists == 1 && initialStatus == http.StatusNotFound {
							w.WriteHeader(http.StatusNotFound)
							fmt.Fprint(w, `{"detail":"Not found."}`)
						} else if lists <= 2 {
							fmt.Fprint(w, "[]")
						} else {
							fmt.Fprintf(w, `[{"name":"Default","default":true,"slug_perm":"token-permanent","identifier":42,"is_active":%t}]`, active)
						}
					case "POST /entitlements/fixture-org/repo-permanent/token-permanent/enable/",
						"POST /entitlements/fixture-org/repo-permanent/token-permanent/disable/":
						if !created || deleted || lists < 3 {
							t.Error("control mutation before readiness or after repository deletion")
						}
						active = strings.HasSuffix(r.URL.Path, "/enable/")
						w.WriteHeader(http.StatusNoContent)
					case "GET /entitlements/fixture-org/repo-permanent/token-permanent/":
						if deleted {
							w.WriteHeader(http.StatusNotFound)
							fmt.Fprint(w, `{"detail":"Not found."}`)
						} else {
							fmt.Fprintf(w, `{"name":"Default","default":true,"slug_perm":"token-permanent","is_active":%t}`, active)
						}
					case "DELETE /repos/fixture-org/repo-permanent/":
						if readyPhase && active {
							t.Error("control was not disabled before deleting repository")
						}
						deleted = true
						w.WriteHeader(http.StatusNoContent)
					default:
						t.Errorf("unexpected endpoint or identifier: %s", event)
						http.Error(w, "unexpected request", http.StatusBadRequest)
					}
				}))
				defer server.Close()
				provider := Provider()
				pc := testEntitlementFixtureProvider(server)
				provider.ConfigureContextFunc = func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) { return pc, nil }
				tc := testAccEntitlementControlTestCase(t.Context(), "fixture-repository", provider)
				if !readyPhase {
					wantError := "404"
					if initialStatus == http.StatusOK {
						wantError = "Invalid index"
					}
					tc.Steps = []resource.TestStep{{
						Config: testAccEntitlementControlConfigBasic("fixture-repository"), ExpectError: regexp.MustCompile(wantError),
					}}
					tc.CheckDestroy = func(s *terraform.State) error {
						if _, ok := s.RootModule().Resources["cloudsmith_entitlement_control.test"]; ok {
							return fmt.Errorf("control unexpectedly created before default token readiness")
						}
						return nil
					}
				}

				resource.UnitTest(t, tc)
				mu.Lock()
				defer mu.Unlock()
				if !created || !deleted {
					t.Fatalf("repository cleanup: created=%t deleted=%t; requests=%v", created, deleted, events)
				}
				var mutations []string
				for _, event := range events {
					if strings.HasPrefix(event, "POST /entitlements/") {
						mutations = append(mutations, event)
					}
				}
				if readyPhase {
					want := []string{
						"POST /entitlements/fixture-org/repo-permanent/token-permanent/enable/",
						"POST /entitlements/fixture-org/repo-permanent/token-permanent/disable/",
						"POST /entitlements/fixture-org/repo-permanent/token-permanent/disable/",
					}
					if !slices.Equal(mutations, want) {
						t.Fatalf("control lifecycle = %v, want %v", mutations, want)
					}
				} else if lists != 1 || len(mutations) != 0 {
					t.Fatalf("baseline should fail first list before control creation: lists=%d mutations=%v", lists, mutations)
				}
				t.Logf("readiness=%t requests=%v", readyPhase, events)
			})
		}
	}
}
