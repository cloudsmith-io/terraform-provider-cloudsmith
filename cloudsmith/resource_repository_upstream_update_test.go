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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestRepositoryUpstreamUpdatePlan(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var reads, writes int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/repos/example-org/example-repo/upstream/composer/upstream-id/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusBadRequest)
			return
		}
		var response map[string]interface{}
		switch r.Method {
		case http.MethodPut:
			writes++
			response = testUpstreamResponse(Composer, true)
		case http.MethodGet:
			reads++
			response = testUpstreamResponse(Composer, reads >= 3)
		default:
			t.Errorf("unexpected method: %s", r.Method)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode upstream: %v", err)
		}
	}))
	defer server.Close()

	pc := testUpstreamProviderConfig(server)
	res := resourceRepositoryUpstream()
	previous := testUpstreamResponse(Composer, false)
	previous[Namespace] = "example-org"
	previous[Repository] = "example-repo"
	previous[UpstreamType] = Composer
	previous[UpdatedAt] = timeToString(stringToTime(previous[UpdatedAt].(string)))
	d := schema.TestResourceDataRaw(t, res.Schema, previous)
	d.SetId("upstream-id")
	config := terraform.NewResourceConfigRaw(testUpstreamUpdateConfig(Composer))
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
	// Acceptance tests also run the post-apply plan with refresh=false.
	diff, err = res.Diff(context.Background(), state, config, pc)
	if err != nil {
		t.Fatal(err)
	}
	if !diff.Empty() {
		for key, attr := range diff.Attributes {
			if key != AuthSecret {
				t.Errorf("nonempty post-apply plan: %s = %q -> %q", key, attr.Old, attr.New)
			}
		}
		t.Fatal("expected an empty post-apply plan")
	}
	mu.Lock()
	defer mu.Unlock()
	if writes != 1 || reads != 3 {
		t.Errorf("requests: writes=%d reads=%d, want writes=1 reads=3", writes, reads)
	}
}

func TestRepositoryUpstreamUpdateFreshness(t *testing.T) {
	t.Parallel()

	for _, upstreamType := range []string{Python, Composer} {
		for _, sequence := range []string{"fractional stale revision", "same second update", "intermediate revision", "stale after fresh", "newer than PUT"} {
			t.Run(upstreamType+"/"+sequence, func(t *testing.T) {
				t.Parallel()

				previous := testUpstreamResponse(upstreamType, false)
				updated := testUpstreamResponse(upstreamType, true)
				stale := testUpstreamResponse(upstreamType, false)
				if sequence == "same second update" {
					updated[UpdatedAt] = "2026-09-29T12:00:00.654321Z"
				}
				// An unrelated change may advance the timestamp without observing our PUT.
				if sequence == "intermediate revision" {
					stale[UpdatedAt] = "2026-09-29T12:00:01.500000Z"
				}
				responses := []map[string]interface{}{stale, stale, updated}
				wantReads := 3
				if sequence == "stale after fresh" {
					stale[UpdatedAt] = timeToString(stringToTime(previous[UpdatedAt].(string)))
					responses = []map[string]interface{}{stale, updated, previous}
					wantReads = 2
				}
				accepted := updated
				if sequence == "newer than PUT" {
					accepted = testUpstreamResponse(upstreamType, true)
					accepted[UpdatedAt] = "2026-09-29T12:00:03.123456Z"
					responses = []map[string]interface{}{accepted}
					wantReads = 1
				}

				var mu sync.Mutex
				var reads, writes int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path != "/repos/example-org/example-repo/upstream/"+upstreamType+"/upstream-id/" {
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
							http.Error(w, "invalid body", http.StatusBadRequest)
							return
						}
						for key, want := range testUpstreamUpdateConfig(upstreamType) {
							if key == Namespace || key == Repository || key == UpstreamType {
								continue
							}
							if got := body[key]; fmt.Sprint(got) != fmt.Sprint(want) {
								t.Errorf("PUT %s = %v, want %v", key, got, want)
							}
						}
					case http.MethodGet:
						if writes != 1 {
							t.Errorf("GET before a single update: writes=%d", writes)
						}
						response = responses[min(reads, len(responses)-1)]
						reads++
					default:
						t.Errorf("unexpected method: %s", r.Method)
						http.Error(w, "unexpected method", http.StatusBadRequest)
						return
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Errorf("encode upstream: %v", err)
					}
				}))
				defer server.Close()

				d := schema.TestResourceDataRaw(t, resourceRepositoryUpstream().Schema, testUpstreamUpdateConfig(upstreamType))
				d.SetId("upstream-id")
				// This is how the existing Read saves the fractional API timestamp.
				if err := d.Set(UpdatedAt, timeToString(stringToTime(previous[UpdatedAt].(string)))); err != nil {
					t.Fatal(err)
				}
				if err := resourceRepositoryUpstreamUpdate(d, testUpstreamProviderConfig(server)); err != nil {
					t.Fatal(err)
				}
				for key, want := range testUpstreamUpdateConfig(upstreamType) {
					if got := d.Get(key); !reflect.DeepEqual(got, want) {
						t.Errorf("state %s = %#v, want %#v", key, got, want)
					}
				}
				if got := d.Get(UpdatedAt); got != timeToString(stringToTime(accepted[UpdatedAt].(string))) {
					t.Errorf("updated_at = %v, want the accepted API revision", got)
				}
				mu.Lock()
				defer mu.Unlock()
				if writes != 1 || reads != wantReads {
					t.Errorf("requests: writes=%d reads=%d, want writes=1 reads=%d", writes, reads, wantReads)
				}
			})
		}
	}
}

