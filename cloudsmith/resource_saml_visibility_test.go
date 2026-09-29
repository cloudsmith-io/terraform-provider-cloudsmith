package cloudsmith

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testSAMLGroupWaits(t *testing.T) {
	t.Helper()
	createTimeout, createInterval := defaultCreationTimeout, defaultCreationInterval
	deleteTimeout, deleteInterval := defaultDeletionTimeout, defaultDeletionInterval
	updateTimeout, updateInterval := defaultUpdateTimeout, defaultUpdateInterval
	defaultCreationTimeout, defaultCreationInterval = 5*time.Second, time.Millisecond
	defaultDeletionTimeout, defaultDeletionInterval = 5*time.Second, time.Millisecond
	defaultUpdateTimeout, defaultUpdateInterval = 5*time.Second, time.Millisecond
	t.Cleanup(func() {
		defaultCreationTimeout, defaultCreationInterval = createTimeout, createInterval
		defaultDeletionTimeout, defaultDeletionInterval = deleteTimeout, deleteInterval
		defaultUpdateTimeout, defaultUpdateInterval = updateTimeout, updateInterval
	})
}

func testSAMLGroupProvider(server *httptest.Server) *providerConfig {
	config := cloudsmith.NewConfiguration()
	config.Servers = cloudsmith.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()
	return &providerConfig{Auth: context.Background(), APIClient: cloudsmith.NewAPIClient(config)}
}

func testSAMLGroupConfig() map[string]interface{} {
	return map[string]interface{}{
		"organization": "organization", "idp_key": "groups", "idp_value": "engineers",
		"role": "Member", "team": "team",
	}
}

func testSAMLGroupMapping(config map[string]interface{}, id string) map[string]interface{} {
	return map[string]interface{}{
		"slug_perm": id, "idp_key": config["idp_key"], "idp_value": config["idp_value"],
		"role": config["role"], "team": config["team"],
	}
}

func testSAMLGroupPage(t *testing.T, w http.ResponseWriter, page, total int, items ...map[string]interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(paginationCountHeader, "2")
	w.Header().Set(paginationPageHeader, strconv.Itoa(page))
	w.Header().Set(paginationPageTotalHeader, strconv.Itoa(total))
	w.Header().Set(paginationPageSizeHeader, "100")
	if items == nil {
		items = []map[string]interface{}{}
	}
	if err := json.NewEncoder(w).Encode(items); err != nil {
		t.Errorf("encode mappings: %v", err)
	}
}

func testSAMLGroupOperation(operation string) func(*schema.ResourceData, interface{}) error {
	switch operation {
	case "create":
		return samlCreate
	case "delete":
		return samlDelete
	default:
		return samlRead
	}
}

func TestSAMLGroupSyncErrors(t *testing.T) {
	testSAMLGroupWaits(t)
	for _, operation := range []string{"create", "read", "delete"} {
		for _, failure := range []struct {
			name   string
			status int
			body   string
			want   string
		}{
			{"forbidden", http.StatusForbidden, `{"detail":"forbidden"}`, "forbidden"},
			{"invalid", http.StatusUnprocessableEntity, `{"detail":"team does not exist"}`, "team does not exist"},
			{"server", http.StatusInternalServerError, `{"detail":"server failure"}`, "server failure"},
			{"malformed", http.StatusOK, `{`, "API error: {"},
			{"missing headers", http.StatusOK, `[]`, "missing required pagination header"},
		} {
			t.Run(operation+"/"+failure.name, func(t *testing.T) {
				var mu sync.Mutex
				var lists int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					const base = "/orgs/organization/saml-group-sync/"
					switch {
					case operation == "create" && r.Method == http.MethodPost && r.URL.Path == base:
						w.WriteHeader(http.StatusCreated)
						if err := json.NewEncoder(w).Encode(testSAMLGroupMapping(testSAMLGroupConfig(), "target")); err != nil {
							t.Errorf("encode mapping: %v", err)
						}
					case operation == "delete" && r.Method == http.MethodDelete && r.URL.Path == base+"target/":
						w.WriteHeader(http.StatusNoContent)
					case r.Method == http.MethodGet && r.URL.Path == base:
						lists++
						w.WriteHeader(failure.status)
						fmt.Fprint(w, failure.body)
					default:
						t.Errorf("unexpected request: %s %s", r.Method, r.URL)
						http.Error(w, `{"detail":"unexpected request"}`, http.StatusBadRequest)
					}
				}))
				defer server.Close()
				d := schema.TestResourceDataRaw(t, resourceSAML().Schema, testSAMLGroupConfig())
				if operation != "create" {
					d.SetId("target")
				}
				err := testSAMLGroupOperation(operation)(d, testSAMLGroupProvider(server))
				if err == nil || !strings.Contains(err.Error(), failure.want) {
					t.Fatalf("error=%v, want %q", err, failure.want)
				}
				if d.Id() != "target" {
					t.Errorf("failure lost ID: %q", d.Id())
				}
				mu.Lock()
				defer mu.Unlock()
				if lists != 1 {
					t.Errorf("retried terminal list error: %d requests", lists)
				}
			})
		}
	}
}

