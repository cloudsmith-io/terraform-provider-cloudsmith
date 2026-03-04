//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePackageDenyPolicy_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPackageDenyPolicyCheckDestroy("cloudsmith_package_deny_policy.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePackageDenyPolicyConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccPackageDenyPolicyCheckExists("cloudsmith_package_deny_policy.test"),
					resource.TestCheckResourceAttrPair(
						"data.cloudsmith_package_deny_policy.test", "namespace",
						"cloudsmith_package_deny_policy.test", "namespace",
					),
					resource.TestCheckResourceAttrPair(
						"data.cloudsmith_package_deny_policy.test", "slug_perm",
						"cloudsmith_package_deny_policy.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.cloudsmith_package_deny_policy.test", "name",
						"cloudsmith_package_deny_policy.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.cloudsmith_package_deny_policy.test", "description",
						"cloudsmith_package_deny_policy.test", "description",
					),
					resource.TestCheckResourceAttrPair(
						"data.cloudsmith_package_deny_policy.test", "package_query",
						"cloudsmith_package_deny_policy.test", "package_query",
					),
					resource.TestCheckResourceAttrPair(
						"data.cloudsmith_package_deny_policy.test", "enabled",
						"cloudsmith_package_deny_policy.test", "enabled",
					),
				),
			},
		},
	})
}

var testAccDataSourcePackageDenyPolicyConfigBasic = fmt.Sprintf(`
resource "cloudsmith_package_deny_policy" "test" {
  namespace = "%s"
  enabled = true
  name = "test-package-deny-policy-ds-terraform-provider"
  description = "Data source acceptance test for package deny policy."
  package_query = "name:example_ds"
}

data "cloudsmith_package_deny_policy" "test" {
  namespace = cloudsmith_package_deny_policy.test.namespace
  slug_perm = cloudsmith_package_deny_policy.test.id
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