func TestRepositoryUpstreamUpdateErrors(t *testing.T) {
	// These tests run serially, before any parallel tests, to shorten only their
	// bounded waiter without changing production defaults or adding retry hooks.
	previousTimeout, previousInterval := defaultUpdateTimeout, defaultUpdateInterval
	defaultUpdateTimeout, defaultUpdateInterval = 20*time.Millisecond, time.Millisecond
	t.Cleanup(func() {
		defaultUpdateTimeout, defaultUpdateInterval = previousTimeout, previousInterval
	})

	for _, tc := range []struct {
		name        string
		method      string
		status      int
		body        string
		wantReads   int
		wantErr     string
		wantTimeout bool
	}{
		{name: "PUT validation", method: http.MethodPut, status: http.StatusBadRequest, body: `{"detail":"Invalid upstream","fields":{"upstream_url":["Invalid URL"]}}`, wantErr: "Invalid upstream"},
		{name: "PUT unprocessable", method: http.MethodPut, status: http.StatusUnprocessableEntity, body: `{"detail":"Invalid credentials"}`, wantErr: "Invalid credentials"},
		{name: "GET not found", method: http.MethodGet, status: http.StatusNotFound, body: `{"detail":"Not found"}`, wantReads: 1, wantErr: "404"},
		{name: "GET unauthorized", method: http.MethodGet, status: http.StatusUnauthorized, body: `{"detail":"Unauthorized"}`, wantReads: 1, wantErr: "401"},
		{name: "GET forbidden", method: http.MethodGet, status: http.StatusForbidden, body: `{"detail":"Forbidden"}`, wantReads: 1, wantErr: "403"},
		{name: "GET throttled", method: http.MethodGet, status: http.StatusTooManyRequests, body: `{"detail":"Throttled"}`, wantReads: 1, wantErr: "429"},
		{name: "GET server failure", method: http.MethodGet, status: http.StatusInternalServerError, body: `{"detail":"Internal error"}`, wantReads: 1, wantErr: "500"},
		{name: "GET invalid JSON", method: http.MethodGet, status: http.StatusOK, body: `{`, wantReads: 1, wantErr: "unexpected end of JSON"},
		{name: "stale reads time out", wantTimeout: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var reads, writes int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodPut:
					writes++
				case http.MethodGet:
					reads++
				default:
					t.Errorf("unexpected method: %s", r.Method)
					http.Error(w, "unexpected method", http.StatusBadRequest)
					return
				}
				if r.Method == tc.method {
					w.WriteHeader(tc.status)
					fmt.Fprint(w, tc.body)
					return
				}
				if err := json.NewEncoder(w).Encode(testUpstreamResponse(Python, r.Method == http.MethodPut)); err != nil {
					t.Errorf("encode upstream: %v", err)
				}
			}))
			defer server.Close()
			d := schema.TestResourceDataRaw(t, resourceRepositoryUpstream().Schema, testUpstreamUpdateConfig(Python))
			d.SetId("upstream-id")
			const previousUpdatedAt = "2026-09-29T12:00:00Z"
			if err := d.Set(UpdatedAt, previousUpdatedAt); err != nil {
				t.Fatal(err)
			}

			err := resourceRepositoryUpstreamUpdate(d, testUpstreamProviderConfig(server))
			if tc.wantTimeout {
				if !errors.Is(err, errTimedOut) {
					t.Fatalf("update error = %v, want wrapped errTimedOut", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("update error = %v, want %q", err, tc.wantErr)
			}
			if tc.method != http.MethodPut && !strings.Contains(err.Error(), "error waiting for upstream (upstream-id) to be updated:") {
				t.Errorf("missing waiter error context: %v", err)
			}
			if d.Id() != "upstream-id" || d.Get(UpdatedAt) != previousUpdatedAt || d.Get(TrustLevel) != "Untrusted" {
				t.Error("failed update must retain identity and must not flatten a stale GET")
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 1 || (!tc.wantTimeout && reads != tc.wantReads) || (tc.wantTimeout && reads < 1) {
				t.Errorf("requests: writes=%d reads=%d, want writes=1 reads=%d (timeout=%t)", writes, reads, tc.wantReads, tc.wantTimeout)
			}
		})
	}
}

