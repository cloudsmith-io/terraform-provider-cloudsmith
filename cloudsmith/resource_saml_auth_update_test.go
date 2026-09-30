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
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testSAMLAuthConfig(inline, enabled, enforced bool) map[string]interface{} {
	config := map[string]interface{}{
		"organization": "example-org", "saml_auth_enabled": enabled, "saml_auth_enforced": enforced,
	}
	if inline {
		config["saml_metadata_inline"] = testSAMLMetadata
	} else {
		config["saml_metadata_url"] = "https://test.idp.example.com/metadata.xml"
	}
	return config
}

func testSAMLAuthResponse(config map[string]interface{}) *cloudsmith.OrganizationSAMLAuth {
	response := cloudsmith.NewOrganizationSAMLAuth(config["saml_auth_enabled"].(bool), config["saml_auth_enforced"].(bool))
	response.SetSamlMetadataInline("")
	response.SetSamlMetadataUrl("")
	if inline, ok := config["saml_metadata_inline"].(string); ok {
		response.SetSamlMetadataInline(inline)
	}
	if url, ok := config["saml_metadata_url"].(string); ok {
		response.SetSamlMetadataUrl(url)
	}
	return response
}

func testSAMLAuthProvider(server *httptest.Server) *providerConfig {
	config := cloudsmith.NewConfiguration()
	config.Servers = cloudsmith.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()
	config.HTTPClient.Timeout = 5 * time.Second
	return &providerConfig{
		APIClient: cloudsmith.NewAPIClient(config),
		Auth: context.WithValue(context.Background(), cloudsmith.ContextAPIKeys,
			map[string]cloudsmith.APIKey{"apikey": {Key: "test-api-key"}}),
	}
}

