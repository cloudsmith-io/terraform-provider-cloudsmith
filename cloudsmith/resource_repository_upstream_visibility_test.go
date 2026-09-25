// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package cloudsmith

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestRepositoryUpstreamNixRefreshVisibility(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		missingReads int
		staleFirst   bool
	}{
		{missingReads: 1},
		{missingReads: 1, staleFirst: true},
		{missingReads: 2},
		{missingReads: 3},
	} {
		t.Run(fmt.Sprintf("missing_%d_stale_first_%t", tc.missingReads, tc.staleFirst), func(t *testing.T) {
			t.Parallel()
			previous := testUpstreamResponse(Nix, false)
			updated := testUpstreamResponse(Nix, true)
			config := testUpstreamUpdateConfig(Nix)
			config[UpstreamUrl] = "https://channels.nixos.org/nixos-25.05"
			previous[UpstreamUrl], updated[UpstreamUrl] = config[UpstreamUrl], config[UpstreamUrl]

			var mu sync.Mutex
			var reads, writes, deletes int
			notFoundRead := 2
			if tc.staleFirst {
				notFoundRead++
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path != "/repos/example-org/example-repo/upstream/nix/upstream-id/" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected path", http.StatusBadRequest)
					return
				}
				response := updated
				switch r.Method {
				case http.MethodPut:
					writes++
				case http.MethodGet:
					reads++
					if (reads >= notFoundRead && reads < notFoundRead+tc.missingReads) || deletes > 0 {
						http.Error(w, `{"detail":"Not found"}`, http.StatusNotFound)
						return
					}
					if tc.staleFirst && reads == 2 {
						response = previous
					}
				case http.MethodDelete:
					deletes++
					w.WriteHeader(http.StatusNoContent)
					return
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

			pc := testUpstreamProviderConfig(server)
			res := resourceRepositoryUpstream()
			previous[Namespace], previous[Repository], previous[UpstreamType] = "example-org", "example-repo", Nix
			d := schema.TestResourceDataRaw(t, res.Schema, previous)
			d.SetId("upstream-id")
			resourceConfig := terraform.NewResourceConfigRaw(config)
			diff, err := res.Diff(context.Background(), d.State(), resourceConfig, pc)
			if err != nil || diff.Empty() || diff.RequiresNew() {
				t.Fatalf("expected in-place update: diff=%v err=%v", diff, err)
			}
			state, diagnostics := res.Apply(context.Background(), d.State(), diff, pc)
			if diagnostics.HasError() {
				t.Fatalf("apply: %v", diagnostics)
			}
			state, diagnostics = res.RefreshWithoutUpgrade(context.Background(), state, pc)
			if diagnostics.HasError() {
				t.Fatalf("refresh: %v", diagnostics)
			}
			diff, err = res.Diff(context.Background(), state, resourceConfig, pc)
			if err != nil {
				t.Fatal(err)
			}
			if state == nil || state.ID == "" {
				// This is the acceptance CheckDestroy error after refresh drops
				// the resource from Terraform's state, not a remote DELETE error.
				cleanupErr := testAccRepositoryUpstreamCheckDestroy("cloudsmith_repository_upstream.nix_channels")(terraform.NewState())
				t.Fatalf("unconfirmed 404 lost resource: recreation plan=%t; cleanup=%v", !diff.Empty(), cleanupErr)
			}
			if !diff.Empty() {
				t.Fatal("refresh must not plan recreation or changes after a transient 404")
			}
			if state.Attributes[UpdatedAt] != updated[UpdatedAt] || state.Attributes[AuthSecret] != config[AuthSecret] {
				t.Fatal("refresh lost the accepted revision or redacted secret")
			}
			_, diagnostics = res.Apply(context.Background(), state, &terraform.InstanceDiff{Destroy: true}, pc)
			if diagnostics.HasError() {
				t.Fatalf("destroy: %v", diagnostics)
			}
			mu.Lock()
			defer mu.Unlock()
			if writes != 1 || deletes != 1 || reads != notFoundRead+tc.missingReads+1 {
				t.Errorf("requests: PUT=%d DELETE=%d GET=%d, want 1/1/%d", writes, deletes, reads, notFoundRead+tc.missingReads+1)
			}
		})
	}
}

