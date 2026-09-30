// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package cloudsmith

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	cloudsmithv2 "github.com/cloudsmith-io/cloudsmith-go-v2"
	v2retry "github.com/cloudsmith-io/cloudsmith-go-v2/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testPolicyActionConfig(kind string, updated bool) map[string]interface{} {
	block := map[string]interface{}{"package_state": "QUARANTINED"}
	if updated {
		block["package_state"] = "DELETED"
	}
	if kind != actionSetPackageState {
		block = map[string]interface{}{"tags": []interface{}{"needs-review", "imported"}}
		if updated {
			block["tags"] = []interface{}{"needs-review"}
		}
	}
	return map[string]interface{}{
		"workspace": "workspace", "policy_slug_perm": "policy", "precedence": 10,
		kind: []interface{}{block},
	}
}

func testPolicyActionResponse(kind string, updated bool) map[string]interface{} {
	config := testPolicyActionConfig(kind, updated)
	actionType := map[string]string{
		actionSetPackageState: "SetPackageState", actionAddPackageTags: "AddPackageTags", actionRemovePackageTags: "RemovePackageTags",
	}[kind]
	response := map[string]interface{}{
		"action_type": actionType, "effect": kind, "slug_perm": "action", "precedence": 10,
		"created_at": "2026-09-29T12:00:00.123456Z", "updated_at": "2026-09-29T12:00:00.123456Z",
	}
	if updated {
		response["updated_at"] = "2026-09-29T12:00:01.654321Z"
	}
	for k, v := range config[kind].([]interface{})[0].(map[string]interface{}) {
		response[k] = v
	}
	return response
}

func testPolicyActionProvider(server *httptest.Server) *providerConfig {
	return &providerConfig{V2ApiClient: cloudsmithv2.New(
		cloudsmithv2.WithServerURL(server.URL),
		cloudsmithv2.WithRetryConfig(v2retry.Config{Strategy: "none"}),
	)}
}

func testPolicyActionData(t *testing.T, res *schema.Resource, kind, revision string) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, res.Schema, testPolicyActionConfig(kind, false))
	d.SetId("action")
	for key, value := range map[string]string{
		"slug_perm": "action", "created_at": "2026-09-29T12:00:00Z", "updated_at": revision,
	} {
		if err := d.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func TestPolicyActionUpdateLifecycle(t *testing.T) {
	t.Parallel()
	for _, kind := range actionTypeBlocks {
		for _, scenario := range []string{
			"stale apply", "stale refresh", "same second refresh", "newer external drift",
			"same second external drift", "equal revision external drift",
		} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				t.Parallel()
				previous := testPolicyActionResponse(kind, false)
				updated := testPolicyActionResponse(kind, true)
				if strings.HasPrefix(scenario, "same second") {
					updated["updated_at"] = "2026-09-29T12:00:00.654321Z"
				}
				fresh := updated
				wantDrift := strings.HasSuffix(scenario, "external drift")
				if wantDrift {
					fresh = testPolicyActionResponse(kind, false)
					fresh["updated_at"] = "2026-09-29T12:00:02.123456Z"
					fresh["precedence"] = 20
					if scenario == "same second external drift" {
						fresh["updated_at"] = "2026-09-29T12:00:00.900000Z"
					}
					if scenario == "equal revision external drift" {
						fresh["updated_at"] = updated["updated_at"]
					}
				}
				var mu sync.Mutex
				var refreshing bool
				var writes, reads int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path != "/v2/workspaces/workspace/policies/policy/actions/action/" {
						t.Errorf("unexpected path: %s", r.URL.Path)
						http.Error(w, "unexpected path", http.StatusBadRequest)
						return
					}
					response := updated
					switch r.Method {
					case http.MethodPut:
						writes++
						var body map[string]interface{}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Errorf("decode PUT: %v", err)
						}
						for key, want := range testPolicyActionConfig(kind, true)[kind].([]interface{})[0].(map[string]interface{}) {
							if fmt.Sprint(body[key]) != fmt.Sprint(want) {
								t.Errorf("PUT %s = %v, want %v", key, body[key], want)
							}
						}
					case http.MethodGet:
						if refreshing {
							response = fresh
							if reads == 0 {
								response = previous
							}
							reads++
						} else if scenario == "stale apply" {
							response = previous
						}
					default:
						t.Errorf("unexpected method: %s", r.Method)
						http.Error(w, "unexpected method", http.StatusBadRequest)
						return
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Errorf("encode action: %v", err)
					}
				}))
				defer server.Close()

				res := resourcePolicyAction()
				pc := testPolicyActionProvider(server)
				// Exercise migration from existing state that rounded API revisions to seconds.
				d := testPolicyActionData(t, res, kind, "2026-09-29T12:00:00Z")
				config := terraform.NewResourceConfigRaw(testPolicyActionConfig(kind, true))
				diff, err := res.Diff(context.Background(), d.State(), config, pc)
				if err != nil {
					t.Fatal(err)
				}
				if diff.Empty() || diff.RequiresNew() {
					t.Fatal("expected an in-place update")
				}
				state, diagnostics := res.Apply(context.Background(), d.State(), diff, pc)
				if diagnostics.HasError() {
					t.Fatalf("apply failed: %v", diagnostics)
				}
				checkPlan := func(state *terraform.InstanceState, phase string, wantDrift bool) {
					t.Helper()
					diff, err := res.Diff(context.Background(), state, config, pc)
					if err != nil {
						t.Fatal(err)
					}
					if diff.Empty() == wantDrift {
						t.Errorf("%s plan: %#v; want drift=%t", phase, diff.Attributes, wantDrift)
					}
				}
				checkPlan(state, "post-apply", false)
				if got := state.Attributes["updated_at"]; got != stringToTime(updated["updated_at"].(string)).Format(time.RFC3339Nano) {
					t.Errorf("applied revision = %s, want %s", got, updated["updated_at"])
				}
				mu.Lock()
				refreshing = true
				mu.Unlock()
				state, diagnostics = res.RefreshWithoutUpgrade(context.Background(), state, pc)
				if diagnostics.HasError() {
					t.Fatalf("refresh failed: %v", diagnostics)
				}
				checkPlan(state, "post-refresh", wantDrift)
				if got := state.Attributes["updated_at"]; got != stringToTime(fresh["updated_at"].(string)).Format(time.RFC3339Nano) {
					t.Errorf("refreshed revision = %s, want %s", got, fresh["updated_at"])
				}
				if wantDrift && state.Attributes["precedence"] != "20" {
					t.Error("refresh did not record newer external drift")
				}
				mu.Lock()
				defer mu.Unlock()
				if writes != 1 || reads != 2 {
					t.Errorf("writes=%d refresh reads=%d, want 1 and 2", writes, reads)
				}
			})
		}
	}
}

