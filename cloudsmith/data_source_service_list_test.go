package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccDataSourceServiceList_basic ensures the service list data source returns at least one service.
func TestAccDataSourceServiceList_basic(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceServiceListConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudsmith_service_list.test", "services.0.slug"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_service_list.test", "services.0.created_at"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_service_list.test", "services.0.role"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_service_list.test", "services.0.key"),
				),
			},
		},
	})
}

func testAccDataSourceServiceListConfig() string {
	return fmt.Sprintf(`
resource "cloudsmith_service" "example" {
  name         = "terraform-acc-test-service-list"
  organization = "%s"
  role         = "Member"
}

data "cloudsmith_service_list" "test" {
  organization = "%s"
  query        = "name:terraform-acc-test-service-list"
  sort         = "-created_at"
  depends_on   = [cloudsmith_service.example]
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))
}
