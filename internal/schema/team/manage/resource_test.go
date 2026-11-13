// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package manage_test

import (
	"fmt"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/acctest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// create basic manage team test function

func TestAccManageTeam_basic(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.TestAccPreCheck(t) },
		Providers:    acctest.TestAccProviders,
		CheckDestroy: acctest.TestAccTeamCheckDestroy("cloudsmith_team.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccManageTeamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					acctest.TestAccTeamCheckExists("cloudsmith_team.test"),
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
