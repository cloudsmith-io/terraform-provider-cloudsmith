// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:testpackage

package organization_member_details_test

import (
	"fmt"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/acctest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccOrganizationMemberDetails_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { acctest.TestAccPreCheck(t) },
		Providers: acctest.TestAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckOrganizationMemberDetailsConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_org_member_details.test", "email", "bblizniak@cloudsmith.io"),
					resource.TestCheckResourceAttr("data.cloudsmith_org_member_details.test", "has_two_factor", "true"),
					resource.TestCheckResourceAttr("data.cloudsmith_org_member_details.test", "is_active", "true"),
					resource.TestCheckResourceAttr("data.cloudsmith_org_member_details.test", "role", "Owner"),
				),
			},
		},
	})
}

func testAccCheckOrganizationMemberDetailsConfig() string {
	return fmt.Sprintf(`

data "cloudsmith_org_member_details" "test" {
	organization = "%s"
	member = "bblizniak"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
}
