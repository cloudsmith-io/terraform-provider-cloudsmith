//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccRepositoryConnectedList_basic creates a source repository, a target
// repository, and a connection between them, then asserts the
// cloudsmith_repository_connected_list data source returns the connection.
func TestAccRepositoryConnectedList_basic(t *testing.T) {
	t.Parallel()

	sourceName := testAccUniqueRepositoryName("tf-acc-cnx-list-src")
	targetName := testAccUniqueRepositoryName("tf-acc-cnx-list-tgt")

	const dataSourceName = "data.cloudsmith_repository_connected_list.test"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryConnectedListConfig(sourceName, targetName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "connected_repositories.#", "1"),
					resource.TestCheckResourceAttrPair(
						dataSourceName, "connected_repositories.0.target_repository",
						"cloudsmith_repository.target", "slug",
					),
					resource.TestCheckResourceAttr(dataSourceName, "connected_repositories.0.is_active", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "connected_repositories.0.priority", "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "connected_repositories.0.slug_perm"),
					resource.TestCheckResourceAttrSet(dataSourceName, "connected_repositories.0.created_at"),
				),
			},
		},
	})
}

func testAccRepositoryConnectedListConfig(sourceName, targetName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "source" {
    name      = "%s"
    namespace = "%s"
}

resource "cloudsmith_repository" "target" {
    name      = "%s"
    namespace = "%s"
}

resource "cloudsmith_repository_connected" "test" {
    namespace         = resource.cloudsmith_repository.source.namespace
    repository        = resource.cloudsmith_repository.source.slug_perm
    target_repository = resource.cloudsmith_repository.target.slug
    is_active         = true
    priority          = 1
}

data "cloudsmith_repository_connected_list" "test" {
    namespace  = resource.cloudsmith_repository.source.namespace
    repository = resource.cloudsmith_repository.source.slug_perm

    depends_on = [cloudsmith_repository_connected.test]
}
`, sourceName, testAccNamespace(), targetName, testAccNamespace())
}
