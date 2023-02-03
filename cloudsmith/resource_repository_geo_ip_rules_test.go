//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const ResourceName string = "cloudsmith_repository_geo_ip_rules.test"
const InitialCidrAllow string = "255.255.255.255/32"
const UpdatedCidrAllow string = "1.1.1.1/32"
const InitialCidrDeny string = "10.0.0.0/24"
const UpdatedCidrDeny string = "6cc2:ab98:2143:7e6e:8827:e81a:1527:9645/128"
const InitialCountryCodeAllow string = "BV"
const UpdatedCountryCodeAllow string = "FO"
const InitialCountryCodeDeny string = "CX"
const UpdatedCountryCodeDeny string = "CK"
const configTemplateWithoutRules string = `
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-repository-geo-ip-rules"
	namespace = "%s"
}

resource "cloudsmith_repository_geo_ip_rules" "test" {
    namespace          = "${resource.cloudsmith_repository.test.namespace}"
    repository         = "${resource.cloudsmith_repository.test.slug_perm}"
}
`
const configTemplateWithRules string = `
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-repository-geo-ip-rules"
	namespace = "%s"
}

resource "cloudsmith_repository_geo_ip_rules" "test" {
    namespace          = "${resource.cloudsmith_repository.test.namespace}"
    repository         = "${resource.cloudsmith_repository.test.slug_perm}"
    cidr_allow         = ["%s"]
    cidr_deny          = ["%s"]
    country_code_allow = ["%s"]
    country_code_deny  = ["%s"]
}
`

var namespace = os.Getenv("CLOUDSMITH_NAMESPACE")
var testAccRepositoryGeoIpRulesConfigCreate = fmt.Sprintf(configTemplateWithRules, namespace, InitialCidrAllow, InitialCidrDeny, InitialCountryCodeAllow, InitialCountryCodeDeny)
var testAccRepositoryGeoIpRulesConfigUpdate = fmt.Sprintf(configTemplateWithRules, namespace, UpdatedCidrAllow, UpdatedCidrDeny, UpdatedCountryCodeAllow, UpdatedCountryCodeDeny)
var testAccRepositoryGeoIpRulesConfigDefault = fmt.Sprintf(configTemplateWithoutRules, namespace)

// TestAccRepositoryGeoIpRules_basic spins up a repository with all default options,
// creates a set of geo/ip rules for the repository and verifies they exist. Then it
// changes the geo/ip rules and verifies they've been set correctly before tearing down the
// resources and verifying deletion.
func TestAccRepositoryGeoIpRules_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryGeoIpRulesCheckDestroy(ResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryGeoIpRulesConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryGeoIpRulesCheckExists(ResourceName, InitialCidrAllow, InitialCidrDeny, InitialCountryCodeAllow, InitialCountryCodeDeny),
				),
			},
			{
				Config: testAccRepositoryGeoIpRulesConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryGeoIpRulesCheckExists(ResourceName, UpdatedCidrAllow, UpdatedCidrDeny, UpdatedCountryCodeAllow, UpdatedCountryCodeDeny),
				),
			},
			{
				Config: testAccRepositoryGeoIpRulesConfigDefault,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryGeoIpRulesCheckExists(ResourceName, "", "", "", ""),
				),
			},
		},
	})
}

//nolint:goerr113
func testAccRepositoryGeoIpRulesCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		repository := resourceState.Primary.Attributes["repository"]

		req := pc.APIClient.ReposApi.ReposGeoipRead(pc.Auth, namespace, repository)
		_, resp, err := pc.APIClient.ReposApi.ReposGeoipReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify geo/ip rules deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify geo/ip rules deletion: still exists: %s/%s", namespace, repository)
		}
		defer resp.Body.Close()

		rreq := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, repository)
		_, resp, err = pc.APIClient.ReposApi.ReposReadExecute(rreq)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify repository deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify repository deletion: still exists: %s/%s", namespace, repository)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:goerr113
func testAccRepositoryGeoIpRulesCheckExists(resourceName string, expectedCidrAllow string, expectedCidrDeny string, expectedCountryCodeAllow string, expectedCountryCodeDeny string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		repository := resourceState.Primary.Attributes["repository"]

		req := pc.APIClient.ReposApi.ReposGeoipRead(pc.Auth, namespace, repository)
		geoIpRules, resp, err := pc.APIClient.ReposApi.ReposGeoipReadExecute(req)
		if err != nil {
			return fmt.Errorf("unable to verify Geo/IP rules existence: %w", err)
		}
		defer resp.Body.Close()

		cidr := geoIpRules.GetCidr()

		if len(expectedCidrAllow) > 0 && !contains(cidr.GetAllow(), expectedCidrAllow) {
			return fmt.Errorf("expected cidr_allow not found in rules")
		} else if len(expectedCidrAllow) == 0 && len(cidr.GetAllow()) > 0 {
			return fmt.Errorf("expected cidr_allow rules to be empty")
		}

		if len(expectedCidrDeny) > 0 && !contains(cidr.GetDeny(), expectedCidrDeny) {
			return fmt.Errorf("expected cidr_deny not found in rules")
		} else if len(expectedCidrDeny) == 0 && len(cidr.GetDeny()) > 0 {
			return fmt.Errorf("expected cidr_deny rules to be empty")
		}

		countryCode := geoIpRules.GetCountryCode()

		if len(expectedCountryCodeAllow) > 0 && !contains(countryCode.GetAllow(), expectedCountryCodeAllow) {
			return fmt.Errorf("expected country_code_allow not found in rules")
		} else if len(expectedCountryCodeAllow) == 0 && len(countryCode.GetAllow()) > 0 {
			return fmt.Errorf("expected country_code_allow rules to be empty")
		}

		if len(expectedCountryCodeDeny) > 0 && !contains(countryCode.GetDeny(), expectedCountryCodeDeny) {
			return fmt.Errorf("expected country_code_deny not found in rules")
		} else if len(expectedCountryCodeDeny) == 0 && len(countryCode.GetDeny()) > 0 {
			return fmt.Errorf("expected country_code_deny rules to be empty")
		}

		return nil
	}
}