func TestSAMLAuthApplyRefreshPlan(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		previous map[string]interface{}
		desired  map[string]interface{}
	}{
		{name: "create URL", desired: testSAMLAuthConfig(false, true, false)},
		{name: "create inline", desired: testSAMLAuthConfig(true, true, false)},
		{name: "URL to inline", previous: testSAMLAuthConfig(false, true, false), desired: testSAMLAuthConfig(true, true, false)},
		{name: "inline to URL", previous: testSAMLAuthConfig(true, true, false), desired: testSAMLAuthConfig(false, true, false)},
		{name: "enforce", previous: testSAMLAuthConfig(true, true, false), desired: testSAMLAuthConfig(true, true, true)},
		{name: "disable", previous: testSAMLAuthConfig(false, true, true), desired: testSAMLAuthConfig(false, false, false)},
	} {
		for _, refresh := range []string{"consistent", "different remote state"} {
			t.Run(tc.name+"/"+refresh, func(t *testing.T) {
				t.Parallel()
				previous := tc.previous
				if previous == nil {
					previous = map[string]interface{}{"saml_auth_enabled": false, "saml_auth_enforced": false}
				}
				stale := testSAMLAuthResponse(previous)
				confirmed := testSAMLAuthResponse(tc.desired)
				var mu sync.Mutex
				var refreshing bool
				var writes, applyReads, refreshReads int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					if r.URL.Path != "/orgs/example-org/saml-authentication" || r.Header.Get("X-Api-Key") != "test-api-key" {
						t.Errorf("unexpected request: %s or missing test authentication", r.URL.Path)
						http.Error(w, "unexpected request", http.StatusBadRequest)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					var response *cloudsmith.OrganizationSAMLAuth
					switch r.Method {
					case http.MethodPatch:
						writes++
						var body map[string]interface{}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Errorf("decode PATCH: %v", err)
						}
						want := map[string]interface{}{
							"saml_auth_enabled": confirmed.GetSamlAuthEnabled(), "saml_auth_enforced": confirmed.GetSamlAuthEnforced(),
							"saml_metadata_inline": confirmed.GetSamlMetadataInline(), "saml_metadata_url": confirmed.GetSamlMetadataUrl(),
						}
						if !reflect.DeepEqual(body, want) {
							t.Error("PATCH did not set desired flags and clear the unused metadata mode")
						}
						response = confirmed
					case http.MethodGet:
						if writes != 1 {
							t.Errorf("GET before a single PATCH: writes=%d", writes)
						}
						response = confirmed
						if refreshing {
							refreshReads++
							if refresh == "different remote state" {
								// Without an ordered revision, a stale replica and a
								// legitimate external rollback are indistinguishable.
								response = stale
							}
						} else {
							applyReads++
							if applyReads > 1 {
								// The first GET confirms the write; a redundant GET
								// immediately afterwards returns the old snapshot.
								response = stale
							}
						}
					default:
						t.Errorf("unexpected method: %s", r.Method)
						http.Error(w, "unexpected method", http.StatusBadRequest)
						return
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Errorf("encode SAML auth: %v", err)
					}
				}))
				defer server.Close()

				res := resourceSAMLAuth()
				pc := testSAMLAuthProvider(server)
				var initial *terraform.InstanceState
				if tc.previous != nil {
					d := schema.TestResourceDataRaw(t, res.Schema, tc.previous)
					d.SetId(generateSAMLAuthID("example-org", stale))
					initial = d.State()
				}
				config := terraform.NewResourceConfigRaw(tc.desired)
				diff, err := res.Diff(context.Background(), initial, config, pc)
				if err != nil {
					t.Fatal(err)
				}
				if diff.Empty() || (initial != nil && diff.RequiresNew()) {
					t.Fatal("expected a create or in-place update")
				}
				state, diagnostics := res.Apply(context.Background(), initial, diff, pc)
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
						t.Errorf("%s plan empty=%t, want drift=%t", phase, diff.Empty(), wantDrift)
					}
				}
				checkPlan(state, "post-apply", false)
				if state.ID != generateSAMLAuthID("example-org", confirmed) {
					t.Error("apply did not retain the confirmed snapshot's identity")
				}
				if state.Attributes["saml_metadata_inline"] != confirmed.GetSamlMetadataInline() ||
					state.Attributes["saml_metadata_url"] != confirmed.GetSamlMetadataUrl() {
					t.Error("apply discarded confirmed metadata")
				}

				mu.Lock()
				refreshing = true
				mu.Unlock()
				state, diagnostics = res.RefreshWithoutUpgrade(context.Background(), state, pc)
				if diagnostics.HasError() {
					t.Fatalf("refresh failed: %v", diagnostics)
				}
				wantDrift := refresh == "different remote state"
				checkPlan(state, "post-refresh", wantDrift)
				wantSnapshot := confirmed
				if wantDrift {
					wantSnapshot = stale
				}
				if state.ID != generateSAMLAuthID("example-org", wantSnapshot) {
					t.Error("refresh did not preserve the observed remote state")
				}
				mu.Lock()
				defer mu.Unlock()
				if writes != 1 || applyReads != 1 || refreshReads != 1 {
					t.Errorf("writes=%d apply reads=%d refresh reads=%d, want 1 each", writes, applyReads, refreshReads)
				}
			})
		}
	}
}