func TestSAMLGroupSyncAbsent(t *testing.T) {
	testSAMLGroupWaits(t)
	for _, tc := range []struct {
		name, operation string
		status          int
		present         bool
		recover         bool
		wantError       bool
	}{
		{name: "create waits through 404", operation: "create", status: http.StatusNotFound, recover: true},
		{name: "create absent times out", operation: "create", status: http.StatusOK, wantError: true},
		{name: "read external deletion empty list", operation: "read", status: http.StatusOK},
		{name: "read external deletion 404", operation: "read", status: http.StatusNotFound},
		{name: "delete confirms list 404", operation: "delete", status: http.StatusNotFound},
		{name: "delete still present times out", operation: "delete", status: http.StatusOK, present: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			timeout, interval := 200*time.Millisecond, time.Second
			if tc.recover {
				timeout, interval = 5*time.Second, time.Millisecond
			}
			defaultCreationTimeout, defaultCreationInterval = timeout, interval
			defaultDeletionTimeout, defaultDeletionInterval = timeout, interval
			var mu sync.Mutex
			var lists int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				const base = "/orgs/organization/saml-group-sync/"
				switch {
				case tc.operation == "create" && r.Method == http.MethodPost && r.URL.Path == base:
					w.WriteHeader(http.StatusCreated)
					if err := json.NewEncoder(w).Encode(testSAMLGroupMapping(testSAMLGroupConfig(), "target")); err != nil {
						t.Errorf("encode mapping: %v", err)
					}
				case tc.operation == "delete" && r.Method == http.MethodDelete && r.URL.Path == base+"target/":
					w.WriteHeader(http.StatusNoContent)
				case r.Method == http.MethodGet && r.URL.Path == base:
					lists++
					if tc.recover && lists > 1 || tc.present {
						testSAMLGroupPage(t, w, 1, 1, testSAMLGroupMapping(testSAMLGroupConfig(), "target"))
					} else if tc.status == http.StatusNotFound {
						http.Error(w, `{"detail":"not found"}`, tc.status)
					} else {
						testSAMLGroupPage(t, w, 1, 1, testSAMLGroupMapping(testSAMLGroupConfig(), "unrelated"))
					}
				case tc.recover && r.Method == http.MethodGet && r.URL.Path == base+"status/":
					fmt.Fprint(w, `{"saml_group_sync_status":true}`)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					http.Error(w, `{"detail":"unexpected request"}`, http.StatusBadRequest)
				}
			}))
			defer server.Close()
			d := schema.TestResourceDataRaw(t, resourceSAML().Schema, testSAMLGroupConfig())
			if tc.operation != "create" {
				d.SetId("target")
			}
			err := testSAMLGroupOperation(tc.operation)(d, testSAMLGroupProvider(server))
			if tc.wantError {
				if !errors.Is(err, errTimedOut) {
					t.Fatalf("error=%v, want bounded absence timeout", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			wantID := "target"
			if tc.operation == "read" {
				wantID = ""
			}
			if d.Id() != wantID {
				t.Errorf("ID=%q, want %q", d.Id(), wantID)
			}
			mu.Lock()
			defer mu.Unlock()
			wantLists := 1
			if tc.recover {
				wantLists = 2
			}
			if lists != wantLists {
				t.Errorf("list requests=%d, want %d", lists, wantLists)
			}
		})
	}
}

