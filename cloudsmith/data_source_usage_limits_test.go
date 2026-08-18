package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	cloudsmithapi "github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestDataSourceUsageLimitsSchema(t *testing.T) {
	t.Parallel()

	dataSource := dataSourceUsageLimits()
	if !dataSource.Schema[usageLimitsOrganization].Required {
		t.Fatal("expected organization to be required")
	}
	for _, name := range []string{
		usageLimitsAllowOpenSourceOverage,
		usageLimitsBandwidthOverageLimit,
		usageLimitsStorageOverageLimit,
		usageLimitsBandwidthMaximum,
		usageLimitsStorageMaximum,
	} {
		if !dataSource.Schema[name].Computed {
			t.Fatalf("expected %q to be computed", name)
		}
	}
}

func TestDataSourceUsageLimitsRead(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/orgs/example-org/usage-limits/" {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"allow_open_source_overage":true,"bandwidth_overage_limit":100,"storage_overage_limit":50,"bandwidth_maximum":200,"storage_maximum":150}`)
	}))
	defer server.Close()

	d := schema.TestResourceDataRaw(t, dataSourceUsageLimits().Schema, map[string]interface{}{
		usageLimitsOrganization: "example-org",
	})
	diagnostics := dataSourceUsageLimitsRead(context.Background(), d, usageLimitsTestProviderConfig(server.URL))
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	assertUsageLimitsState(t, d)
}

func usageLimitsTestProviderConfig(serverURL string) *providerConfig {
	config := cloudsmithapi.NewConfiguration()
	config.Servers = cloudsmithapi.ServerConfigurations{{URL: serverURL}}
	return &providerConfig{
		Auth:      context.Background(),
		APIClient: cloudsmithapi.NewAPIClient(config),
	}
}

func assertUsageLimitsState(t *testing.T, d *schema.ResourceData) {
	t.Helper()

	expected := map[string]interface{}{
		usageLimitsAllowOpenSourceOverage: true,
		usageLimitsBandwidthOverageLimit:  100,
		usageLimitsStorageOverageLimit:    50,
		usageLimitsBandwidthMaximum:       200,
		usageLimitsStorageMaximum:         150,
	}
	for name, want := range expected {
		if got := d.Get(name); got != want {
			t.Errorf("unexpected %s: got %v, want %v", name, got, want)
		}
	}
	if d.Id() != "example-org" {
		t.Errorf("unexpected ID: got %q, want %q", d.Id(), "example-org")
	}
}
