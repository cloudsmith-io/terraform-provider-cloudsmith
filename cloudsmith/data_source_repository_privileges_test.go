package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccDataSourceRepositoryPrivileges_basic tests the basic functionality of the data source.
func TestAccDataSourceRepositoryPrivileges_basic(t *testing.T) {
	callerKind, callerSlug := testAccRepositoryPrivilegesCaller(t)
	t.Parallel()

	repositoryName := testAccUniqueRepositoryName("terraform-acc-test-read-privs")
	callerBlock := testAccRepositoryPrivilegesCallerBlock(callerKind, callerSlug)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRepositoryPrivilegesConfigBasic(repositoryName, callerBlock),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryPrivilegeForResource("data.cloudsmith_repository_privileges.test_data", "service", "cloudsmith_service.test", "Read"),
					resource.TestCheckTypeSetElemNestedAttrs("data.cloudsmith_repository_privileges.test_data", callerKind+".*", map[string]string{
						"slug":      callerSlug,
						"privilege": "Admin",
					}),
				),
			},
		},
	})
}

func testAccDataSourceRepositoryPrivilegesConfigBasic(repositoryName, callerBlock string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}

resource "cloudsmith_service" "test" {
	name         = "TF Test Service Data Privs"
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

data "cloudsmith_repository_privileges" "test_data" {
	organization = cloudsmith_repository_privileges.test.organization
	repository   = cloudsmith_repository_privileges.test.repository
	depends_on = [cloudsmith_repository.test]
  }
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"), callerBlock)
}
