//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// create basic oidc test function

func TestAccOidc_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOidcCheckDestroy("cloudsmith_oidc.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccOidcConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					testAccOidcCheckExists("cloudsmith_oidc.test"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "namespace", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "claims.key", "value"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "name", "test-oidc-terraform-provider"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "provider_url", "https://test.com"),
					resource.TestMatchResourceAttr("cloudsmith_oidc.test", "service_accounts.0", regexp.MustCompile("^test-oidc-service-account.*$")),
				),
			},
			{
				Config: testAccOidcConfigBasicUpdateName,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					testAccOidcCheckExists("cloudsmith_oidc.test"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "namespace", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "claims.key", "value2"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "enabled", "false"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "name", "test-oidc-terraform-provider-updated"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "provider_url", "https://test.com"),
					resource.TestMatchResourceAttr("cloudsmith_oidc.test", "service_accounts.0", regexp.MustCompile("^test-oidc-service-account.*$")),
				),
			},
			{
				Config: testAccOidcConfigBasicUpdateProps,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					testAccOidcCheckExists("cloudsmith_oidc.test"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "namespace", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "claims.key", "value"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "claims.key2", "value2"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "name", "test-oidc-terraform-provider-updated"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test", "provider_url", "https://test-updated-url.com"),
					resource.TestMatchResourceAttr("cloudsmith_oidc.test", "service_accounts.0", regexp.MustCompile("^test-oidc-service-account.*$")),
				),
			},
			{
				Config:      testAccOidcConfigInvalidProviderURL,
				ExpectError: regexp.MustCompile(`expected "provider_url" to have a host, got invalid-url`),
			},
			{
				Config:      testAccOidcConfigInvalidServiceAccount,
				ExpectError: regexp.MustCompile(`422 Unprocessable Entity  \(Invalid input.\)`),
			},
		},
	})
}

// Test dynamic OIDC provider scenarios: create with dynamic mappings, update mappings, then switch to static.
func TestAccOidc_dynamic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOidcCheckDestroy("cloudsmith_oidc.test_dynamic"),
		Steps: []resource.TestStep{
			{
				Config: testAccOidcConfigDynamicCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test_dyn_a"),
					testAccServiceCheckExists("cloudsmith_service.test_dyn_b"),
					testAccOidcCheckExists("cloudsmith_oidc.test_dynamic"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "namespace", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "mapping_claim", "sub"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "dynamic_mappings.#", "1"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "dynamic_mappings.0.claim_value", "value1"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "dynamic_mappings.0.service_account", "test-oidc-service-account-a"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "service_accounts.#", "0"),
				),
			},
			{
				Config: testAccOidcConfigDynamicUpdateMappings,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test_dyn_a"),
					testAccServiceCheckExists("cloudsmith_service.test_dyn_b"),
					testAccOidcCheckExists("cloudsmith_oidc.test_dynamic"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "mapping_claim", "sub"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "dynamic_mappings.#", "2"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "service_accounts.#", "0"),
				),
			},
			{
				// Switch from dynamic to static provider
				Config: testAccOidcConfigDynamicSwitchToStatic,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test_dyn_a"),
					testAccServiceCheckExists("cloudsmith_service.test_dyn_b"),
					testAccOidcCheckExists("cloudsmith_oidc.test_dynamic"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "mapping_claim", ""),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "dynamic_mappings.#", "0"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "service_accounts.#", "1"),
					resource.TestCheckResourceAttr("cloudsmith_oidc.test_dynamic", "service_accounts.0", "test-oidc-service-account-a"),
				),
			},
		},
	})
}

//nolint:goerr113

func testAccOidcCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.OrgsApi.OrgsOpenidConnectRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.OrgsApi.OrgsOpenidConnectReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify oidc deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify oidc deletion: still exists: %s/%s", os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		}
		defer resp.Body.Close()

		return nil
	}
}

// create a check exists function for oidc

func testAccOidcCheckExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("oidc not found")
		}

		return nil
	}
}

var testAccOidcConfigBasic = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
    organization = "%s"
    name = "test-oidc-service-account"
}

resource "cloudsmith_oidc" "test" {
      depends_on = [cloudsmith_service.test]
      namespace = "%s"
      claims = {
        "key" = "value"
      }
      enabled = true
      name = "test-oidc-terraform-provider"
      provider_url = "https://test.com"
      service_accounts = [cloudsmith_service.test.slug]
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccOidcConfigBasicUpdateName = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
    organization = "%s"
    name = "test-oidc-service-account"
}

