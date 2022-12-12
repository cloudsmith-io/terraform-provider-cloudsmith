//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccRepository_basic spins up a repository with all default options,
// verifies it exists and checks the name is set correctly. Then it changes the
// name, and verifies it's been set correctly before tearing down the resource
// and verifying deletion.
//
// NOTE: It is not necessary to check properties that have been explicitly set
// as Terraform performs a drift/plan check after every step anyway. Only
// computed properties need explicitly checked.
func TestAccRepository_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test"),
					// check a sample of computed properties have been set correctly
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "contextual_auth_realm", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "copy_own", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "copy_packages", "Read"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "docker_refresh_tokens_enabled", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "is_private", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "is_public", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "replace_packages_by_default", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "use_vulnerability_scanning", "true"),
				),
			},
			{
				Config: testAccRepositoryConfigBasicUpdateName,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test"),
				),
			},
			{
				Config:      testAccRepositoryConfigBasicInvalidProp,
				ExpectError: regexp.MustCompile("expected copy_packages to be one of"),
			},
			{
				Config: testAccRepositoryConfigBasicUpdateProps,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test"),
				),
			},
		},
	})
}

//nolint:goerr113
func testAccRepositoryCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.ReposApi.ReposRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req)
		if err != nil {
			if is404(resp) {
				return nil
			}
			return err
		}
		defer resp.Body.Close()

		return fmt.Errorf("repository still exists")
	}
}

//nolint:goerr113
func testAccRepositoryCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.ReposApi.ReposRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	}
}

var testAccRepositoryConfigBasic = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test"
	namespace = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccRepositoryConfigBasicUpdateName = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-update"
	namespace = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccRepositoryConfigBasicInvalidProp = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-update"
	namespace = "%s"

	copy_packages = "Sudo"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccRepositoryConfigBasicUpdateProps = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-update"
	namespace = "%s"

	contextual_auth_realm         = false
	copy_packages                 = "Write"
	docker_refresh_tokens_enabled = true
	replace_packages_by_default   = true
	use_vulnerability_scanning    = false
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
