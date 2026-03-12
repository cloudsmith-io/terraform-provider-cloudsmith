package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// create basic manage team test function

func TestAccManageTeam_basic(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTeamCheckDestroy("cloudsmith_team.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccManageTeamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("cloudsmith_team.test"),
					resource.TestCheckResourceAttr("cloudsmith_manage_team.test", "team_name", "tf-test-manage-team-members"),
					resource.TestCheckResourceAttr("cloudsmith_manage_team.test", "members.0.role", "Member"),
					resource.TestCheckResourceAttr("cloudsmith_manage_team.test", "members.0.user", "bblizniak"),
				),
				// This is required as when creating a team, the creator gets automatically added which causes a 422 error
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

var testAccManageTeamConfigBasic = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	organization = "%s"
	name = "tf-test-manage-team-members"
}

resource "cloudsmith_manage_team" "test" {
	depends_on = [cloudsmith_team.test]
	organization = cloudsmith_team.test.organization
	team_name = cloudsmith_team.test.name
	members {
		role = "Member"
		user = "bblizniak"
	}
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