resource "cloudsmith_oidc" "test" {
      depends_on = [cloudsmith_service.test]
      namespace = "%s"
      claims = {
        "key" = "value2"
      }
      enabled = false
      name = "test-oidc-terraform-provider-updated"
      provider_url = "https://test.com"
      service_accounts = [cloudsmith_service.test.slug]
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccOidcConfigBasicUpdateProps = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
    organization = "%s"
    name = "test-oidc-service-account"
}

resource "cloudsmith_oidc" "test" {
      depends_on = [cloudsmith_service.test]
      namespace = "%s"
      claims = {
        "key" = "value"
        "key2" = "value2"
      }
      enabled = true
      name = "test-oidc-terraform-provider-updated"
      provider_url = "https://test-updated-url.com"
      service_accounts = [cloudsmith_service.test.slug]
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

// test invalid oidc config, invalid URL and invalid Service Account

var testAccOidcConfigInvalidProviderURL = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
    organization = "%s"
    name = "test-oidc-service-account"
}

resource "cloudsmith_oidc" "test" {
      depends_on = [cloudsmith_service.test]
      namespace = "%s"
      claims = {
        "key" = "value"
      }
      enabled = true
      name = "test-oidc-terraform-provider-updated"
      provider_url = "invalid-url"
      service_accounts = [cloudsmith_service.test.slug]
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccOidcConfigInvalidServiceAccount = fmt.Sprintf(`

resource "cloudsmith_oidc" "test" {
      namespace = "%s"
      claims = {
        "key" = "value"
      }
      enabled = true
      name = "test-oidc-terraform-provider-updated"
      provider_url = "https://test-updated-url.com"
      service_accounts = ["invalid-service-account"]
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

// Dynamic provider configs
var testAccOidcConfigDynamicCreate = fmt.Sprintf(`
resource "cloudsmith_service" "test_dyn_a" {
	organization = "%s"
	name = "test-oidc-service-account-a"
}

resource "cloudsmith_service" "test_dyn_b" {
	organization = "%s"
	name = "test-oidc-service-account-b"
}

resource "cloudsmith_oidc" "test_dynamic" {
	depends_on = [cloudsmith_service.test_dyn_a, cloudsmith_service.test_dyn_b]
	namespace = "%s"
	claims = {
		"aud" = "example"
	}
	enabled = true
	name = "test-oidc-terraform-provider-dynamic"
	provider_url = "https://dynamic.example.com"
	mapping_claim = "sub"
	dynamic_mappings {
		claim_value = "value1"
		service_account = cloudsmith_service.test_dyn_a.slug
	}
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccOidcConfigDynamicUpdateMappings = fmt.Sprintf(`
resource "cloudsmith_service" "test_dyn_a" {
	organization = "%s"
	name = "test-oidc-service-account-a"
}

resource "cloudsmith_service" "test_dyn_b" {
	organization = "%s"
	name = "test-oidc-service-account-b"
}

resource "cloudsmith_oidc" "test_dynamic" {
	depends_on = [cloudsmith_service.test_dyn_a, cloudsmith_service.test_dyn_b]
	namespace = "%s"
	claims = {
		"aud" = "example"
	}
	enabled = true
	name = "test-oidc-terraform-provider-dynamic"
	provider_url = "https://dynamic.example.com"
	mapping_claim = "sub"
	dynamic_mappings {
		claim_value = "value1"
		service_account = cloudsmith_service.test_dyn_a.slug
	}
	dynamic_mappings {
		claim_value = "value2"
		service_account = cloudsmith_service.test_dyn_b.slug
	}
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccOidcConfigDynamicSwitchToStatic = fmt.Sprintf(`
resource "cloudsmith_service" "test_dyn_a" {
	organization = "%s"
	name = "test-oidc-service-account-a"
}

resource "cloudsmith_service" "test_dyn_b" {
	organization = "%s"
	name = "test-oidc-service-account-b"
}

resource "cloudsmith_oidc" "test_dynamic" {
	depends_on = [cloudsmith_service.test_dyn_a, cloudsmith_service.test_dyn_b]
	namespace = "%s"
	claims = {
		"aud" = "example"
	}
	enabled = true
	name = "test-oidc-terraform-provider-dynamic"
	provider_url = "https://dynamic.example.com"
	service_accounts = [cloudsmith_service.test_dyn_a.slug]
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))
