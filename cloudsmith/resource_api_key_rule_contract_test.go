// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package cloudsmith

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type apiKeyRuleFixture struct {
	t        *testing.T
	mu       sync.Mutex
	rule     map[string]interface{}
	requests []map[string]interface{}
	key      string
	keyAge   int
	ruleType string
}

func (f *apiKeyRuleFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.Header.Get("X-Api-Key") != "local-fixture-only" ||
		(r.URL.Path != "/orgs/isolated-org/api-key-rules/" && r.URL.Path != "/orgs/isolated-org/api-key-rules/rule-id/") {
		http.Error(w, `{"detail":"unexpected request"}`, http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPost, http.MethodPatch:
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"detail":"invalid body"}`, http.StatusBadRequest)
			return
		}
		f.requests = append(f.requests, body)
		if f.rule == nil {
			f.rule = map[string]interface{}{
				"slug_perm": "rule-id", "slug": "account-policy", "rule_type": f.ruleType,
				"created_at": "2026-09-29T12:00:00Z", "last_applied_at": nil, "is_enabled": true,
				"enforce_refresh": false,
			}
		}
		for _, field := range []string{"enforce_refresh", "is_enabled", "max_age_hours"} {
			if value, ok := body[field]; ok {
				f.rule[field] = value
			}
		}
		f.rule["updated_at"] = fmt.Sprintf("2026-09-29T12:00:%02dZ", len(f.requests))
	case http.MethodGet:
		if f.rule == nil {
			http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
			return
		}
	case http.MethodDelete:
		f.rule = nil
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		http.Error(w, `{"detail":"unexpected method"}`, http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	}
	if err := json.NewEncoder(w).Encode(f.rule); err != nil {
		f.t.Errorf("encode API key rule: %v", err)
	}
}

func testAPIKeyRuleProvider(server *httptest.Server) *providerConfig {
	config := cloudsmith.NewConfiguration()
	config.Servers = cloudsmith.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()
	config.HTTPClient.Timeout = 5 * time.Second
	return &providerConfig{
		APIClient: cloudsmith.NewAPIClient(config),
		Auth: context.WithValue(context.Background(), cloudsmith.ContextAPIKeys,
			map[string]cloudsmith.APIKey{"apikey": {Key: "local-fixture-only"}}),
	}
}

func testAPIKeyRuleConfig(ruleType string, refresh interface{}, maxAge interface{}, enabled bool) map[string]interface{} {
	config := map[string]interface{}{
		Organization: "isolated-org", RuleType: ruleType, IsEnabled: enabled,
	}
	if refresh != nil {
		config[EnforceRefresh] = refresh
	}
	if maxAge != nil {
		config[MaxAgeHours] = maxAge
	}
	return config
}

func TestAPIKeyRuleContractLifecycle(t *testing.T) {
	t.Parallel()
	for _, ruleType := range apiKeyRuleTypes {
		for _, refresh := range []interface{}{nil, false, true} {
			t.Run(fmt.Sprintf("%s/refresh=%v", ruleType, refresh), func(t *testing.T) {
				t.Parallel()
				fixture := &apiKeyRuleFixture{t: t, key: "unchanged-local-key", keyAge: 96, ruleType: ruleType}
				server := httptest.NewServer(fixture)
				defer server.Close()
				pc := testAPIKeyRuleProvider(server)
				res := resourceAPIKeyRule()
				var state *terraform.InstanceState
				for step, configMap := range []map[string]interface{}{
					testAPIKeyRuleConfig(ruleType, refresh, 48, true),
					testAPIKeyRuleConfig(ruleType, false, 72, false),
					testAPIKeyRuleConfig(ruleType, false, nil, false),
				} {
					config := terraform.NewResourceConfigRaw(configMap)
					if diagnostics := res.Validate(config); diagnostics.HasError() {
						t.Fatal(diagnostics)
					}
					diff, err := res.Diff(context.Background(), state, config, pc)
					if err != nil {
						t.Fatal(err)
					}
					applied, diagnostics := res.Apply(context.Background(), state, diff, pc)
					if diagnostics.HasError() {
						t.Fatal(diagnostics)
					}
					state, diagnostics = res.RefreshWithoutUpgrade(context.Background(), applied, pc)
					if diagnostics.HasError() {
						t.Fatal(diagnostics)
					}
					wantRefresh := "false"
					if configured, ok := configMap[EnforceRefresh]; ok {
						wantRefresh = fmt.Sprint(configured)
					}
					if state.Attributes[EnforceRefresh] != wantRefresh || state.Attributes[IsEnabled] != fmt.Sprint(configMap[IsEnabled]) {
						t.Fatalf("step %d: state must reflect the API, got enforce_refresh=%s is_enabled=%s",
							step, state.Attributes[EnforceRefresh], state.Attributes[IsEnabled])
					}
					diff, err = res.Diff(context.Background(), state, config, pc)
					if err != nil || !diff.Empty() {
						t.Fatalf("step %d: nonempty post-refresh diff=%#v err=%v", step, diff, err)
					}
					fixture.mu.Lock()
					request := fixture.requests[len(fixture.requests)-1]
					want := map[string]interface{}{IsEnabled: configMap[IsEnabled], MaxAgeHours: nil}
					if age, ok := configMap[MaxAgeHours]; ok {
						want[MaxAgeHours] = float64(age.(int))
					}
					if refresh, ok := configMap[EnforceRefresh]; ok {
						want[EnforceRefresh] = refresh
					}
					if step == 0 {
						want[RuleType] = ruleType
					}
					if !reflect.DeepEqual(request, want) {
						t.Errorf("step %d payload=%v, want %v", step, request, want)
					}
					if _, ok := request["refresh_immediately"]; ok {
						t.Error("request must never trigger an immediate rotation")
					}
					if fixture.key != "unchanged-local-key" || fixture.keyAge != 96 {
						t.Error("rule operations must not rotate fixture keys")
					}
					fixture.mu.Unlock()
				}

				fixture.mu.Lock()
				fixture.rule[IsEnabled] = true
				fixture.rule[MaxAgeHours] = float64(96)
				fixture.mu.Unlock()
				refreshed, diagnostics := res.RefreshWithoutUpgrade(context.Background(), state, pc)
				if diagnostics.HasError() {
					t.Fatal(diagnostics)
				}
				if refreshed.Attributes[IsEnabled] != "true" || refreshed.Attributes[MaxAgeHours] != "96" {
					t.Fatal("refresh masked changes to supported remote fields")
				}
				drift, err := res.Diff(context.Background(), refreshed,
					terraform.NewResourceConfigRaw(testAPIKeyRuleConfig(ruleType, false, nil, false)), pc)
				if err != nil || drift.Empty() {
					t.Fatalf("expected external drift, got diff=%#v err=%v", drift, err)
				}

				d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{EnforceRefresh: true})
				d.SetId("isolated-org.rule-id")
				imported, err := res.Importer.StateContext(context.Background(), d, pc)
				if err != nil || len(imported) != 1 {
					t.Fatalf("import: %v", err)
				}
				importedState, diagnostics := res.RefreshWithoutUpgrade(context.Background(), imported[0].State(), pc)
				if diagnostics.HasError() || importedState.ID != "rule-id" || importedState.Attributes[EnforceRefresh] != "false" {
					t.Fatalf("import must read enforce_refresh from the API, not the import config: %v", diagnostics)
				}
				importedState.Attributes[EnforceRefresh] = "true"
				state, diagnostics = res.Apply(context.Background(), importedState, &terraform.InstanceDiff{Destroy: true}, pc)
				if diagnostics.HasError() || state != nil && state.ID != "" {
					t.Fatalf("delete: state=%v diagnostics=%v", state, diagnostics)
				}
			})
		}
	}
}

func TestAPIKeyRuleRequestDefaults(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		request interface{}
	}{
		{name: "create", request: cloudsmith.NewOrganizationApiKeyRuleRequest("User Accounts")},
		{name: "update", request: cloudsmith.NewOrganizationApiKeyRuleRequestPatch()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]interface{}
			if err := json.Unmarshal(body, &fields); err != nil {
				t.Fatal(err)
			}
			for _, removed := range []string{EnforceRefresh, "refresh_immediately"} {
				if _, ok := fields[removed]; ok {
					t.Errorf("SDK constructor must not initialize %s", removed)
				}
			}
		})
	}
}

func TestAPIKeyRuleReadsRefreshFromAPI(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		api  interface{}
		want string
	}{
		{name: "omitted", api: nil, want: "false"},
		{name: "false", api: false, want: "false"},
		{name: "true", api: true, want: "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := &apiKeyRuleFixture{
				t: t, ruleType: "User Accounts",
				rule: map[string]interface{}{
					"slug_perm": "rule-id", "slug": "account-policy", RuleType: "User Accounts",
					IsEnabled: false, MaxAgeHours: 72,
				},
			}
			if tc.api != nil {
				fixture.rule[EnforceRefresh] = tc.api
			}
			server := httptest.NewServer(fixture)
			defer server.Close()
			pc := testAPIKeyRuleProvider(server)
			res := resourceAPIKeyRule()
			d := schema.TestResourceDataRaw(t, res.Schema, testAPIKeyRuleConfig("User Accounts", nil, 48, true))
			d.SetId("rule-id")
			state, diagnostics := res.RefreshWithoutUpgrade(context.Background(), d.State(), pc)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if state.Attributes[EnforceRefresh] != tc.want || state.Attributes[IsEnabled] != "false" || state.Attributes[MaxAgeHours] != "72" {
				t.Fatalf("read must mirror the API, got enforce_refresh=%s", state.Attributes[EnforceRefresh])
			}
		})
	}
}

// A rule can have auto-rotation switched on in the web app; configuration that
// never mentions enforce_refresh must not switch it back off.
func TestAPIKeyRuleOmittedRefreshKeepsRemoteSetting(t *testing.T) {
	t.Parallel()
	fixture := &apiKeyRuleFixture{
		t: t, ruleType: "Service Accounts",
		rule: map[string]interface{}{
			"slug_perm": "rule-id", "slug": "account-policy", RuleType: "Service Accounts",
			IsEnabled: true, MaxAgeHours: float64(48), EnforceRefresh: true,
		},
	}
	server := httptest.NewServer(fixture)
	defer server.Close()
	pc := testAPIKeyRuleProvider(server)
	res := resourceAPIKeyRule()

	d := schema.TestResourceDataRaw(t, res.Schema, testAPIKeyRuleConfig("Service Accounts", nil, 48, true))
	d.SetId("rule-id")
	state, diagnostics := res.RefreshWithoutUpgrade(context.Background(), d.State(), pc)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}

	config := terraform.NewResourceConfigRaw(testAPIKeyRuleConfig("Service Accounts", nil, 72, true))
	diff, err := res.Diff(context.Background(), state, config, pc)
	if err != nil {
		t.Fatal(err)
	}
	if attribute, ok := diff.Attributes[EnforceRefresh]; ok && attribute.New != attribute.Old {
		t.Fatalf("omitted enforce_refresh must not plan a change: %#v", attribute)
	}
	if _, diagnostics := res.Apply(context.Background(), state, diff, pc); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}

	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if sent, ok := fixture.requests[len(fixture.requests)-1][EnforceRefresh]; ok && sent != true {
		t.Errorf("update sent enforce_refresh=%v for a rule that has it on", sent)
	}
	if fixture.rule[EnforceRefresh] != true {
		t.Error("auto-rotation set outside Terraform was switched off")
	}
}

func TestAPIKeyRuleErrors(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"create", "read", "update", "delete"} {
		for _, status := range []int{http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
			t.Run(fmt.Sprintf("%s/%d", operation, status), func(t *testing.T) {
				t.Parallel()
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					fmt.Fprint(w, `{"detail":"fixture API failure"}`)
				}))
				defer server.Close()
				pc := testAPIKeyRuleProvider(server)
				d := schema.TestResourceDataRaw(t, resourceAPIKeyRule().Schema, testAPIKeyRuleConfig("User Accounts", false, 48, true))
				d.SetId("rule-id")
				before := d.State()
				callback := map[string]func(*schema.ResourceData, interface{}) error{
					"create": resourceAPIKeyRuleCreate, "read": resourceAPIKeyRuleRead,
					"update": resourceAPIKeyRuleUpdate, "delete": resourceAPIKeyRuleDelete,
				}[operation]
				err := callback(d, pc)
				if operation == "read" && status == http.StatusNotFound {
					if err != nil || d.Id() != "" {
						t.Fatalf("read 404 must clear identity: ID=%s err=%v", d.Id(), err)
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), "fixture API failure") {
					t.Errorf("expected original API error detail, got %v", err)
				}
				if !reflect.DeepEqual(before, d.State()) {
					t.Error("API error changed state or identity")
				}
			})
		}
	}
}