func TestRepositoryUpstreamReadAbsenceConfirmation(t *testing.T) {
	previousTimeout, previousInterval := defaultUpdateTimeout, defaultUpdateInterval
	t.Cleanup(func() {
		defaultUpdateTimeout, defaultUpdateInterval = previousTimeout, previousInterval
	})

	const stale = 0
	for _, tc := range []struct {
		name     string
		statuses []int
		wantErr  string
		deleted  bool
		legacy   bool
		timeout  bool
		single   bool
	}{
		{name: "visible again", statuses: []int{404, 200}},
		{name: "two missing responses", statuses: []int{404, 404, 200}},
		{name: "three missing responses", statuses: []int{404, 404, 404, 200}},
		{name: "legacy state", statuses: []int{404, 200}, legacy: true},
		{name: "sustained deletion", statuses: []int{404}, deleted: true},
		{name: "older response resets absence", statuses: []int{404, stale, 404, 200}},
		{name: "absence after older response is unconfirmed", statuses: []int{404, stale, 404}, timeout: true},
		{name: "unauthorized", statuses: []int{404, 401}, wantErr: "401"},
		{name: "forbidden", statuses: []int{404, 403}, wantErr: "403"},
		{name: "rate limited", statuses: []int{404, 429}, wantErr: "429"},
		{name: "server failure", statuses: []int{404, 500}, wantErr: "500"},
		{name: "unavailable", statuses: []int{404, 503}, wantErr: "503"},
		{name: "timeout before confirmation", statuses: []int{404}, timeout: true, single: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaultUpdateTimeout, defaultUpdateInterval = 5*time.Second, time.Millisecond
			if tc.deleted || tc.timeout {
				defaultUpdateTimeout, defaultUpdateInterval = 200*time.Millisecond, 20*time.Millisecond
			}
			if tc.single {
				defaultUpdateInterval = time.Second
			}
			var mu sync.Mutex
			reads := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				reads++
				if r.Method != http.MethodGet || r.URL.Path != "/repos/example-org/example-repo/upstream/nix/upstream-id/" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusBadRequest)
					return
				}
				if reads > len(tc.statuses) && !tc.deleted && !tc.timeout {
					t.Error("retried after a conclusive response")
					http.Error(w, "unexpected retry", http.StatusInternalServerError)
					return
				}
				status := tc.statuses[min(reads-1, len(tc.statuses)-1)]
				w.Header().Set("Content-Type", "application/json")
				if status != http.StatusOK && status != stale {
					http.Error(w, `{"detail":"API failure"}`, status)
					return
				}
				if err := json.NewEncoder(w).Encode(testUpstreamResponse(Nix, status == http.StatusOK)); err != nil {
					t.Errorf("encode upstream: %v", err)
				}
			}))
			defer server.Close()
			d := schema.TestResourceDataRaw(t, resourceRepositoryUpstream().Schema, testUpstreamUpdateConfig(Nix))
			d.SetId("upstream-id")
			prior := "2026-09-29T12:00:02.123456Z"
			if tc.legacy {
				prior = "2026-09-29T12:00:02Z"
			}
			if err := d.Set(UpdatedAt, prior); err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			err := resourceRepositoryUpstreamRead(d, testUpstreamProviderConfig(server))
			switch {
			case tc.timeout:
				if !errors.Is(err, errTimedOut) {
					t.Fatalf("error=%v, want polling timeout", err)
				}
			case tc.wantErr != "":
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error=%v, want %s", err, tc.wantErr)
				}
			case err != nil:
				t.Fatal(err)
			}
			if tc.deleted {
				if d.Id() != "" {
					t.Error("confirmed deletion must clear the resource")
				}
				if time.Since(start) < defaultUpdateTimeout {
					t.Error("deletion accepted before the complete visibility window")
				}
			} else if d.Id() != "upstream-id" {
				t.Error("unconfirmed absence, errors and timeouts must retain identity")
			}
			if err != nil && (d.Get(UpdatedAt) != prior || d.Get(AuthSecret) != "mock-secret") {
				t.Error("failed refresh must preserve prior revision and secret")
			}
			mu.Lock()
			defer mu.Unlock()
			if (tc.deleted || tc.timeout) && !tc.single {
				if reads < 3 {
					t.Errorf("reads=%d, expected polling across the visibility window", reads)
				}
			} else if reads != len(tc.statuses) {
				t.Errorf("reads=%d, want %d", reads, len(tc.statuses))
			}
		})
	}
}

func TestRepositoryUpstreamReadRequestDeadlines(t *testing.T) {
	previousTimeout, previousInterval := defaultUpdateTimeout, defaultUpdateInterval
	t.Cleanup(func() {
		defaultUpdateTimeout, defaultUpdateInterval = previousTimeout, previousInterval
	})

	for _, tc := range []struct {
		name         string
		firstMissing bool
		cancel       bool
		external     bool
	}{
		{name: "blocked initial GET"},
		{name: "blocked GET after 404", firstMissing: true},
		{name: "external deadline after 404", firstMissing: true, external: true},
		{name: "external cancellation after 404", firstMissing: true, cancel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaultUpdateTimeout, defaultUpdateInterval = 300*time.Millisecond, time.Millisecond
			ctx, cancel := context.WithCancel(context.Background())
			if tc.external {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 300*time.Millisecond)
			}
			if tc.external || tc.cancel {
				defaultUpdateTimeout = 5 * time.Second
			}
			defer cancel()
			release := make(chan struct{})
			var mu sync.Mutex
			reads := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				reads++
				read := reads
				mu.Unlock()
				if r.Header.Get("X-Api-Key") != "mock-key" {
					t.Error("bounded read lost its authentication context")
				}
				if read == 1 && tc.firstMissing {
					http.Error(w, `{"detail":"Not found"}`, http.StatusNotFound)
					return
				}
				if tc.cancel {
					cancel()
				}
				// Only test cleanup releases this handler. Client cancellation
				// must return without help from the server or counter mutex.
				<-release
				http.Error(w, "test handler released", http.StatusServiceUnavailable)
			}))
			defer server.Close()
			defer close(release)
			pc := testUpstreamProviderConfig(server)
			pc.Auth = context.WithValue(ctx, cloudsmith.ContextAPIKeys, map[string]cloudsmith.APIKey{
				"apikey": {Key: "mock-key"},
			})
			d := schema.TestResourceDataRaw(t, resourceRepositoryUpstream().Schema, testUpstreamUpdateConfig(Nix))
			d.SetId("upstream-id")
			const prior = "2026-09-29T12:00:02.123456Z"
			if err := d.Set(UpdatedAt, prior); err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			err := resourceRepositoryUpstreamRead(d, pc)
			want := context.DeadlineExceeded
			if tc.cancel {
				want = context.Canceled
			}
			if !errors.Is(err, want) {
				t.Fatalf("error=%v, want %v", err, want)
			}
			if time.Since(start) > 2*time.Second {
				t.Fatal("in-flight read was not bounded")
			}
			if d.Id() != "upstream-id" || d.Get(UpdatedAt) != prior || d.Get(AuthSecret) != "mock-secret" {
				t.Error("in-flight cancellation or deadline must preserve identity, revision and secret")
			}
			mu.Lock()
			defer mu.Unlock()
			wantReads := 1
			if tc.firstMissing {
				wantReads++
			}
			if reads != wantReads {
				t.Errorf("reads=%d, want %d", reads, wantReads)
			}
		})
	}
}

