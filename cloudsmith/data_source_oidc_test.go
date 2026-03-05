//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	testAccOidcDataServiceName       = testAccUniqueName("test-oidc-service-account-data")
	testAccOidcDataProviderName      = testAccUniqueName("test-oidc-terraform-provider")
	testAccOidcDataDynServiceAName   = testAccUniqueName("test-oidc-service-account-dyn-a")
	testAccOidcDataDynProviderName   = testAccUniqueName("test-oidc-terraform-provider-dynamic")
	testAccOidcDataDynMappingClaim   = "sub"
	testAccOidcDataDynMappingValue   = "value1"
	testAccOidcDataDynExpectedClaims = "example"
)

// TestAccOidc_data reads the configured OIDC provider using a data source and
// verifies that the expected fields are set with appropriate values.
func TestAccOidc_data(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOidcDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_oidc.test", "name", testAccOidcDataProviderName),
					resource.TestCheckResourceAttr("data.cloudsmith_oidc.test", "enabled", "true"),
					resource.TestCheckResourceAttr("data.cloudsmith_oidc.test", "provider_url", "https://test.com"),
					resource.TestCheckResourceAttr("data.cloudsmith_oidc.test", "claims.key", "value"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_oidc.test", "slug"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_oidc.test", "slug_perm"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_oidc.test", "service_accounts.#"),
				),
			},
		},
	})
}

// TestAccOidc_dataDynamic tests reading dynamic OIDC provider configuration
func TestAccOidc_dataDynamic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOidcDataSourceDynamicConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_oidc.test_dynamic", "name", testAccOidcDataDynProviderName),
					resource.TestCheckResourceAttr("data.cloudsmith_oidc.test_dynamic", "enabled", "true"),
					resource.TestCheckResourceAttr("data.cloudsmith_oidc.test_dynamic", "provider_url", "https://dynamic.example.com"),
					resource.TestCheckResourceAttr("data.cloudsmith_oidc.test_dynamic", "claims.aud", testAccOidcDataDynExpectedClaims),
					resource.TestCheckResourceAttr("data.cloudsmith_oidc.test_dynamic", "mapping_claim", testAccOidcDataDynMappingClaim),
					resource.TestCheckResourceAttrSet("data.cloudsmith_oidc.test_dynamic", "slug"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_oidc.test_dynamic", "slug_perm"),
					resource.TestCheckResourceAttr("data.cloudsmith_oidc.test_dynamic", "dynamic_mappings.#", "1"),
				),
			},
		},
	})
}

// Static OIDC configuration for data source testing
var testAccOidcDataSourceConfig = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
    organization = "%s"
    name = "%s"
}

resource "cloudsmith_oidc" "test" {
    depends_on = [cloudsmith_service.test]
    namespace = "%s"
    claims = {
        "key" = "value"
    }
    enabled = true
    name = "%s"
    provider_url = "https://test.com"
    service_accounts = [cloudsmith_service.test.slug]
}

data "cloudsmith_oidc" "test" {
    depends_on = [cloudsmith_oidc.test]
    namespace = "%s"
    slug_perm = cloudsmith_oidc.test.slug_perm
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), testAccOidcDataServiceName, os.Getenv("CLOUDSMITH_NAMESPACE"), testAccOidcDataProviderName, os.Getenv("CLOUDSMITH_NAMESPACE"))

// Dynamic OIDC configuration for data source testing
var testAccOidcDataSourceDynamicConfig = fmt.Sprintf(`
resource "cloudsmith_service" "test_dyn_a" {
	organization = "%s"
	name = "%s"
}

resource "cloudsmith_oidc" "test_dynamic" {
	depends_on = [cloudsmith_service.test_dyn_a]
	namespace = "%s"
	claims = {
		"aud" = "%s"
	}
	enabled = true
	name = "%s"
	provider_url = "https://dynamic.example.com"
	mapping_claim = "%s"
	dynamic_mappings {
		claim_value = "%s"
		service_account = cloudsmith_service.test_dyn_a.slug
	}
}

data "cloudsmith_oidc" "test_dynamic" {
	depends_on = [cloudsmith_oidc.test_dynamic]
	namespace = "%s"
	slug_perm = cloudsmith_oidc.test_dynamic.slug_perm
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), testAccOidcDataDynServiceAName, os.Getenv("CLOUDSMITH_NAMESPACE"), testAccOidcDataDynExpectedClaims, testAccOidcDataDynProviderName, testAccOidcDataDynMappingClaim, testAccOidcDataDynMappingValue, os.Getenv("CLOUDSMITH_NAMESPACE"))
