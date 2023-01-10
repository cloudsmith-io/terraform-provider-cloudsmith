//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccOrganization_data reads the configured organization using a data source and
// verifies that the expected fields are set with appropriate values.
func TestAccOrganization_data(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationData,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_organization.test", "slug", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttrSet("data.cloudsmith_organization.test", "country"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_organization.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_organization.test", "location"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_organization.test", "name"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_organization.test", "slug_perm"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_organization.test", "tagline"),
				),
			},
		},
	})
}

var testAccOrganizationData = fmt.Sprintf(`
data "cloudsmith_organization" "test" {
	slug = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
