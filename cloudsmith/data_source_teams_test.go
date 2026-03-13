package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccDataSourceTeamList_basic ensures the team list data source returns at least one team.
func TestAccDataSourceTeamList_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceTeamListConfig(),
				Check: resource.ComposeTestCheckFunc(
					// Validate all exposed team fields for the created team
					resource.TestCheckResourceAttr("data.cloudsmith_team_list.test", "teams.0.name", "1-terraform-acc-test-team-list"),
					resource.TestCheckResourceAttr("data.cloudsmith_team_list.test", "teams.0.description", "Acceptance test team list"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_team_list.test", "teams.0.slug"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_team_list.test", "teams.0.slug_perm"),
					resource.TestCheckResourceAttr("data.cloudsmith_team_list.test", "teams.0.visibility", "Visible"),
				),
			},
		},
	})
}

func testAccDataSourceTeamListConfig() string {
	return fmt.Sprintf(`
	resource "cloudsmith_team" "example" {
		name         = "1-terraform-acc-test-team-list"
		organization = "%s"
		description  = "Acceptance test team list"
		visibility   = "Visible"
	}

	data "cloudsmith_team_list" "test" {
		organization = "%s"
		depends_on   = [cloudsmith_team.example]
	}
	`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))
}
