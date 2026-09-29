//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccRepositoryPrivileges_basic spins up a repository with default options,
// creates a service account and a couple of teams, assigning and modifying
// their permissions before tearing down and verifying deletion.
func TestAccRepositoryPrivileges_basic(t *testing.T) {
	callerKind, callerSlug := testAccRepositoryPrivilegesCaller(t)
	t.Parallel()

	repositoryName := testAccUniqueRepositoryName("terraform-acc-test-privs")
	callerBlock := testAccRepositoryPrivilegesCallerBlock(callerKind, callerSlug)
	checkCaller := resource.TestCheckTypeSetElemNestedAttrs("cloudsmith_repository_privileges.test", callerKind+".*", map[string]string{
		"slug":      callerSlug,
		"privilege": "Admin",
	})

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryPrivilegesConfigBasic(repositoryName, callerBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryPrivilegeForResource("cloudsmith_repository_privileges.test", "service", "cloudsmith_service.test", "Read"),
					checkCaller,
				),
			},
			{
				Config: testAccRepositoryPrivilegesConfigBasicUpdatePrivilege(repositoryName, callerBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryPrivilegeForResource("cloudsmith_repository_privileges.test", "service", "cloudsmith_service.test", "Write"),
					checkCaller,
				),
			},
			{
				Config: testAccRepositoryPrivilegesConfigBasicAddTeam(repositoryName, callerBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryPrivilegeForResource("cloudsmith_repository_privileges.test", "service", "cloudsmith_service.test", "Write"),
					testAccRepositoryPrivilegeForResource("cloudsmith_repository_privileges.test", "team", "cloudsmith_team.test_1", "Write"),
					checkCaller,
				),
			},
			{
				Config: testAccRepositoryPrivilegesConfigBasicAddAnotherTeam(repositoryName, callerBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryPrivilegeForResource("cloudsmith_repository_privileges.test", "service", "cloudsmith_service.test", "Write"),
					testAccRepositoryPrivilegeForResource("cloudsmith_repository_privileges.test", "team", "cloudsmith_team.test_2", "Write"),
					testAccRepositoryPrivilegeForResource("cloudsmith_repository_privileges.test", "team", "cloudsmith_team.test_1", "Read"),
					checkCaller,
				),
			},
			{
				ResourceName: "cloudsmith_repository_privileges.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources["cloudsmith_repository_privileges.test"]
					return fmt.Sprintf(
						"%s.%s",
						resourceState.Primary.Attributes["organization"],
						resourceState.Primary.Attributes["repository"],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRepositoryPrivilegesConfigBasic(repositoryName, callerBlock string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}

resource "cloudsmith_service" "test" {
	name         = "TF Test Service Privs"
	organization = cloudsmith_repository.test.namespace
	role         = "Member"
}

resource "cloudsmith_repository_privileges" "test" {
    organization = cloudsmith_repository.test.namespace
    repository   = cloudsmith_repository.test.slug

	service {
		privilege = "Read"
		slug      = cloudsmith_service.test.slug
	}

	# Include the authenticated account explicitly to satisfy lockout safeguard.
%s
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"), callerBlock)
}

func testAccRepositoryPrivilegesConfigBasicUpdatePrivilege(repositoryName, callerBlock string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}

resource "cloudsmith_service" "test" {
	name         = "TF Test Service Privs"
	organization = cloudsmith_repository.test.namespace
	role         = "Member"
}

resource "cloudsmith_repository_privileges" "test" {
    organization = cloudsmith_repository.test.namespace
    repository   = cloudsmith_repository.test.slug

	service {
		privilege = "Write"
		slug      = cloudsmith_service.test.slug
	}

	# Include the authenticated account explicitly to satisfy lockout safeguard.
%s
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"), callerBlock)
}

func testAccRepositoryPrivilegesConfigBasicAddTeam(repositoryName, callerBlock string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}

resource "cloudsmith_service" "test" {
	name         = "TF Test Service Privs"
	organization = cloudsmith_repository.test.namespace
	role         = "Member"
}

resource "cloudsmith_team" "test_1" {
	name         = "TF Test Team Privs 1"
	organization = cloudsmith_repository.test.namespace
}

resource "cloudsmith_repository_privileges" "test" {
    organization = cloudsmith_repository.test.namespace
    repository   = cloudsmith_repository.test.slug

	service {
		privilege = "Write"
		slug      = cloudsmith_service.test.slug
	}

	team {
		privilege = "Write"
		slug      = cloudsmith_team.test_1.slug
	}

%s
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"), callerBlock)
}

func testAccRepositoryPrivilegesConfigBasicAddAnotherTeam(repositoryName, callerBlock string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}

resource "cloudsmith_service" "test" {
	name         = "TF Test Service Privs"
	organization = cloudsmith_repository.test.namespace
	role         = "Member"
}

resource "cloudsmith_team" "test_1" {
	name         = "TF Test Team Privs 1"
	organization = cloudsmith_repository.test.namespace
}

resource "cloudsmith_team" "test_2" {
	name         = "TF Test Team Privs 2"
	organization = cloudsmith_repository.test.namespace
}

resource "cloudsmith_repository_privileges" "test" {
    organization = cloudsmith_repository.test.namespace
    repository   = cloudsmith_repository.test.slug

	service {
		privilege = "Write"
		slug      = cloudsmith_service.test.slug
	}

	team {
		privilege = "Write"
		slug      = cloudsmith_team.test_2.slug
	}

	team {
		privilege = "Read"
		slug      = cloudsmith_team.test_1.slug
	}

%s
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"), callerBlock)
}
