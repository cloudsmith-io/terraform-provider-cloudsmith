// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package policy_test

import (
	"fmt"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/acctest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPolicyDataSource_basic(t *testing.T) {
	t.Parallel()

	name := acctest.TestAccUniquePolicyName("TF Acc DS Policy")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.TestAccPreCheck(t) },
		Providers:    acctest.TestAccProviders,
		CheckDestroy: acctest.TestAccPolicyCheckDestroy("cloudsmith_policy.test"),
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
`, acctest.TestAccNamespace(), name, acctest.TestAccNamespace())
}
