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
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository_privileges.test_data"),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceRepositoryPrivilegesConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudsmith_repository_privileges.test_data", "service.#"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_repository_privileges.test_data", "team.#"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_repository_privileges.test_data", "user.#"),
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

resource "cloudsmith_repository_privileges" "test" {
    organization = cloudsmith_repository.test.namespace
    repository   = cloudsmith_repository.test.slug

	service {
		privilege = "Read"
		slug      = cloudsmith_service.test.slug
	}
}

data "cloudsmith_repository_privileges" "test_data" {
  organization = cloudsmith_repository.test.namespace
  repository   = cloudsmith_repository.test.slug
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
