package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccDataSourceServiceDetails_basic validates retrieval of a single service's details.
func TestAccDataSourceServiceDetails_basic(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceServiceDetailsConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_service_details.test", "name", "terraform-acc-test-service-details"),
					resource.TestCheckResourceAttr("data.cloudsmith_service_details.test", "role", "Member"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_service_details.test", "slug"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_service_details.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_service_details.test", "created_by"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_service_details.test", "key"),
				),
			},
		},
	})
}

func testAccDataSourceServiceDetailsConfig() string {
	return fmt.Sprintf(`
resource "cloudsmith_service" "example" {
  name         = "terraform-acc-test-service-details"
  organization = "%s"
  role         = "Member"
}

data "cloudsmith_service_details" "test" {
  organization = "%s"
  service      = cloudsmith_service.example.slug
  depends_on   = [cloudsmith_service.example]
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))
}
