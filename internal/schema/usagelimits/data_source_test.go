// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package usagelimits_test

import (
	"context"
	"fmt"
	. "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/usagelimits"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/util"
	"net/http"
	"net/http/httptest"
	"testing"

	cloudsmithapi "github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestDataSourceUsageLimitsSchema(t *testing.T) {
	t.Parallel()

	dataSource := DataSourceUsageLimits()
	if !dataSource.Schema[UsageLimitsOrganization].Required {
		t.Fatal("expected organization to be required")
	}
	for _, name := range []string{
		UsageLimitsAllowOpenSourceOverage,
		UsageLimitsBandwidthOverageLimit,
		UsageLimitsStorageOverageLimit,
		UsageLimitsBandwidthMaximum,
		UsageLimitsStorageMaximum,
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

	d := schema.TestResourceDataRaw(t, DataSourceUsageLimits().Schema, map[string]interface{}{
		UsageLimitsOrganization: "example-org",
	})
	diagnostics := DataSourceUsageLimitsRead(context.Background(), d, usageLimitsTestProviderConfig(server.URL))
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	assertUsageLimitsState(t, d)
}

func usageLimitsTestProviderConfig(serverURL string) *util.ProviderConfig {
	config := cloudsmithapi.NewConfiguration()
	config.Servers = cloudsmithapi.ServerConfigurations{{URL: serverURL}}
	return &util.ProviderConfig{
		Auth:      context.Background(),
		APIClient: cloudsmithapi.NewAPIClient(config),
	}
}

func assertUsageLimitsState(t *testing.T, d *schema.ResourceData) {
	t.Helper()

	expected := map[string]interface{}{
		UsageLimitsAllowOpenSourceOverage: true,
		UsageLimitsBandwidthOverageLimit:  100,
		UsageLimitsStorageOverageLimit:    50,
		UsageLimitsBandwidthMaximum:       200,
		UsageLimitsStorageMaximum:         150,
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