func TestSAMLGroupSyncWriteErrors(t *testing.T) {
	testSAMLGroupWaits(t)
	for _, tc := range []struct {
		name, operation string
		status          int
		body, want      string
	}{
		{"create null", "create", http.StatusCreated, `null`, "permanent slug"},
		{"create missing slug", "create", http.StatusCreated, `{"idp_key":"groups","idp_value":"engineers","team":"team"}`, "permanent slug"},
		{"create denied", "create", http.StatusForbidden, `{"detail":"forbidden"}`, "403"},
		{"delete denied", "delete", http.StatusForbidden, `{"detail":"forbidden"}`, "403"},
		{"delete already gone", "delete", http.StatusNotFound, `{"detail":"not found"}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				const base = "/orgs/organization/saml-group-sync/"
				method, path := http.MethodPost, base
				if tc.operation == "delete" {
					method, path = http.MethodDelete, base+"target/"
				}
				if r.Method != method || r.URL.Path != path {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					http.Error(w, `{"detail":"unexpected request"}`, http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			d := schema.TestResourceDataRaw(t, resourceSAML().Schema, testSAMLGroupConfig())
			wantID := ""
			if tc.operation == "delete" {
				wantID = "target"
				d.SetId(wantID)
			}
			err := testSAMLGroupOperation(tc.operation)(d, testSAMLGroupProvider(server))
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
			if d.Id() != wantID {
				t.Errorf("ID=%q, want %q", d.Id(), wantID)
			}
		})
	}
}

func TestSAMLGroupSyncEnabled(t *testing.T) {
	testSAMLGroupWaits(t)
	for _, operation := range []string{"read", "set"} {
		for _, tc := range []struct {
			name, body string
			enabled    bool
			status     int
			recover    bool
			wantError  string
		}{
			{name: "disabled", body: `{"saml_group_sync_status":false}`, status: http.StatusOK},
			{name: "enabled", body: `{"saml_group_sync_status":true}`, enabled: true, status: http.StatusOK},
			{name: "status 404", body: `{"detail":"not found"}`, enabled: true, status: http.StatusNotFound, recover: true},
			{name: "null status", body: `null`, status: http.StatusOK, wantError: "missing its status"},
			{name: "missing status", body: `{}`, status: http.StatusOK, wantError: "missing its status"},
			{name: "forbidden status", body: `{"detail":"forbidden"}`, status: http.StatusForbidden, wantError: "forbidden"},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				var mu sync.Mutex
				var reads, writes int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					const base = "/orgs/organization/saml-group-sync/"
					setPath := base + "disable/"
					if tc.enabled {
						setPath = base + "enable/"
					}
					switch {
					case operation == "set" && r.Method == http.MethodPost && r.URL.Path == setPath:
						writes++
						w.WriteHeader(http.StatusNoContent)
					case r.Method == http.MethodGet && r.URL.Path == base+"status/":
						reads++
						if tc.recover && reads > 1 {
							fmt.Fprint(w, `{"saml_group_sync_status":true}`)
						} else {
							w.WriteHeader(tc.status)
							fmt.Fprint(w, tc.body)
						}
					default:
						t.Errorf("unexpected request: %s %s", r.Method, r.URL)
						http.Error(w, `{"detail":"unexpected request"}`, http.StatusBadRequest)
					}
				}))
				defer server.Close()
				pc := testSAMLGroupProvider(server)
				var err error
				if operation == "read" {
					var enabled bool
					enabled, err = samlReadEnabled(pc, "organization")
					if err == nil && enabled != tc.enabled {
						t.Errorf("enabled=%t, want %t", enabled, tc.enabled)
					}
				} else {
					config := testSAMLGroupConfig()
					config["enabled"] = tc.enabled
					d := schema.TestResourceDataRaw(t, resourceSAML().Schema, config)
					err = samlSetEnabled(d, pc)
				}
				if tc.wantError == "" {
					if err != nil {
						t.Fatal(err)
					}
				} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error=%v, want %q", err, tc.wantError)
				}
				mu.Lock()
				defer mu.Unlock()
				wantReads, wantWrites := 1, 0
				if tc.recover {
					wantReads = 2
				}
				if operation == "set" {
					wantWrites = 1
				}
				if reads != wantReads || writes != wantWrites {
					t.Errorf("status reads=%d writes=%d, want %d/%d", reads, writes, wantReads, wantWrites)
				}
			})
		}
	}
}

type samlGroupErrorTransport struct{ err error }

func (transport samlGroupErrorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, transport.err
}

func TestSAMLGroupSyncTransportErrors(t *testing.T) {
	testSAMLGroupWaits(t)
	for _, operation := range []string{"create", "read", "delete", "status"} {
		t.Run(operation, func(t *testing.T) {
			sentinel := errors.New("transport failure without HTTP response")
			config := cloudsmith.NewConfiguration()
			config.Servers = cloudsmith.ServerConfigurations{{URL: "https://unused.invalid"}}
			config.HTTPClient = &http.Client{Transport: samlGroupErrorTransport{err: sentinel}}
			pc := &providerConfig{Auth: context.Background(), APIClient: cloudsmith.NewAPIClient(config)}
			d := schema.TestResourceDataRaw(t, resourceSAML().Schema, testSAMLGroupConfig())
			wantID := ""
			if operation != "create" {
				wantID = "target"
				d.SetId(wantID)
			}
			var err error
			if operation == "status" {
				_, err = samlReadEnabled(pc, "organization")
			} else {
				err = testSAMLGroupOperation(operation)(d, pc)
			}
			if !errors.Is(err, sentinel) || d.Id() != wantID {
				t.Fatalf("error=%v ID=%q, want transport error and ID=%q", err, d.Id(), wantID)
			}
		})
	}
}

func TestSAMLGroupSyncCancellation(t *testing.T) {
	testSAMLGroupWaits(t)
	for _, operation := range []string{"create", "read", "delete"} {
		t.Run(operation, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				const base = "/orgs/organization/saml-group-sync/"
				switch {
				case operation == "create" && r.Method == http.MethodPost && r.URL.Path == base:
					w.WriteHeader(http.StatusCreated)
					if err := json.NewEncoder(w).Encode(testSAMLGroupMapping(testSAMLGroupConfig(), "target")); err != nil {
						t.Errorf("encode mapping: %v", err)
					}
				case operation == "delete" && r.Method == http.MethodDelete && r.URL.Path == base+"target/":
					w.WriteHeader(http.StatusNoContent)
				case r.Method == http.MethodGet && r.URL.Path == base:
					close(started)
					<-release
					// Never let a canceled handler implicitly return a successful empty HTTP 200.
					http.Error(w, `{"detail":"canceled"}`, http.StatusServiceUnavailable)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					http.Error(w, `{"detail":"unexpected request"}`, http.StatusBadRequest)
				}
			}))
			defer server.Close()
			defer close(release)
			pc := testSAMLGroupProvider(server)
			ctx, cancel := context.WithCancel(pc.Auth)
			defer cancel()
			pc.Auth = ctx
			d := schema.TestResourceDataRaw(t, resourceSAML().Schema, testSAMLGroupConfig())
			if operation != "create" {
				d.SetId("target")
			}
			done := make(chan error, 1)
			go func() { done <- testSAMLGroupOperation(operation)(d, pc) }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("list request did not start")
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) || d.Id() != "target" {
					t.Fatalf("canceled request: error=%v ID=%q", err, d.Id())
				}
			case <-time.After(5 * time.Second):
				t.Fatal("canceled operation did not stop")
			}
		})
	}
}
func TestSAMLGroupSyncReadDeadline(t *testing.T) {
	testSAMLGroupWaits(t)
	defaultCreationTimeout = 200 * time.Millisecond
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/orgs/organization/saml-group-sync/" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			http.Error(w, `{"detail":"unexpected request"}`, http.StatusBadRequest)
			return
		}
		<-release
		http.Error(w, `{"detail":"deadline exceeded"}`, http.StatusServiceUnavailable)
	}))
	defer server.Close()
	defer close(release)
	d := schema.TestResourceDataRaw(t, resourceSAML().Schema, testSAMLGroupConfig())
	d.SetId("target")
	err := samlRead(d, testSAMLGroupProvider(server))
	if !errors.Is(err, context.DeadlineExceeded) || d.Id() != "target" {
		t.Fatalf("slow HTTP response treated as deletion: error=%v ID=%q", err, d.Id())
	}
}

func TestSAMLGroupSyncExternalDrift(t *testing.T) {
	testSAMLGroupWaits(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/organization/saml-group-sync/":
			config := testSAMLGroupConfig()
			config["role"] = "Manager"
			testSAMLGroupPage(t, w, 1, 1, testSAMLGroupMapping(config, "target"))
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/organization/saml-group-sync/status/":
			fmt.Fprint(w, `{"saml_group_sync_status":true}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			http.Error(w, `{"detail":"unexpected request"}`, http.StatusBadRequest)
		}
	}))
	defer server.Close()
	pc, res := testSAMLGroupProvider(server), resourceSAML()
	d := schema.TestResourceDataRaw(t, res.Schema, testSAMLGroupConfig())
	d.SetId("target")
	state, diagnostics := res.RefreshWithoutUpgrade(context.Background(), d.State(), pc)
	if diagnostics.HasError() || state == nil || state.ID != "target" || state.Attributes["role"] != "Manager" {
		t.Fatalf("refresh did not observe external drift: state=%#v diagnostics=%v", state, diagnostics)
	}
	diff, err := res.Diff(context.Background(), state, terraform.NewResourceConfigRaw(testSAMLGroupConfig()), pc)
	if err != nil {
		t.Fatal(err)
	}
	role := diff.Attributes["role"]
	if diff.Empty() || role == nil || role.Old != "Manager" || role.New != "Member" {
		t.Fatalf("external drift was hidden: %#v", diff)
	}
}

