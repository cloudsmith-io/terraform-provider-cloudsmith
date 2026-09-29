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
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/go-cty/cty"
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
			}
		}
		for _, field := range []string{"is_enabled", "max_age_hours"} {
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

func TestAPIKeyRuleRejectsRemovedRefresh(t *testing.T) {
	t.Parallel()
	for _, ruleType := range apiKeyRuleTypes {
		t.Run(ruleType, func(t *testing.T) {
			t.Parallel()
			config := terraform.NewResourceConfigRaw(testAPIKeyRuleConfig(ruleType, true, 48, true))
			diagnostics := resourceAPIKeyRule().Validate(config)
			if !diagnostics.HasError() {
				// Exercise the pre-fix acceptance failure through real SDK state
				// handoff, not just a request-serialization proxy.
				fixture := &apiKeyRuleFixture{t: t, key: "unchanged-local-key", keyAge: 96, ruleType: ruleType}
				server := httptest.NewServer(fixture)
				defer server.Close()
				pc := testAPIKeyRuleProvider(server)
				res := resourceAPIKeyRule()
				diff, err := res.Diff(context.Background(), nil, config, pc)
				if err != nil {
					t.Fatal(err)
				}
				state, applyDiagnostics := res.Apply(context.Background(), nil, diff, pc)
				if applyDiagnostics.HasError() {
					t.Fatal(applyDiagnostics)
				}
				state, applyDiagnostics = res.RefreshWithoutUpgrade(context.Background(), state, pc)
				if applyDiagnostics.HasError() {
					t.Fatal(applyDiagnostics)
				}
				diff, err = res.Diff(context.Background(), state, config, pc)
				if err != nil {
					t.Fatal(err)
				}
				fixture.mu.Lock()
				t.Logf("unsupported request accepted: sent enforce_refresh=%v, refreshed state=%s, post-refresh diff empty=%t, key unchanged=%t",
					fixture.requests[0][EnforceRefresh], state.Attributes[EnforceRefresh], diff.Empty(), fixture.key == "unchanged-local-key")
				fixture.mu.Unlock()
			}
			if !diagnostics.HasError() || !strings.Contains(fmt.Sprint(diagnostics), "automatic API key refresh is no longer supported") {
				t.Fatalf("enforce_refresh=true must fail validation, got %v", diagnostics)
			}
		})
	}
}

func TestAPIKeyRuleContractLifecycle(t *testing.T) {
	t.Parallel()
	for _, ruleType := range apiKeyRuleTypes {
		for _, refresh := range []interface{}{nil, false} {
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
					if state.Attributes[EnforceRefresh] != "false" || state.Attributes[IsEnabled] != fmt.Sprint(configMap[IsEnabled]) {
						t.Fatal("state must reflect current API fields, not unsupported refresh intent")
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
					if step == 0 {
						want[RuleType] = ruleType
					}
					if !reflect.DeepEqual(request, want) {
						t.Errorf("step %d payload=%v, want %v", step, request, want)
					}
					for _, removed := range []string{EnforceRefresh, "refresh_immediately"} {
						if _, ok := request[removed]; ok {
							t.Errorf("request must omit %s, including explicit false", removed)
						}
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
					t.Fatalf("import did not read current API contract: %v", diagnostics)
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

func TestAPIKeyRuleRejectsRefreshAtApply(t *testing.T) {
	t.Parallel()
	for _, ruleType := range apiKeyRuleTypes {
		for _, operation := range []string{"create", "update"} {
			for _, entry := range []string{"SDK Apply", "direct callback"} {
				t.Run(ruleType+"/"+operation+"/"+entry, func(t *testing.T) {
					t.Parallel()
					var requests atomic.Int64
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						requests.Add(1)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusServiceUnavailable)
						fmt.Fprint(w, `{"detail":"unexpected request"}`)
					}))
					defer server.Close()
					pc := testAPIKeyRuleProvider(server)
					res := resourceAPIKeyRule()
					known := testAPIKeyRuleConfig(ruleType, true, 48, true)

					if entry == "SDK Apply" {
						coreSchema := res.CoreConfigSchema()
						values := make(map[string]cty.Value, len(coreSchema.Attributes))
						for name, attribute := range coreSchema.Attributes {
							values[name] = cty.NullVal(attribute.Type)
						}
						values[Organization] = cty.StringVal("isolated-org")
						values[RuleType] = cty.StringVal(ruleType)
						values[MaxAgeHours] = cty.NumberIntVal(48)
						values[IsEnabled] = cty.True
						values[EnforceRefresh] = cty.UnknownVal(cty.Bool)
						unknown := terraform.NewResourceConfigShimmed(cty.ObjectVal(values), coreSchema)
						if diagnostics := res.Validate(unknown); diagnostics.HasError() {
							t.Fatalf("unknown input must pass schema validation: %v", diagnostics)
						}
						var initial *terraform.InstanceState
						if operation == "update" {
							d := schema.TestResourceDataRaw(t, res.Schema, testAPIKeyRuleConfig(ruleType, false, 48, true))
							d.SetId("rule-id")
							initial = d.State()
						}
						planned, err := res.Diff(context.Background(), initial, unknown, pc)
						if err != nil {
							t.Fatal(err)
						}
						if attribute := planned.Attributes[EnforceRefresh]; attribute == nil || !attribute.NewComputed {
							t.Fatal("plan must retain the unknown refresh value")
						}
						// Terraform resolves the unknown expression before Apply.
						resolved, err := res.Diff(context.Background(), initial, terraform.NewResourceConfigRaw(known), pc)
						if err != nil {
							t.Fatal(err)
						}
						_, diagnostics := res.Apply(context.Background(), initial, resolved, pc)
						if !diagnostics.HasError() || !strings.Contains(fmt.Sprint(diagnostics), "automatic API key refresh is no longer supported") {
							t.Errorf("apply must reject resolved true: %v", diagnostics)
						}
					} else {
						d := schema.TestResourceDataRaw(t, res.Schema, known)
						callback := resourceAPIKeyRuleCreate
						if operation == "update" {
							d.SetId("rule-id")
							callback = resourceAPIKeyRuleUpdate
						}
						before := d.State()
						err := callback(d, pc)
						if err == nil || !strings.Contains(err.Error(), "automatic API key refresh is no longer supported") {
							t.Errorf("callback must reject true: %v", err)
						}
						if !reflect.DeepEqual(before, d.State()) {
							t.Error("rejected callback changed resource state")
						}
					}
					if got := requests.Load(); got != 0 {
						t.Errorf("unsupported true made %d HTTP requests, want zero", got)
					}
				})
			}
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

func TestAPIKeyRuleReadClearsLegacyRefresh(t *testing.T) {
	t.Parallel()
	for _, explicitFalse := range []bool{false, true} {
		t.Run(fmt.Sprintf("explicit API false=%t", explicitFalse), func(t *testing.T) {
			t.Parallel()
			fixture := &apiKeyRuleFixture{
				t: t, ruleType: "User Accounts",
				rule: map[string]interface{}{
					"slug_perm": "rule-id", "slug": "account-policy", RuleType: "User Accounts",
					IsEnabled: false, MaxAgeHours: 72,
				},
			}
			if explicitFalse {
				fixture.rule[EnforceRefresh] = false
			}
			server := httptest.NewServer(fixture)
			defer server.Close()
			pc := testAPIKeyRuleProvider(server)
			res := resourceAPIKeyRule()
			d := schema.TestResourceDataRaw(t, res.Schema, testAPIKeyRuleConfig("User Accounts", true, 48, true))
			d.SetId("rule-id")
			state, diagnostics := res.RefreshWithoutUpgrade(context.Background(), d.State(), pc)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if state.Attributes[EnforceRefresh] != "false" || state.Attributes[IsEnabled] != "false" || state.Attributes[MaxAgeHours] != "72" {
				t.Fatal("read must not retain obsolete true state or mask actual API values")
			}
			diff, err := res.Diff(context.Background(), state,
				terraform.NewResourceConfigRaw(testAPIKeyRuleConfig("User Accounts", false, 48, true)), pc)
			if err != nil || diff.Empty() {
				t.Fatalf("expected real supported-field drift: diff=%v err=%v", diff, err)
			}
		})
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
