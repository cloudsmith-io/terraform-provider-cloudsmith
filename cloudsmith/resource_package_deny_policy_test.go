//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// create a baisc package deny policy function

func TestAccPackageDenyPolicy_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPackageDenyPolicyCheckDestroy("cloudsmith_package_deny_policy.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccPackageDenyPolicyConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccPackageDenyPolicyCheckExists("cloudsmith_package_deny_policy.test"),
					resource.TestCheckResourceAttr("cloudsmith_package_deny_policy.test", "namespace", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_package_deny_policy.test", "enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_package_deny_policy.test", "name", "test-package-deny-policy-terraform-provider"),
					resource.TestCheckResourceAttr("cloudsmith_package_deny_policy.test", "package_query", "name:example_new"),
				),
			},
		},
	})
}

// create a basic package deny policy config

var testAccPackageDenyPolicyConfigBasic = fmt.Sprintf(`
resource "cloudsmith_package_deny_policy" "test" {
  namespace = "%s"
  enabled = true
  name = "test-package-deny-policy-terraform-provider"
  package_query = "name:example_new"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

// create a package deny policy check destroy function

func testAccPackageDenyPolicyCheckDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*providerConfig).APIClient
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "cloudsmith_package_deny_policy" {
				continue
			}

			_, _, err := client.OrgsApi.OrgsDenyPolicyRead(testAccProvider.Meta().(*providerConfig).Auth, rs.Primary.Attributes["namespace"], rs.Primary.ID).Execute()
			if err == nil {
				return fmt.Errorf("Package deny policy still exists")
			}
		}
		return nil
	}
}

// create a package deny policy check exists function

func testAccPackageDenyPolicyCheckExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*providerConfig).APIClient
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "cloudsmith_package_deny_policy" {
				continue
			}

			_, _, err := client.OrgsApi.OrgsDenyPolicyRead(testAccProvider.Meta().(*providerConfig).Auth, rs.Primary.Attributes["namespace"], rs.Primary.ID).Execute()
			if err != nil {
				return fmt.Errorf("Package deny policy does not exist")
			}
		}
		return nil
	}
}
