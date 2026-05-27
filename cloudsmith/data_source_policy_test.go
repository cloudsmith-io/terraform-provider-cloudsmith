package cloudsmith

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPolicyDataSource_basic(t *testing.T) {
	t.Parallel()

	name := testAccUniquePolicyName("TF Acc DS Policy")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPolicyCheckDestroy("cloudsmith_policy.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDataSourceConfigBasic(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.cloudsmith_policy.lookup", "name", "cloudsmith_policy.test", "name"),
					resource.TestCheckResourceAttrPair("data.cloudsmith_policy.lookup", "rego", "cloudsmith_policy.test", "rego"),
					resource.TestCheckResourceAttrPair("data.cloudsmith_policy.lookup", "policy_slug_perm", "cloudsmith_policy.test", "slug_perm"),
				),
			},
		},
	})
}

func testAccPolicyDataSourceConfigBasic(name string) string {
	return fmt.Sprintf(`
resource "cloudsmith_policy" "test" {
    workspace = "%s"
    name      = "%s"
    rego      = <<-EOT
        package cloudsmith.policy
        default allow := true
    EOT
}

data "cloudsmith_policy" "lookup" {
    workspace        = "%s"
    policy_slug_perm = cloudsmith_policy.test.slug_perm
}
`, testAccNamespace(), name, testAccNamespace())
}
