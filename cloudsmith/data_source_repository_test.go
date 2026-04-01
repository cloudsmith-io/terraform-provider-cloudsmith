//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccRepository_data spins up a repository with all default options,
// verifies it exists, then reads the same repository using a data source and
// verifies that the expected fields are set with default values.
func TestAccRepository_data(t *testing.T) {
	t.Parallel()

	repositoryName := testAccUniqueRepositoryName("terraform-acc-test-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryData(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "name", repositoryName),
					resource.TestCheckResourceAttr("data.cloudsmith_repository.test", "name", repositoryName),
					// testing 5 representative fields, could be exhaustive here but feels like overkill for now
					resource.TestCheckResourceAttr("data.cloudsmith_repository.test", "contextual_auth_realm", "true"),
					resource.TestCheckResourceAttr("data.cloudsmith_repository.test", "docker_refresh_tokens_enabled", "false"),
					resource.TestCheckResourceAttr("data.cloudsmith_repository.test", "resync_own", "true"),
					resource.TestCheckResourceAttr("data.cloudsmith_repository.test", "resync_packages", "Admin"),
					resource.TestCheckResourceAttr("data.cloudsmith_repository.test", "use_vulnerability_scanning", "true"),
					resource.TestCheckResourceAttr("data.cloudsmith_repository.test", "broadcast_state", "Off"),
				),
			},
		},
	})
}

func testAccRepositoryData(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}

data "cloudsmith_repository" "test" {
	identifier = "%s"
	namespace  = cloudsmith_repository.test.namespace
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"), repositoryName)
}
