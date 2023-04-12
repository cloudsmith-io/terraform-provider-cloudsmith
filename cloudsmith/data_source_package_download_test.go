package cloudsmith

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPackageDownload_data(t *testing.T) {
	t.Parallel()

	namespace := ""
	repository := ""
	packageName := ""
	packageVersion := ""
	destinationPath := ""

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPackageDownloadData(namespace, repository, packageName, packageVersion, destinationPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "namespace", namespace),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "repository", repository),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "package_name", packageName),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "package_version", packageVersion),
					resource.TestCheckResourceAttrSet("data.cloudsmith_package_download.test", "cdn_url"),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "destination_path", destinationPath),
				),
			},
		},
	})
}

func testAccPackageDownloadData(namespace, repository, packageName, packageVersion, destinationPath string) string {
	return fmt.Sprintf(`
data "cloudsmith_package_download" "test" {
	namespace        = "%s"
	repository       = "%s"
	package_name     = "%s"
	package_version  = "%s"
	destination_path = "%s"
}
`, namespace, repository, packageName, packageVersion, destinationPath)
}
