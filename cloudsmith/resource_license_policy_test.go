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

func TestAccOrgLicensePolicy_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testOrgLicensePolicyCheckDestroy("cloudsmith_license_policy.test"),
		Steps: []resource.TestStep{
			{
				Config: testOrgLicensePolicyBasic,
				Check: resource.ComposeTestCheckFunc(
					testOrgLicensePolicyCheckExists("cloudsmith_license_policy.test"),
					// check computed properties have been set correctly
					resource.TestCheckResourceAttrSet("cloudsmith_license_policy.test", "allow_unknown_licenses"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policy.test", "on_violation_quarantine"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policy.test", "created_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policy.test", "updated_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policy.test", "slug_perm"),
				),
			},
			{
				Config:      testOrgLicensePolicyBasicInvalidSpdx,
				ExpectError: regexp.MustCompile("invalid spdx_identifiers:"),
			},
			{
				Config: testOrgLicensePolicyBasicUpdate,
				Check: resource.ComposeTestCheckFunc(
					testOrgLicensePolicyCheckExists("cloudsmith_license_policy.test"),
					// check computed properties have been set correctly
					resource.TestCheckResourceAttrSet("cloudsmith_license_policy.test", "created_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policy.test", "updated_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policy.test", "slug_perm"),
				),
			},
			{
				ResourceName: "cloudsmith_license_policy.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources["cloudsmith_license_policy.test"]
					return fmt.Sprintf(
						"%s.%s",
						resourceState.Primary.Attributes["organization"],
						resourceState.Primary.Attributes["slug_perm"],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

//nolint:goerr113
func testOrgLicensePolicyCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.OrgsApi.OrgsLicensePolicyRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.OrgsApi.OrgsLicensePolicyReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify license policy deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify license policy deletion: still exists: %s/%s", os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:goerr113
func testOrgLicensePolicyCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.OrgsApi.OrgsLicensePolicyRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.OrgsApi.OrgsLicensePolicyReadExecute(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	}
}

var testOrgLicensePolicyBasic = fmt.Sprintf(`
resource "cloudsmith_license_policy" "test" {
	name             = "TF Test Policy"
	description      = "TF Test Policy Description"
	spdx_identifiers = ["Apache-1.0"]
	organization     = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testOrgLicensePolicyBasicInvalidSpdx = fmt.Sprintf(`
resource "cloudsmith_license_policy" "test" {
	name             = "TF Test Policy"
	description      = "TF Test Policy Description"
	spdx_identifiers = ["Not a spdx"]
	organization     = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testOrgLicensePolicyBasicUpdate = fmt.Sprintf(`
resource "cloudsmith_license_policy" "test" {
	name                    = "TF Test Policy Updated"
	description             = "TF Test Policy Description Updated"
	spdx_identifiers        = ["Apache-2.0"]
	on_violation_quarantine = true
	allow_unknown_licenses  = true
	organization            = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