type upstreamObservedResponseBody struct {
	io.ReadCloser
	onClose func()
	once    sync.Once
}

func (body *upstreamObservedResponseBody) Close() error {
	err := body.ReadCloser.Close()
	body.once.Do(body.onClose)
	return err
}

func TestRepositoryUpstreamReadPollingCancellation(t *testing.T) {
	previousTimeout, previousInterval := defaultUpdateTimeout, defaultUpdateInterval
	t.Cleanup(func() {
		defaultUpdateTimeout, defaultUpdateInterval = previousTimeout, previousInterval
	})

	for _, tc := range []struct {
		name      string
		cancel    bool
		longSleep bool
	}{
		{name: "parent cancellation after repeated completed 404s", cancel: true},
		{name: "parent deadline after repeated completed 404s"},
		{name: "parent cancellation interrupts long polling sleep", cancel: true, longSleep: true},
		{name: "parent deadline interrupts long polling sleep", longSleep: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaultUpdateTimeout, defaultUpdateInterval = 5*time.Second, 300*time.Millisecond
			parentTimeout := 750 * time.Millisecond
			completedBeforeCancel := int32(2)
			if tc.longSleep {
				defaultUpdateInterval = 5 * time.Second
				parentTimeout = 300 * time.Millisecond
				completedBeforeCancel = 1
			}
			ctx, cancel := context.WithCancel(context.Background())
			if !tc.cancel {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), parentTimeout)
			}
			defer cancel()
			var requests, completed atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, `{"detail":"Not found"}`, http.StatusNotFound)
			}))
			defer server.Close()
			client := server.Client()
			transport := client.Transport
			client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests.Add(1)
				response, err := transport.RoundTrip(r)
				if err != nil {
					return response, err
				}
				response.Body = &upstreamObservedResponseBody{
					ReadCloser: response.Body,
					onClose: func() {
						if completed.Add(1) == completedBeforeCancel && tc.cancel {
							// The SDK has consumed and closed the 404 response.
							// Cancel well inside the following polling interval.
							timer := time.AfterFunc(100*time.Millisecond, cancel)
							t.Cleanup(func() { timer.Stop() })
						}
					},
				}
				return response, nil
			})
			pc := testUpstreamProviderConfig(server)
			pc.Auth = ctx
			d := schema.TestResourceDataRaw(t, resourceRepositoryUpstream().Schema, testUpstreamUpdateConfig(Nix))
			d.SetId("upstream-id")
			const prior = "2026-09-29T12:00:02.123456Z"
			if err := d.Set(UpdatedAt, prior); err != nil {
				t.Fatal(err)
			}

			start := time.Now()
			err := resourceRepositoryUpstreamRead(d, pc)
			want := context.DeadlineExceeded
			if tc.cancel {
				want = context.Canceled
			}
			if !errors.Is(err, want) {
				t.Fatalf("error=%v, want exact parent cause %v", err, want)
			}
			if time.Since(start) > 2*time.Second {
				t.Fatal("parent cancellation did not interrupt the polling sleep promptly")
			}
			if d.Id() != "upstream-id" || d.Get(UpdatedAt) != prior || d.Get(AuthSecret) != "mock-secret" {
				t.Error("parent cancellation between completed 404s must not confirm deletion or alter state")
			}
			if requests.Load() != completed.Load() || completed.Load() < completedBeforeCancel {
				t.Fatalf("requests=%d completed=%d; expected completed responses before cancellation", requests.Load(), completed.Load())
			}
			if (tc.cancel || tc.longSleep) && completed.Load() != completedBeforeCancel {
				t.Errorf("completed=%d, want %d before cancellation without another request", completed.Load(), completedBeforeCancel)
			}
		})
	}
}