func TestSAMLAuthImportAndDelete(t *testing.T) {
	t.Parallel()
	for _, inline := range []bool{false, true} {
		t.Run(fmt.Sprintf("inline=%t", inline), func(t *testing.T) {
			t.Parallel()
			snapshot := testSAMLAuthResponse(testSAMLAuthConfig(inline, true, true))
			var mu sync.Mutex
			var deleting bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodPatch {
					var body map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode PATCH: %v", err)
					}
					want := map[string]interface{}{
						"saml_auth_enabled": false, "saml_auth_enforced": false,
						"saml_metadata_inline": "", "saml_metadata_url": "",
					}
					if !reflect.DeepEqual(body, want) {
						t.Error("delete must disable both flags and clear both metadata modes")
					}
					deleting = true
				} else if r.Method != http.MethodGet {
					t.Errorf("unexpected method: %s", r.Method)
				}
				response := snapshot
				if deleting {
					response = cloudsmith.NewOrganizationSAMLAuth(false, false)
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Errorf("encode SAML auth: %v", err)
				}
			}))
			defer server.Close()
			res := resourceSAMLAuth()
			pc := testSAMLAuthProvider(server)
			d := schema.TestResourceDataRaw(t, res.Schema, nil)
			d.SetId("example-org")
			imported, err := res.Importer.StateContext(context.Background(), d, pc)
			if err != nil || len(imported) != 1 {
				t.Fatalf("import: resources=%d err=%v", len(imported), err)
			}
			if imported[0].Id() != generateSAMLAuthID("example-org", snapshot) || imported[0].Get("organization") != "example-org" {
				t.Fatal("import did not restore the organization and snapshot identity")
			}
			state, diagnostics := res.RefreshWithoutUpgrade(context.Background(), imported[0].State(), pc)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			state, diagnostics = res.Apply(context.Background(), state, &terraform.InstanceDiff{Destroy: true}, pc)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if state != nil && state.ID != "" {
				t.Fatal("delete did not clear identity")
			}
		})
	}
}

func TestSAMLAuthWaitForState(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		previous *cloudsmith.OrganizationSAMLAuth
		desired  *cloudsmith.OrganizationSAMLAuth
	}{
		{name: "URL to inline", previous: testSAMLAuthResponse(testSAMLAuthConfig(false, true, false)), desired: testSAMLAuthResponse(testSAMLAuthConfig(true, true, false))},
		{name: "inline to URL", previous: testSAMLAuthResponse(testSAMLAuthConfig(true, true, false)), desired: testSAMLAuthResponse(testSAMLAuthConfig(false, true, false))},
		{name: "enable", previous: testSAMLAuthResponse(testSAMLAuthConfig(false, false, false)), desired: testSAMLAuthResponse(testSAMLAuthConfig(false, true, false))},
		{name: "enforce", previous: testSAMLAuthResponse(testSAMLAuthConfig(true, true, false)), desired: testSAMLAuthResponse(testSAMLAuthConfig(true, true, true))},
		{name: "clear metadata", previous: testSAMLAuthResponse(testSAMLAuthConfig(false, false, false)), desired: cloudsmith.NewOrganizationSAMLAuth(false, false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var reads int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Method != http.MethodGet || r.Header.Get("X-Api-Key") != "test-api-key" {
					t.Error("waiter did not make an authenticated GET")
				}
				reads++
				response := tc.previous
				if reads > 1 {
					response = tc.desired
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Errorf("encode SAML auth: %v", err)
				}
			}))
			defer server.Close()
			response, err := waitForSAMLAuthState(context.Background(), testSAMLAuthProvider(server), "example-org",
				tc.desired.GetSamlAuthEnabled(), tc.desired.GetSamlAuthEnforced(),
				tc.desired.GetSamlMetadataInline(), tc.desired.GetSamlMetadataUrl(), 5*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if response.GetSamlMetadataInline() != tc.desired.GetSamlMetadataInline() ||
				response.GetSamlMetadataUrl() != tc.desired.GetSamlMetadataUrl() ||
				response.GetSamlAuthEnabled() != tc.desired.GetSamlAuthEnabled() ||
				response.GetSamlAuthEnforced() != tc.desired.GetSamlAuthEnforced() {
				t.Error("waiter returned a snapshot before all managed values converged")
			}
			mu.Lock()
			defer mu.Unlock()
			if reads != 2 {
				t.Errorf("reads=%d, want 2", reads)
			}
		})
	}
}

