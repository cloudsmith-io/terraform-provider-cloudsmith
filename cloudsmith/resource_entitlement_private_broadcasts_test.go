//nolint:testpackage
package cloudsmith

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cloudsmithapi "github.com/cloudsmith-io/cloudsmith-api-go"
)

func TestSetEntitlementPrivateBroadcasts(t *testing.T) {
	t.Parallel()

	var (
		requestBody string
		requestPath string
		requestAuth string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		requestAuth = r.Header.Get("X-Api-Key")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		requestBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_private_broadcasts":true}`))
	}))
	defer server.Close()

	config := cloudsmithapi.NewConfiguration()
	config.Servers = cloudsmithapi.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()
	config.UserAgent = "terraform-provider-cloudsmith-test"

	pc := &providerConfig{
		APIClient: cloudsmithapi.NewAPIClient(config),
		Auth: context.WithValue(
			context.Background(),
			cloudsmithapi.ContextAPIKeys,
			map[string]cloudsmithapi.APIKey{
				"apikey": {Key: "test-api-key"},
			},
		),
	}

	if err := setEntitlementPrivateBroadcasts(pc, "org", "repo", "token", true); err != nil {
		t.Fatalf("setEntitlementPrivateBroadcasts() error = %v", err)
	}

	if requestPath != "/entitlements/org/repo/token/toggle-private-broadcasts/" {
		t.Fatalf("unexpected request path %q", requestPath)
	}
	if requestAuth != "test-api-key" {
		t.Fatalf("unexpected api key %q", requestAuth)
	}
	if strings.TrimSpace(requestBody) != `{"access_private_broadcasts":true}` {
		t.Fatalf("unexpected request body %q", requestBody)
	}
}

func TestSetEntitlementPrivateBroadcasts_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "payment required", http.StatusPaymentRequired)
	}))
	defer server.Close()

	config := cloudsmithapi.NewConfiguration()
	config.Servers = cloudsmithapi.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	pc := &providerConfig{
		APIClient: cloudsmithapi.NewAPIClient(config),
		Auth: context.WithValue(
			context.Background(),
			cloudsmithapi.ContextAPIKeys,
			map[string]cloudsmithapi.APIKey{
				"apikey": {Key: "test-api-key"},
			},
		),
	}

	err := setEntitlementPrivateBroadcasts(pc, "org", "repo", "token", true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "402") {
		t.Fatalf("expected status in error, got %q", err)
	}
	if !strings.Contains(err.Error(), "Payment Required") {
		t.Fatalf("expected status phrase in error, got %q", err)
	}
}
