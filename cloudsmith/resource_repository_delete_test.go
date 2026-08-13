//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccRepositoryDelete_afterPrivilegesRevoked reproduces the ordering from
// issue #214: a repository with default_privilege = "None" (so the acceptance
// test identity has no access to it except what's granted explicitly), a
// team, and a cloudsmith_repository_privileges resource granting the
// acceptance test identity (and the team) Admin on the repository.
//
// Terraform's dependency graph destroys cloudsmith_repository_privileges
// before cloudsmith_repository (since privileges depends on the repository),
// creating the lockout risk before the repository's own Delete is called. The
// privileges Delete must preserve the authenticated service account's Admin
// grant while removing the managed team grant, so the subsequent repository
// DELETE remains authorized and actually removes the repository.
func TestAccRepositoryDelete_afterPrivilegesRevoked(t *testing.T) {
	t.Parallel()

	repositoryName := testAccUniqueRepositoryName("terraform-acc-test-del")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryDeleteConfigPrivilegesRevoked(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "default_privilege", "None"),
				),
			},
		},
		// No further steps: resource.Test destroys everything created above at
		// the end of the test, which exercises the exact ordering (privileges
		// destroyed before the repository) that triggered the bug. If destroy
		// returns an error, the test fails here.
	})
}

func testAccRepositoryDeleteConfigPrivilegesRevoked(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name               = "%s"
	namespace          = "%s"
	default_privilege  = "None"
}

resource "cloudsmith_team" "test" {
	name         = "%s-team"
	organization = cloudsmith_repository.test.namespace
}

data "cloudsmith_user_self" "current" {}

resource "cloudsmith_repository_privileges" "test" {
	organization = cloudsmith_repository.test.namespace
	repository   = cloudsmith_repository.test.slug

	# Grants the acceptance-test identity (standing in for the "provisioning
	# SA" from the issue) Admin on the repository. Since default_privilege is
	# "None" above, this is the *only* source of access this identity has to
	# the repository, so destroying this resource before the repository
	# reproduces "SA lost visibility after privileges revoked" from #214.
	# The acceptance test credentials authenticate as a service account, so
	# the grant must go in the "service" block, not "user".
	service {
		privilege = "Admin"
		slug      = data.cloudsmith_user_self.current.slug
	}

	team {
		privilege = "Admin"
		slug      = cloudsmith_team.test.slug
	}
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"), repositoryName)
}
