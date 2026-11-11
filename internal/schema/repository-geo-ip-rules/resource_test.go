// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:testpackage
package repository_geo_ip_rules_test

import (
	"fmt"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/acctest"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/util"
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

// TestAccRepositoryGeoIpRules_basic spins up a repository with all default options,
// creates a set of geo/ip rules for the repository and verifies they exist. Then it
// changes the geo/ip rules and verifies they've been set correctly before tearing down the
// resources and verifying deletion.
func TestAccRepositoryGeoIpRules_basic(t *testing.T) {
	t.Parallel()

	repositoryName := acctest.TestAccUniqueRepositoryName("terraform-acc-test-repository-geo-ip-rules")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.TestAccPreCheck(t) },
		Providers:    acctest.TestAccProviders,
		CheckDestroy: testAccRepositoryGeoIpRulesCheckDestroy(ResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryGeoIpRulesConfigCreate(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryGeoIpRulesCheckExists(ResourceName, InitialCidrAllow, InitialCidrDeny, InitialCountryCodeAllow, InitialCountryCodeDeny),
				),
			},
			{
				Config: testAccRepositoryGeoIpRulesConfigUpdate(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryGeoIpRulesCheckExists(ResourceName, UpdatedCidrAllow, UpdatedCidrDeny, UpdatedCountryCodeAllow, UpdatedCountryCodeDeny),
				),
			},
			{
				ResourceName: ResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[ResourceName]
					return fmt.Sprintf(
						"%s.%s",
						resourceState.Primary.Attributes["namespace"],
						resourceState.Primary.Attributes["repository"],
					), nil
				},
				ImportStateVerify: true,
			},
			{
				Config: testAccRepositoryGeoIpRulesConfigDefault(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryGeoIpRulesCheckExists(ResourceName, "", "", "", ""),
				),
			},
		},
	})
}

func testAccRepositoryGeoIpRulesConfigDefault(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}

resource "cloudsmith_repository_geo_ip_rules" "test" {
    namespace          = "${resource.cloudsmith_repository.test.namespace}"
    repository         = "${resource.cloudsmith_repository.test.slug_perm}"
}
`, repositoryName, acctest.TestAccNamespace())
}

func testAccRepositoryGeoIpRulesConfigCreate(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
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
`, repositoryName, acctest.TestAccNamespace(), InitialCidrAllow, InitialCidrDeny, InitialCountryCodeAllow, InitialCountryCodeDeny)
}

func testAccRepositoryGeoIpRulesConfigUpdate(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
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
`, repositoryName, acctest.TestAccNamespace(), UpdatedCidrAllow, UpdatedCidrDeny, UpdatedCountryCodeAllow, UpdatedCountryCodeDeny)
}

//nolint:err113
func testAccRepositoryGeoIpRulesCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := acctest.TestAccProvider.Meta().(*util.ProviderConfig)

		repository := resourceState.Primary.Attributes["repository"]

		req := pc.APIClient.ReposApi.ReposGeoipRead(pc.Auth, acctest.TestAccNamespace(), repository)
		_, resp, err := pc.APIClient.ReposApi.ReposGeoipReadExecute(req)
		if err != nil && !util.Is404(resp) {
			return fmt.Errorf("unable to verify geo/ip rules deletion: %w", err)
		} else if util.Is200(resp) {
			return fmt.Errorf("unable to verify geo/ip rules deletion: still exists: %s/%s", acctest.TestAccNamespace(), repository)
		}
		defer resp.Body.Close()

		rreq := pc.APIClient.ReposApi.ReposRead(pc.Auth, acctest.TestAccNamespace(), repository)
		_, resp, err = pc.APIClient.ReposApi.ReposReadExecute(rreq)
		if err != nil && !util.Is404(resp) {
			return fmt.Errorf("unable to verify repository deletion: %w", err)
		} else if util.Is200(resp) {
			return fmt.Errorf("unable to verify repository deletion: still exists: %s/%s", acctest.TestAccNamespace(), repository)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:err113
func testAccRepositoryGeoIpRulesCheckExists(resourceName string, expectedCidrAllow string, expectedCidrDeny string, expectedCountryCodeAllow string, expectedCountryCodeDeny string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := acctest.TestAccProvider.Meta().(*util.ProviderConfig)

		repository := resourceState.Primary.Attributes["repository"]

		req := pc.APIClient.ReposApi.ReposGeoipRead(pc.Auth, acctest.TestAccNamespace(), repository)
		geoIpRules, resp, err := pc.APIClient.ReposApi.ReposGeoipReadExecute(req)
		if err != nil {
			return fmt.Errorf("unable to verify Geo/IP rules existence: %w", err)
		}
		defer resp.Body.Close()

		cidr := geoIpRules.GetCidr()

		if len(expectedCidrAllow) > 0 && !util.Contains(cidr.GetAllow(), expectedCidrAllow) {
			return fmt.Errorf("expected cidr_allow not found in rules")
		} else if len(expectedCidrAllow) == 0 && len(cidr.GetAllow()) > 0 {
			return fmt.Errorf("expected cidr_allow rules to be empty")
		}

		if len(expectedCidrDeny) > 0 && !util.Contains(cidr.GetDeny(), expectedCidrDeny) {
			return fmt.Errorf("expected cidr_deny not found in rules")
		} else if len(expectedCidrDeny) == 0 && len(cidr.GetDeny()) > 0 {
			return fmt.Errorf("expected cidr_deny rules to be empty")
		}

		countryCode := geoIpRules.GetCountryCode()

		if len(expectedCountryCodeAllow) > 0 && !util.Contains(countryCode.GetAllow(), expectedCountryCodeAllow) {
			return fmt.Errorf("expected country_code_allow not found in rules")
		} else if len(expectedCountryCodeAllow) == 0 && len(countryCode.GetAllow()) > 0 {
			return fmt.Errorf("expected country_code_allow rules to be empty")
		}

		if len(expectedCountryCodeDeny) > 0 && !util.Contains(countryCode.GetDeny(), expectedCountryCodeDeny) {
			return fmt.Errorf("expected country_code_deny not found in rules")
		} else if len(expectedCountryCodeDeny) == 0 && len(countryCode.GetDeny()) > 0 {
			return fmt.Errorf("expected country_code_deny rules to be empty")
		}

		return nil
	}
}
