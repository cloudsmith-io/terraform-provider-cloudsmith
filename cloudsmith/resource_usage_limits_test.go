package cloudsmith

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceUsageLimitsCreate(t *testing.T) {
	t.Parallel()

	var patchBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/example-org/usage-limits/" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPatch:
			if err := json.NewDecoder(r.Body).Decode(&patchBody); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Fprint(w, `{"allow_open_source_overage":true,"bandwidth_overage_limit":100,"storage_overage_limit":50}`)
		case http.MethodGet:
			fmt.Fprint(w, `{"allow_open_source_overage":true,"bandwidth_overage_limit":100,"storage_overage_limit":50,"bandwidth_maximum":200,"storage_maximum":150}`)
		default:
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	d := schema.TestResourceDataRaw(t, resourceUsageLimits().Schema, map[string]interface{}{
		usageLimitsOrganization:           "example-org",
		usageLimitsAllowOpenSourceOverage: true,
		usageLimitsBandwidthOverageLimit:  100,
		usageLimitsStorageOverageLimit:    50,
	})
	diagnostics := resourceUsageLimitsCreate(context.Background(), d, usageLimitsTestProviderConfig(server.URL))
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	expectedPatch := map[string]interface{}{
		"allow_open_source_overage": true,
		"bandwidth_overage_limit":   float64(100),
		"storage_overage_limit":     float64(50),
	}
	for name, want := range expectedPatch {
		if got := patchBody[name]; got != want {
			t.Errorf("unexpected PATCH field %s: got %v, want %v", name, got, want)
		}
	}
	assertUsageLimitsState(t, d)
}

func TestResourceUsageLimitsSchema(t *testing.T) {
	t.Parallel()

	resource := resourceUsageLimits()

	for _, name := range []string{
		usageLimitsOrganization,
		usageLimitsAllowOpenSourceOverage,
		usageLimitsBandwidthOverageLimit,
		usageLimitsStorageOverageLimit,
	} {
		if !resource.Schema[name].Required {
			t.Fatalf("expected %q to be required", name)
		}
	}

	for _, name := range []string{usageLimitsBandwidthMaximum, usageLimitsStorageMaximum} {
		if !resource.Schema[name].Computed {
			t.Fatalf("expected %q to be computed", name)
		}
	}

	for _, name := range []string{usageLimitsBandwidthOverageLimit, usageLimitsStorageOverageLimit} {
		if warnings, errors := resource.Schema[name].ValidateFunc(-1, name); len(warnings) != 0 || len(errors) == 0 {
			t.Fatalf("expected %q to reject a negative value", name)
		}
		if warnings, errors := resource.Schema[name].ValidateFunc(0, name); len(warnings) != 0 || len(errors) != 0 {
			t.Fatalf("expected %q to accept zero, warnings: %v, errors: %v", name, warnings, errors)
		}
	}
}

func TestResourceUsageLimitsImport(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, resourceUsageLimits().Schema, nil)
	d.SetId("example-org")

	resources, err := resourceUsageLimitsImport(context.Background(), d, nil)
	if err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}
	if got := resources[0].Get(usageLimitsOrganization); got != "example-org" {
		t.Fatalf("expected imported organization to be example-org, got %q", got)
	}
}

func TestResourceUsageLimitsDeleteOnlyClearsState(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, resourceUsageLimits().Schema, nil)
	d.SetId("example-org")

	if diags := resourceUsageLimitsDelete(context.Background(), d, nil); diags.HasError() {
		t.Fatalf("unexpected delete diagnostics: %v", diags)
	}
	if d.Id() != "" {
		t.Fatalf("expected resource ID to be cleared, got %q", d.Id())
	}
}

func TestAccUsageLimits_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccUsageLimitsCheckDestroy("cloudsmith_usage_limits.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccUsageLimitsConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccUsageLimitsCheckExists("cloudsmith_usage_limits.test"),
					resource.TestCheckResourceAttr("cloudsmith_usage_limits.test", "allow_open_source_overage", "false"),
					resource.TestCheckResourceAttr("cloudsmith_usage_limits.test", "bandwidth_overage_limit", "0"),
					resource.TestCheckResourceAttr("cloudsmith_usage_limits.test", "storage_overage_limit", "0"),
					resource.TestCheckResourceAttrSet("cloudsmith_usage_limits.test", "bandwidth_maximum"),
					resource.TestCheckResourceAttrSet("cloudsmith_usage_limits.test", "storage_maximum"),
				),
			},
			{
				Config: testAccUsageLimitsConfigOpenSourceOverage,
				Check: resource.ComposeTestCheckFunc(
					testAccUsageLimitsCheckExists("cloudsmith_usage_limits.test"),
					resource.TestCheckResourceAttr("cloudsmith_usage_limits.test", "allow_open_source_overage", "true"),
				),
			},
			{
				ResourceName: "cloudsmith_usage_limits.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return os.Getenv("CLOUDSMITH_NAMESPACE"), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

// testAccUsageLimitsCheckDestroy asserts the destroy behaviour of this
// resource: Cloudsmith has no delete or reset operation for usage limits, so a
// destroy only drops Terraform state and the organization's settings must
// remain readable afterwards.
func testAccUsageLimitsCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		pc := testAccProvider.Meta().(*providerConfig)
		organization := resourceState.Primary.Attributes["organization"]

		if _, _, err := pc.APIClient.OrgsApi.OrgsRetrieveUsageLimits(pc.Auth, organization).Execute(); err != nil {
			return fmt.Errorf("error reading usage limits after destroy: %w", formatAPIError(err))
		}

		return nil
	}
}

func testAccUsageLimitsCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)
		organization := resourceState.Primary.Attributes["organization"]

		if _, _, err := pc.APIClient.OrgsApi.OrgsRetrieveUsageLimits(pc.Auth, organization).Execute(); err != nil {
			return fmt.Errorf("error checking usage limits existence: %w", formatAPIError(err))
		}

		return nil
	}
}

// Overage limits are kept at zero because the accepted range depends on the
// organization's plan maximums and its current overage usage, neither of which
// is known ahead of the test run.
var testAccUsageLimitsConfigBasic = strings.TrimSpace(fmt.Sprintf(`
resource "cloudsmith_usage_limits" "test" {
    organization              = "%s"
    allow_open_source_overage = false
    bandwidth_overage_limit   = 0
    storage_overage_limit     = 0
}
`, os.Getenv("CLOUDSMITH_NAMESPACE")))

var testAccUsageLimitsConfigOpenSourceOverage = strings.TrimSpace(fmt.Sprintf(`
resource "cloudsmith_usage_limits" "test" {
    organization              = "%s"
    allow_open_source_overage = true
    bandwidth_overage_limit   = 0
    storage_overage_limit     = 0
}
`, os.Getenv("CLOUDSMITH_NAMESPACE")))
