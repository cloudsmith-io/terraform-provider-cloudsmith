package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccDataSourceRepositoryPrivileges_basic tests the basic functionality of the data source.
func TestAccDataSourceRepositoryPrivileges_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRepositoryPrivilegesConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudsmith_repository_privileges.test_data", "service.#"),
				),
			},
		},
	})
}

var testAccDataSourceRepositoryPrivilegesConfigBasic = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-read-privs"
	namespace = "%s"
}

resource "cloudsmith_service" "test" {
	name         = "TF Test Service Data Privs"
	organization = cloudsmith_repository.test.namespace
	role         = "Member"
}

data "cloudsmith_user_self" "current" {}

resource "cloudsmith_repository_privileges" "test" {
    organization = cloudsmith_repository.test.namespace
    repository   = cloudsmith_repository.test.slug

	service {
		privilege = "Read"
		slug      = cloudsmith_service.test.slug
	}

	# Include the authenticated account explicitly to satisfy lockout safeguard.
	user {
		privilege = "Admin"
		slug      = data.cloudsmith_user_self.current.slug
	}
}

data "cloudsmith_repository_privileges" "test_data" {
	organization = cloudsmith_repository_privileges.test.organization
	repository   = cloudsmith_repository_privileges.test.repository
	depends_on = [cloudsmith_repository.test]
  }
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