func TestPolicyActionReadErrors(t *testing.T) {
	// Like the upstream waiter tests, shorten shared defaults only in serial tests.
	oldTimeout, oldInterval := defaultUpdateTimeout, defaultUpdateInterval
	defaultUpdateInterval = time.Millisecond
	t.Cleanup(func() { defaultUpdateTimeout, defaultUpdateInterval = oldTimeout, oldInterval })

	for _, tc := range []struct {
		name       string
		status     int
		body       string
		wantError  string
		staleFirst bool
		block      bool
	}{
		{name: "not found", status: http.StatusNotFound},
		{name: "not found after stale", status: http.StatusNotFound, staleFirst: true},
		{name: "forbidden", status: http.StatusForbidden, body: `{"detail":"forbidden"}`, wantError: "forbidden"},
		{name: "forbidden after stale", status: http.StatusForbidden, body: `{"detail":"forbidden"}`, wantError: "forbidden", staleFirst: true},
		{name: "throttled", status: http.StatusTooManyRequests, body: `{"detail":"throttled"}`, wantError: "throttled", staleFirst: true},
		{name: "server error", status: http.StatusInternalServerError, body: `{"detail":"server failed"}`, wantError: "server failed", staleFirst: true},
		{name: "empty body", status: http.StatusOK, wantError: "unexpected end of JSON input", staleFirst: true},
		{name: "null body", status: http.StatusOK, body: "null", wantError: "unsupported policy action", staleFirst: true},
		{name: "malformed body", status: http.StatusOK, body: "{", wantError: "unexpected end of JSON input"},
		{name: "unknown variant", status: http.StatusOK, body: `{"action_type":"FutureAction"}`, wantError: "unsupported policy action"},
		{name: "perpetually stale", wantError: "last observed revision"},
		{name: "request deadline", block: true, wantError: "last observed revision"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaultUpdateTimeout = 5 * time.Second
			if tc.status == 0 {
				defaultUpdateTimeout = 30 * time.Millisecond
			}
			var mu sync.Mutex
			var reads int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				reads++
				stale := tc.status == 0 || (tc.staleFirst && reads == 1)
				mu.Unlock()
				if tc.block {
					<-r.Context().Done()
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if stale {
					if err := json.NewEncoder(w).Encode(testPolicyActionResponse(actionSetPackageState, false)); err != nil {
						t.Errorf("encode stale response: %v", err)
					}
					return
				}
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			d := testPolicyActionData(t, resourcePolicyAction(), actionSetPackageState, "2026-09-29T12:00:01.654321Z")
			start := time.Now()
			diagnostics := resourcePolicyActionRead(context.Background(), d, testPolicyActionProvider(server))
			if tc.status == 0 && time.Since(start) > time.Second {
				t.Error("read exceeded bounded timeout")
			}
			if tc.wantError == "" {
				if diagnostics.HasError() || d.Id() != "" {
					t.Fatalf("missing action: diagnostics=%v ID=%q", diagnostics, d.Id())
				}
			} else {
				if !diagnostics.HasError() || !strings.Contains(fmt.Sprint(diagnostics), tc.wantError) {
					t.Errorf("diagnostics = %v, want %q", diagnostics, tc.wantError)
				}
				if tc.status == 0 && !strings.Contains(fmt.Sprint(diagnostics), "timeout") && !strings.Contains(fmt.Sprint(diagnostics), "deadline") {
					t.Errorf("expected timeout/deadline, got %v", diagnostics)
				}
				if d.Id() != "action" || d.Get("updated_at") != "2026-09-29T12:00:01.654321Z" {
					t.Error("failed read changed identity or flattened a stale snapshot")
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if tc.status != 0 {
				wantReads := 1
				if tc.staleFirst {
					wantReads++
				}
				if reads != wantReads {
					t.Errorf("reads=%d, want %d; terminal responses must not be retried", reads, wantReads)
				}
			}
		})
	}
}

func TestPolicyActionUpdateErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "not found", status: http.StatusNotFound, body: `{"detail":"not found"}`},
		{name: "forbidden", status: http.StatusForbidden, body: `{"detail":"forbidden"}`},
		{name: "throttled", status: http.StatusTooManyRequests, body: `{"detail":"throttled"}`},
		{name: "server error", status: http.StatusInternalServerError, body: `{"detail":"failed"}`},
		{name: "empty body", status: http.StatusOK},
		{name: "null body", status: http.StatusOK, body: "null"},
		{name: "malformed body", status: http.StatusOK, body: "{"},
		{name: "unknown variant", status: http.StatusOK, body: `{"action_type":"FutureAction"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("unexpected %s: failed update must not read", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			d := testPolicyActionData(t, resourcePolicyAction(), actionSetPackageState, "2026-09-29T12:00:00.123456Z")
			diagnostics := resourcePolicyActionUpdate(context.Background(), d, testPolicyActionProvider(server))
			if !diagnostics.HasError() {
				t.Fatal("expected update error")
			}
			if d.Id() != "action" || d.Get("updated_at") != "2026-09-29T12:00:00.123456Z" {
				t.Error("failed update changed identity or state")
			}
		})
	}
}

func TestPolicyActionCancellation(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"read", "update"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				cancel()
				<-release
			}))
			defer server.Close()
			defer close(release)
			d := testPolicyActionData(t, resourcePolicyAction(), actionSetPackageState, "2026-09-29T12:00:00.123456Z")
			callback := resourcePolicyActionRead
			if operation == "update" {
				callback = resourcePolicyActionUpdate
			}
			start := time.Now()
			diagnostics := callback(ctx, d, testPolicyActionProvider(server))
			if !diagnostics.HasError() || !strings.Contains(fmt.Sprint(diagnostics), "context canceled") {
				t.Errorf("expected cancellation, got %v", diagnostics)
			}
			if time.Since(start) > time.Second {
				t.Error("cancellation did not promptly stop the request")
			}
			if d.Id() != "action" || d.Get("updated_at") != "2026-09-29T12:00:00.123456Z" {
				t.Error("canceled operation changed identity or state")
			}
		})
	}
}

func TestPolicyActionReadWithoutRevision(t *testing.T) {
	t.Parallel()
	for _, revision := range []string{"", "2026-09-29T12:00:00Z"} {
		t.Run(revision, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(testPolicyActionResponse(actionSetPackageState, true)); err != nil {
					t.Errorf("encode action: %v", err)
				}
			}))
			defer server.Close()
			res := resourcePolicyAction()
			d := testPolicyActionData(t, res, actionSetPackageState, revision)
			state, diagnostics := res.RefreshWithoutUpgrade(context.Background(), d.State(), testPolicyActionProvider(server))
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if state.Attributes["set_package_state.0.package_state"] != "DELETED" {
				t.Error("import/legacy refresh failed to record external drift")
			}
		})
	}
}