func TestSAMLAuthWaitReturnsObservedSnapshot(t *testing.T) {
	t.Parallel()
	for _, inline := range []bool{false, true} {
		t.Run(fmt.Sprintf("inline=%t", inline), func(t *testing.T) {
			t.Parallel()
			snapshot := testSAMLAuthResponse(testSAMLAuthConfig(inline, true, false))
			if inline {
				snapshot.SetSamlMetadataInline("\n" + testSAMLMetadata + "\n")
				snapshot.SetSamlMetadataUrlNil()
			} else {
				snapshot.SetSamlMetadataUrl(" https://test.idp.example.com/metadata.xml ")
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(snapshot); err != nil {
					t.Errorf("encode SAML auth: %v", err)
				}
			}))
			defer server.Close()
			response, err := waitForSAMLAuthState(context.Background(), testSAMLAuthProvider(server), "example-org",
				true, false, strings.TrimSpace(snapshot.GetSamlMetadataInline()), strings.TrimSpace(snapshot.GetSamlMetadataUrl()), 5*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if response.GetSamlMetadataInline() != snapshot.GetSamlMetadataInline() ||
				response.GetSamlMetadataUrl() != snapshot.GetSamlMetadataUrl() {
				t.Error("waiter synthesized desired values instead of returning the observed snapshot")
			}
		})
	}
}

func testSAMLAuthOperation(operation string, ctx context.Context, d *schema.ResourceData, pc *providerConfig) diag.Diagnostics {
	switch operation {
	case "create":
		return samlAuthCreate(ctx, d, pc)
	case "update":
		return samlAuthUpdate(ctx, d, pc)
	case "delete":
		return samlAuthDelete(ctx, d, pc)
	case "import":
		_, err := samlAuthImport(ctx, d, pc)
		return diag.FromErr(err)
	case "wait":
		_, err := waitForSAMLAuthState(ctx, pc, "example-org", false, false, "", "", 5*time.Second)
		return diag.FromErr(err)
	default:
		return samlAuthRead(ctx, d, pc)
	}
}

type samlAuthTestTransport func(*http.Request) (*http.Response, error)

func (f samlAuthTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSAMLAuthErrors(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"create", "update", "read", "delete", "import", "wait"} {
		for _, tc := range []struct {
			name   string
			status int
			body   string
			want   string
		}{
			{name: "not found", status: http.StatusNotFound, body: `{"detail":"not found"}`, want: "not found"},
			{name: "forbidden", status: http.StatusForbidden, body: `{"detail":"forbidden"}`, want: "forbidden"},
			{name: "throttled", status: http.StatusTooManyRequests, body: `{"detail":"throttled"}`, want: "throttled"},
			{name: "server error", status: http.StatusInternalServerError, body: `{"detail":"server failed"}`, want: "server failed"},
			{name: "invalid fields", status: http.StatusBadRequest, body: `{"detail":"invalid","fields":{"saml_metadata_url":["invalid URL"]}}`, want: "invalid URL"},
			{name: "empty body", status: http.StatusOK, want: "empty SAML authentication response"},
			{name: "null body", status: http.StatusOK, body: "null", want: "empty SAML authentication response"},
			{name: "malformed body", status: http.StatusOK, body: "{", want: "API error: {"},
			{name: "transport error", want: "transport failed"},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var requests int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					requests++
					mu.Unlock()
					wantMethod := http.MethodPatch
					if operation == "read" || operation == "import" || operation == "wait" {
						wantMethod = http.MethodGet
					}
					if r.Method != wantMethod {
						t.Errorf("request method=%s, want %s", r.Method, wantMethod)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.status)
					fmt.Fprint(w, tc.body)
				}))
				defer server.Close()
				pc := testSAMLAuthProvider(server)
				if tc.status == 0 {
					pc.APIClient.GetConfig().HTTPClient.Transport = samlAuthTestTransport(func(_ *http.Request) (*http.Response, error) {
						mu.Lock()
						requests++
						mu.Unlock()
						return nil, errors.New("transport failed")
					})
				}
				d := schema.TestResourceDataRaw(t, resourceSAMLAuth().Schema, testSAMLAuthConfig(true, true, true))
				d.SetId("example-org")
				before := d.State()
				diagnostics := testSAMLAuthOperation(operation, context.Background(), d, pc)
				if operation == "read" && tc.status == http.StatusNotFound {
					if diagnostics.HasError() || d.Id() != "" {
						t.Fatalf("missing configuration: diagnostics=%v ID=%q", diagnostics, d.Id())
					}
				} else {
					if !diagnostics.HasError() || !strings.Contains(fmt.Sprint(diagnostics), tc.want) {
						t.Errorf("diagnostics=%v, want %q", diagnostics, tc.want)
					}
					if !reflect.DeepEqual(d.State(), before) {
						t.Error("failed operation changed state or identity")
					}
				}
				mu.Lock()
				defer mu.Unlock()
				if requests != 1 {
					t.Errorf("requests=%d, terminal errors must not be retried", requests)
				}
			})
		}
	}
}