func TestRepositoryUpstreamReadStateFields(t *testing.T) {
	t.Parallel()
	for _, upstreamType := range upstreamTypes {
		t.Run(upstreamType, func(t *testing.T) {
			t.Parallel()
			response := testUpstreamResponse(upstreamType, true)
			want := testUpstreamUpdateConfig(upstreamType)
			want[CreatedAt] = timeToString(stringToTime(response[CreatedAt].(string)))
			want[UpdatedAt] = timeToString(stringToTime(response[UpdatedAt].(string)))
			want[SlugPerm] = "upstream-id"
			switch upstreamType {
			case Deb:
				response[Component], want[Component] = "main", "main"
				response[DistroVersions] = []string{"stable", "testing"}
				want[DistroVersions] = schema.NewSet(schema.HashString, []interface{}{"stable", "testing"})
				response[IncludeSources], want[IncludeSources] = true, true
				response[UpstreamDistribution], want[UpstreamDistribution] = "ubuntu", "ubuntu"
			case Generic:
				response[UpstreamPrefix], want[UpstreamPrefix] = "packages", "packages"
			case Rpm:
				response[DistroVersion], want[DistroVersion] = "fedora/35", "fedora/35"
				response[IncludeSources], want[IncludeSources] = true, true
			}
			// A cleared nullable field must remove the old value from state.
			response[ExtraHeader2], want[ExtraHeader2] = nil, ""
			response[ExtraValue2], want[ExtraValue2] = nil, ""
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/repos/example-org/example-repo/upstream/"+upstreamType+"/upstream-id/" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Errorf("encode upstream: %v", err)
				}
			}))
			defer server.Close()
			d := schema.TestResourceDataRaw(t, resourceRepositoryUpstream().Schema, testUpstreamUpdateConfig(upstreamType))
			d.SetId("upstream-id")
			if err := resourceRepositoryUpstreamRead(d, testUpstreamProviderConfig(server)); err != nil {
				t.Fatal(err)
			}
			for key, value := range want {
				got := d.Get(key)
				if key == DistroVersions {
					gotSet := got.(*schema.Set)
					if !gotSet.Equal(schema.NewSet(gotSet.F, value.(*schema.Set).List())) {
						t.Errorf("%s = %v, want %v", key, got, value)
					}
				} else if !reflect.DeepEqual(got, value) {
					t.Errorf("%s = %#v, want %#v", key, got, value)
				}
			}
		})
	}
}

func testUpstreamProviderConfig(server *httptest.Server) *providerConfig {
	config := cloudsmith.NewConfiguration()
	config.Servers = cloudsmith.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()
	return &providerConfig{Auth: context.Background(), APIClient: cloudsmith.NewAPIClient(config)}
}

func testUpstreamUpdateConfig(upstreamType string) map[string]interface{} {
	config := map[string]interface{}{
		Namespace:    "example-org",
		Repository:   "example-repo",
		UpstreamType: upstreamType,
		Name:         "example-upstream",
		UpstreamUrl:  "https://upstream.example.com",
		AuthMode:     "Username and Password",
		AuthUsername: "example-user",
		AuthSecret:   "mock-secret",
		ExtraHeader1: "X-Example",
		ExtraHeader2: "X-Other",
		ExtraValue1:  "example-value",
		ExtraValue2:  "other-value",
		IsActive:     false,
		Mode:         "Cache and Proxy",
		Priority:     1,
		VerifySsl:    false,
	}
	if upstreamType == Python || upstreamType == Maven || upstreamType == Npm {
		config[TrustLevel] = "Untrusted"
	}
	return config
}

func testUpstreamResponse(upstreamType string, updated bool) map[string]interface{} {
	response := testUpstreamUpdateConfig(upstreamType)
	delete(response, Namespace)
	delete(response, Repository)
	delete(response, UpstreamType)
	// Secrets are deliberately absent from both the PUT response and GETs.
	delete(response, AuthSecret)
	response[SlugPerm] = "upstream-id"
	response[CreatedAt] = "2026-09-29T11:00:00.123456Z"
	response[UpdatedAt] = "2026-09-29T12:00:02.123456Z"
	if !updated {
		response[UpdatedAt] = "2026-09-29T12:00:00.123456Z"
		response[AuthMode] = "None"
		response[AuthUsername] = nil
		response[ExtraHeader1] = nil
		response[ExtraHeader2] = nil
		response[ExtraValue1] = nil
		response[ExtraValue2] = nil
		response[IsActive] = true
		response[Mode] = "Proxy Only"
		response[VerifySsl] = true
		if upstreamType == Python || upstreamType == Maven || upstreamType == Npm {
			response[TrustLevel] = "Trusted"
		}
	}
	return response
}
