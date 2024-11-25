package cloudsmith

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccUserSelf_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccUserSelfConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudsmith_user_self.test", "email"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_user_self.test", "name"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_user_self.test", "slug"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_user_self.test", "slug_perm"),
				),
			},
		},
	})
}

const testAccUserSelfConfig = `
data "cloudsmith_user_self" "test" {}
`
