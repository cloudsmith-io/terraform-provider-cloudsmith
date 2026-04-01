//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccEntitlementTokenList_data spins up an entitlement token with all default options,
// verifies it exists, then reads the same entitlement token using a data source and
// verifies that the expected fields are set with default values.
func TestAccDataSourceEntitlementTokenList(t *testing.T) {
	repositoryName := testAccUniqueRepositoryName("terraform-acc-test-ent-list")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceEntitlementTokenListConfig(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudsmith_entitlement.test", "name"),
				),
			},
		},
	})
}

func testAccDataSourceEntitlementTokenListConfig(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}

resource "cloudsmith_entitlement" "test" {
    name       = "Test Entitlement"
    namespace  = "${cloudsmith_repository.test.namespace}"
    repository = "${cloudsmith_repository.test.slug_perm}"
}
data "cloudsmith_entitlement_list" "test" {
    query      = ["name:Test Entitlement"]
    repository = "${cloudsmith_repository.test.slug_perm}"
    namespace  = cloudsmith_repository.test.namespace
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}