func TestSAMLGroupSyncLifecycle(t *testing.T) {
	testSAMLGroupWaits(t)
	for _, change := range []string{"create", "role", "role delayed create", "idp", "enabled only"} {
		t.Run(change, func(t *testing.T) {
			initial, desired := testSAMLGroupConfig(), testSAMLGroupConfig()
			enabled := true
			switch change {
			case "role", "role delayed create":
				desired["role"] = "Manager"
			case "idp":
				desired["idp_key"], desired["idp_value"], desired["enabled"] = "roles", "managers", false
			case "enabled only":
				enabled = false
				desired["enabled"] = true
			}
			initial["enabled"] = enabled
			currentID := "old-mapping"
			if change == "create" {
				currentID = ""
			}
			var mu sync.Mutex
			var creates, deletes, statusWrites, createScans, deleteScans int
			var deleting, created bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				const base = "/orgs/organization/saml-group-sync/"
				switch {
				case r.Method == http.MethodPost && r.URL.Path == base:
					creates++
					if currentID != "" {
						t.Errorf("POST before old target was absent: id=%s deletion scans=%d", currentID, deleteScans)
						http.Error(w, `{"detail":"old mapping still visible"}`, http.StatusConflict)
						return
					}
					var body map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode mapping: %v", err)
						http.Error(w, `{"detail":"invalid request"}`, http.StatusBadRequest)
						return
					}
					for _, key := range []string{"organization", "idp_key", "idp_value", "role", "team"} {
						if body[key] != desired[key] {
							t.Errorf("POST %s=%v, want %v", key, body[key], desired[key])
						}
					}
					currentID, created = "new-mapping", true
					w.WriteHeader(http.StatusCreated)
					if err := json.NewEncoder(w).Encode(testSAMLGroupMapping(desired, currentID)); err != nil {
						t.Errorf("encode mapping: %v", err)
					}
				case r.Method == http.MethodDelete && r.URL.Path == base+currentID+"/" && currentID != "":
					deletes++
					deleting, deleteScans = true, 0
					if change == "role delayed create" && currentID == "old-mapping" {
						currentID, deleting = "", false
					}
					w.WriteHeader(http.StatusNoContent)
				case r.Method == http.MethodGet && r.URL.Path == base:
					page := 1
					if raw := r.URL.Query().Get("page"); raw != "" {
						var err error
						page, err = strconv.Atoi(raw)
						if err != nil || page < 1 || page > 2 {
							t.Errorf("invalid page: %s", raw)
							http.Error(w, `{"detail":"invalid page"}`, http.StatusBadRequest)
							return
						}
					}
					if page == 1 {
						if deleting {
							deleteScans++
						} else if created {
							createScans++
						}
						// A matching body with a different permanent ID must not satisfy the waiter.
						testSAMLGroupPage(t, w, 1, 2, testSAMLGroupMapping(desired, "unrelated-mapping"))
						return
					}
					if deleting {
						if deleteScans <= 2 {
							testSAMLGroupPage(t, w, 2, 2, testSAMLGroupMapping(initial, currentID))
						} else {
							currentID, deleting = "", false
							testSAMLGroupPage(t, w, 2, 2)
						}
					} else if currentID == "" || (created && (createScans <= 2 || createScans == 4)) {
						// Delayed visibility, then a stale replica after the first observed target.
						testSAMLGroupPage(t, w, 2, 2)
					} else {
						testSAMLGroupPage(t, w, 2, 2, testSAMLGroupMapping(desired, currentID))
					}
				case r.Method == http.MethodPost && (r.URL.Path == base+"enable/" || r.URL.Path == base+"disable/"):
					statusWrites++
					enabled = r.URL.Path == base+"enable/"
					w.WriteHeader(http.StatusNoContent)
				case r.Method == http.MethodGet && r.URL.Path == base+"status/":
					fmt.Fprintf(w, `{"saml_group_sync_status":%t}`, enabled)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					http.Error(w, `{"detail":"unexpected request"}`, http.StatusBadRequest)
				}
			}))
			defer server.Close()
			pc, res := testSAMLGroupProvider(server), resourceSAML()
			var state *terraform.InstanceState
			if change != "create" {
				d := schema.TestResourceDataRaw(t, res.Schema, initial)
				d.SetId("old-mapping")
				if err := d.Set("slug_perm", "old-mapping"); err != nil {
					t.Fatal(err)
				}
				state = d.State()
			}
			config := terraform.NewResourceConfigRaw(desired)
			diff, err := res.Diff(context.Background(), state, config, pc)
			if err != nil {
				t.Fatal(err)
			}
			if diff.Empty() || (change != "create" && diff.RequiresNew()) {
				t.Fatal("expected an in-place change, or initial creation")
			}
			// The protocol server supplies RawConfig; direct SDK Diff does not.
			raw := map[string]cty.Value{}
			for key, typ := range res.CoreConfigSchema().ImpliedType().AttributeTypes() {
				raw[key] = cty.NullVal(typ)
				switch value := desired[key].(type) {
				case string:
					raw[key] = cty.StringVal(value)
				case bool:
					raw[key] = cty.BoolVal(value)
				}
			}
			diff.RawConfig = cty.ObjectVal(raw)
			state, diagnostics := res.Apply(context.Background(), state, diff, pc)
			if diagnostics.HasError() {
				t.Fatalf("Apply: %v", diagnostics)
			}
			checkState := func(phase string) {
				t.Helper()
				if state == nil || state.ID == "" {
					t.Fatalf("%s: ROOT present -> absent", phase)
				}
				wantID := "new-mapping"
				if change == "enabled only" {
					wantID = "old-mapping"
				}
				if state.ID != wantID || state.Attributes["slug_perm"] != wantID {
					t.Errorf("%s identity: %#v", phase, state)
				}
				mu.Lock()
				wantEnabled := enabled
				mu.Unlock()
				if state.Attributes["enabled"] != strconv.FormatBool(wantEnabled) {
					t.Errorf("%s enabled=%s, want %t", phase, state.Attributes["enabled"], wantEnabled)
				}
				diff, err := res.Diff(context.Background(), state, config, pc)
				if err != nil {
					t.Fatal(err)
				}
				if !diff.Empty() {
					t.Fatalf("%s: nonempty plan: %#v", phase, diff.Attributes)
				}
			}
			checkState("post-Apply")
			state, diagnostics = res.RefreshWithoutUpgrade(context.Background(), state, pc)
			if diagnostics.HasError() {
				t.Fatalf("Refresh: %v", diagnostics)
			}
			checkState("post-Refresh")
			state, diagnostics = res.Apply(context.Background(), state, &terraform.InstanceDiff{Destroy: true}, pc)
			if diagnostics.HasError() || (state != nil && state.ID != "") {
				t.Fatalf("cleanup: state=%#v diagnostics=%v", state, diagnostics)
			}
			mu.Lock()
			defer mu.Unlock()
			wantCreates, wantDeletes, wantStatusWrites := 1, 2, 0
			switch change {
			case "create":
				wantDeletes = 1
			case "idp":
				wantStatusWrites = 1
			case "enabled only":
				wantCreates, wantDeletes, wantStatusWrites = 0, 1, 1
			}
			if creates != wantCreates || deletes != wantDeletes || statusWrites != wantStatusWrites || currentID != "" {
				t.Errorf("requests: POST=%d DELETE=%d status writes=%d remaining ID=%q", creates, deletes, statusWrites, currentID)
			}
			if creates != 0 && createScans != 5 {
				t.Errorf("create/refresh scans=%d, want 5 including stale-after-visible", createScans)
			}
		})
	}
}
