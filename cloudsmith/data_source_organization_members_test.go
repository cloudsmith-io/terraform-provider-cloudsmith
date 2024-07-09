//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Test member list function

func TestAccOrganizationMembersList_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckOrganizationMembersListConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_list_org_members.test", "is_active", "true"),
					resource.TestCheckResourceAttr("data.cloudsmith_list_org_members.test", "members.0.has_two_factor", "true"),
					resource.TestCheckResourceAttr("data.cloudsmith_list_org_members.test", "members.0.is_active", "true"),
					resource.TestCheckResourceAttr("data.cloudsmith_list_org_members.test", "members.0.user", "bblizniak"),
				),
			},
		},
	})
}

func testAccCheckOrganizationMembersListConfig() string {
	return fmt.Sprintf(`
data "cloudsmith_list_org_members" "test" {
    namespace = "%s"
    is_active = true
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
}