func TestSAMLAuthCancellation(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"create", "update", "read", "delete", "import", "wait"} {
		for _, phase := range []string{"request", "waiter"} {
			if phase == "waiter" && operation != "create" && operation != "update" && operation != "delete" {
				continue
			}
			t.Run(operation+"/"+phase, func(t *testing.T) {
				t.Parallel()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				release := make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if phase == "waiter" && r.Method == http.MethodPatch {
						w.Header().Set("Content-Type", "application/json")
						if err := json.NewEncoder(w).Encode(testSAMLAuthResponse(testSAMLAuthConfig(true, true, true))); err != nil {
							t.Errorf("encode SAML auth: %v", err)
						}
						return
					}
					cancel()
					<-release
				}))
				defer server.Close()
				defer close(release)
				pc := testSAMLAuthProvider(server)
				d := schema.TestResourceDataRaw(t, resourceSAMLAuth().Schema, testSAMLAuthConfig(true, true, true))
				d.SetId("example-org")
				start := time.Now()
				diagnostics := testSAMLAuthOperation(operation, ctx, d, pc)
				if !diagnostics.HasError() || !strings.Contains(fmt.Sprint(diagnostics), "context canceled") {
					t.Errorf("expected cancellation, got %v", diagnostics)
				}
				if time.Since(start) > 2*time.Second {
					t.Error("cancellation did not promptly stop the request")
				}
				wantID := "example-org"
				if operation == "create" && phase == "waiter" {
					wantID = generateSAMLAuthID("example-org", testSAMLAuthResponse(testSAMLAuthConfig(true, true, true)))
				}
				if d.Id() != wantID || d.Get("saml_metadata_inline") != testSAMLMetadata {
					t.Error("canceled operation discarded state or identity")
				}
			})
		}
	}
}

func TestSAMLAuthWaitDeadline(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"stale response", "both metadata modes", "blocked request"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if scenario == "blocked request" {
					<-release
					return
				}
				w.Header().Set("Content-Type", "application/json")
				response := testSAMLAuthResponse(testSAMLAuthConfig(false, true, false))
				if scenario == "both metadata modes" {
					response.SetSamlMetadataInline(testSAMLMetadata)
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Errorf("encode SAML auth: %v", err)
				}
			}))
			defer server.Close()
			defer close(release)
			start := time.Now()
			response, err := waitForSAMLAuthState(context.Background(), testSAMLAuthProvider(server),
				"example-org", true, false, testSAMLMetadata, "", 50*time.Millisecond)
			if response != nil || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("expected nil snapshot and deadline error, got snapshot=%t err=%v", response != nil, err)
			}
			if strings.Contains(err.Error(), testSAMLMetadata) || strings.Contains(err.Error(), "EntityDescriptor") ||
				!strings.Contains(err.Error(), "sha256=") || !strings.Contains(err.Error(), "len=") {
				t.Error("timeout must describe inline metadata without disclosing XML")
			}
			if time.Since(start) > 2*time.Second {
				t.Error("deadline did not bound the request and poll interval")
			}
		})
	}
}
