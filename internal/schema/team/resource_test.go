// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:testpackage
package team_test

import (
	"fmt"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/acctest"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccTeam_basic spins up a team with all default options,
// verifies it exists and checks the name is set correctly. Then it changes the
// name, and verifies it's been set correctly before tearing down the resource
// and verifying deletion.
//
// NOTE: It is not necessary to check properties that have been explicitly set
// as Terraform performs a drift/plan check after every step anyway. Only
// computed properties need explicitly checked.
func TestAccTeam_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.TestAccPreCheck(t) },
		Providers:    acctest.TestAccProviders,
		CheckDestroy: acctest.TestAccTeamCheckDestroy("cloudsmith_team.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					acctest.TestAccTeamCheckExists("cloudsmith_team.test"),
					// check a sample of computed properties have been set correctly
					resource.TestCheckResourceAttr("cloudsmith_team.test", "description", ""),
					resource.TestCheckResourceAttr("cloudsmith_team.test", "slug", "tf-test-team"),
					resource.TestCheckResourceAttrSet("cloudsmith_team.test", "slug_perm"),
					resource.TestCheckResourceAttr("cloudsmith_team.test", "visibility", "Visible"),
				),
			},
			{
				Config: testAccTeamConfigBasicUpdateName,
				Check: resource.ComposeTestCheckFunc(
					acctest.TestAccTeamCheckExists("cloudsmith_team.test"),
				),
			},
			{
				Config:      testAccTeamConfigBasicInvalidProp,
				ExpectError: regexp.MustCompile("expected visibility to be one of"),
			},
			{
				Config: testAccTeamConfigBasicUpdateProps,
				Check: resource.ComposeTestCheckFunc(
					acctest.TestAccTeamCheckExists("cloudsmith_team.test"),
				),
			},
			{
				ResourceName: "cloudsmith_team.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources["cloudsmith_team.test"]
					return fmt.Sprintf(
						"%s.%s",
						resourceState.Primary.Attributes["organization"],
						resourceState.Primary.Attributes["slug"],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

var testAccTeamConfigBasic = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	name         = "TF Test Team"
	organization = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccTeamConfigBasicUpdateName = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	name         = "TF Test Team Updated"
	organization = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccTeamConfigBasicInvalidProp = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	name         = "TF Test Team Updated"
	organization = "%s"

	visibility = "Nope"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccTeamConfigBasicUpdateProps = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	name         = "TF Test Team Updated"
	organization = "%s"

	description = "I am the team, coo coo ca choo"
	slug        = "my-very-imaginative-slug"
	visibility  = "Visible"

}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
