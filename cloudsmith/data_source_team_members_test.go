package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccDataSourceTeamMembers_basic validates that team members are listed with expected fields.
func TestAccDataSourceTeamMembers_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceTeamMembersConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_team_members.test", "team_name", "terraform-acc-test-team-members"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_team_members.test", "members.0.user"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_team_members.test", "members.0.role"),
				),
			},
		},
	})
}

func testAccDataSourceTeamMembersConfig() string {
	return fmt.Sprintf(`
resource "cloudsmith_team" "example" {
  name         = "terraform-acc-test-team-members"
  organization = "%s"
  description  = "Acceptance test team members"
  visibility   = "Visible"
}

data "cloudsmith_team_members" "test" {
  organization = "%s"
  team_name    = cloudsmith_team.example.name
  depends_on   = [cloudsmith_team.example]
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))
}
