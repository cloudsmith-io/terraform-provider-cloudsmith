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

func TestOrgLicensePolicies_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testOrgLicensePoliciesCheckDestroy("cloudsmith_license_policies.test"),
		Steps: []resource.TestStep{
			{
				Config: testOrgLicensePoliciesBasic,
				Check: resource.ComposeTestCheckFunc(
					testOrgLicensePoliciesCheckExists("cloudsmith_license_policies.test"),
					// check properties have been set correctly
					resource.TestCheckResourceAttr("cloudsmith_license_policies.test", "description", "TF Test Policy Description"),
					resource.TestCheckResourceAttr("cloudsmith_license_policies.test", "name", "TF Test Policy"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policies.test", "on_violation_quarantine"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policies.test", "spdx_identifiers.#"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policies.test", "created_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policies.test", "updated_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_license_policies.test", "slug_perm"),
				),
			},
			{
				Config:      testOrgLicensePoliciesBasicInvalidSpdx,
				ExpectError: regexp.MustCompile("invalid spdx_identifiers:"),
			},
			{
				Config: testOrgLicensePoliciesBasicUpdate,
				Check: resource.ComposeTestCheckFunc(
					testOrgLicensePoliciesCheckExists("cloudsmith_license_policies.test"),
					// check values have been updated
					resource.TestCheckResourceAttr("cloudsmith_license_policies.test", "description", "TF Test Policy Description Updated"),
					resource.TestCheckResourceAttr("cloudsmith_license_policies.test", "name", "TF Test Policy Updated"),
					resource.TestCheckResourceAttr("cloudsmith_license_policies.test", "on_violation_quarantine", "true"),
				),
			},
			{
				ResourceName: "cloudsmith_license_policies.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources["cloudsmith_license_policies.test"]
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
func testOrgLicensePoliciesCheckDestroy(resourceName string) resource.TestCheckFunc {
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
func testOrgLicensePoliciesCheckExists(resourceName string) resource.TestCheckFunc {
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

var testOrgLicensePoliciesBasic = fmt.Sprintf(`
resource "cloudsmith_license_policies" "test" {
	name             = "TF Test Policy"
	description      = "TF Test Policy Description"
	spdx_identifiers = ["Apache-1.0"]
	organization     = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testOrgLicensePoliciesBasicInvalidSpdx = fmt.Sprintf(`
resource "cloudsmith_license_policies" "test" {
	name             = "TF Test Policy"
	description      = "TF Test Policy Description"
	spdx_identifiers = ["Not a spdx"]
	organization     = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testOrgLicensePoliciesBasicUpdate = fmt.Sprintf(`
resource "cloudsmith_license_policies" "test" {
	name                    = "TF Test Policy Updated"
	description             = "TF Test Policy Description Updated"
	spdx_identifiers        = ["Apache-2.0"]
	on_violation_quarantine = true
	organization            = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
