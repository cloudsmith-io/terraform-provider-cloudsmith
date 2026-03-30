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
					resource.TestCheckTypeSetElemNestedAttrs(
						"cloudsmith_manage_team.test",
						"members.*",
						map[string]string{
							"role": "Member",
							"user": "bblizniak",
						},
					),
				),
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
