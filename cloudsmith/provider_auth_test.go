package cloudsmith

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOIDCTokenSourceCachesAndRefreshesBeforeExpiry(t *testing.T) {
	base := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	now := base
	tokens := []string{
		testJWT(t, base.Add(10*time.Minute)),
		testJWT(t, base.Add(30*time.Minute)),
	}
	var exchanges atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := int(exchanges.Add(1))
		if call > len(tokens) {
			http.Error(w, "unexpected exchange", http.StatusInternalServerError)
			return
		}
		writeOIDCToken(t, w, tokens[call-1])
	}))
	defer server.Close()
	source := testOIDCTokenSource(server.URL+"/v1", func() time.Time { return now })

	first, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("first token: %v", err)
	}
	second, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("cached token: %v", err)
	}
	if first != second || exchanges.Load() != 1 {
		t.Fatalf("cached token = %q after %d exchanges, want %q after 1", second, exchanges.Load(), first)
	}

	now = base.Add(9 * time.Minute)
	refreshed, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("refreshed token: %v", err)
	}
	if refreshed != tokens[1] || exchanges.Load() != 2 {
		t.Fatalf("refreshed token = %q after %d exchanges, want second token after 2", refreshed, exchanges.Load())
	}
}

func TestOIDCTokenSourceRetriesAfterFailedExchange(t *testing.T) {
	base := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	wantToken := testJWT(t, base.Add(10*time.Minute))
	var exchanges atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if exchanges.Add(1) == 1 {
			http.Error(w, "exchange failed", http.StatusBadGateway)
			return
		}
		writeOIDCToken(t, w, wantToken)
	}))
	defer server.Close()
	source := testOIDCTokenSource(server.URL+"/v1", func() time.Time { return base })

	if _, err := source.Token(context.Background()); err == nil {
		t.Fatal("first token exchange succeeded, want error")
	}
	got, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("second token exchange: %v", err)
	}
	if got != wantToken || exchanges.Load() != 2 {
		t.Fatalf("token = %q after %d exchanges, want %q after 2", got, exchanges.Load(), wantToken)
	}
}

func TestOIDCTokenSourceSerializesConcurrentRefresh(t *testing.T) {
	base := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	wantToken := testJWT(t, base.Add(10*time.Minute))
	var exchanges atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		exchanges.Add(1)
		writeOIDCToken(t, w, wantToken)
	}))
	defer server.Close()
	source := testOIDCTokenSource(server.URL+"/v1", func() time.Time { return base })

	const callers = 12
	start := make(chan struct{})
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			token, err := source.Token(context.Background())
			if err == nil && token != wantToken {
				err = fmt.Errorf("token = %q, want %q", token, wantToken)
			}
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	if exchanges.Load() != 1 {
		t.Errorf("exchanges = %d, want 1", exchanges.Load())
	}
}

func TestOIDCTokenSourceIgnoresConcurrentStaleInvalidation(t *testing.T) {
	base := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	wantToken := testJWT(t, base.Add(10*time.Minute))
	source := &oidcTokenSource{
		cached: wantToken,
		expiry: base.Add(10 * time.Minute),
		now:    func() time.Time { return base },
	}

	const callers = 12
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			source.invalidate("stale-token")
		}()
	}
	wg.Wait()

	got, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("token after stale invalidation: %v", err)
	}
	if got != wantToken {
		t.Errorf("token = %q, want cached token %q", got, wantToken)
	}
}

func testOIDCTokenSource(apiHost string, now func() time.Time) *oidcTokenSource {
	return &oidcTokenSource{
		identity: oidcIdentity{
			organization: "test-org",
			serviceSlug:  "test-service",
		},
		apiHost: apiHost,
		getenv: func(key string) string {
			if key == "TFC_WORKLOAD_IDENTITY_TOKEN_CLOUDSMITH" {
				return "workload-identity-token"
			}
			return ""
		},
		now: now,
	}
}

func testJWT(t *testing.T, expiry time.Time) string {
	t.Helper()
	payload, err := json.Marshal(map[string]int64{"exp": expiry.Unix()})
	if err != nil {
		t.Fatalf("marshal JWT payload: %v", err)
	}
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func writeOIDCToken(t *testing.T, w http.ResponseWriter, token string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"token": token}); err != nil {
		t.Errorf("write OIDC response: %v", err)
	}
}
