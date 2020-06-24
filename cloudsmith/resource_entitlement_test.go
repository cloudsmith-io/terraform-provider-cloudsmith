//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

// TestAccEntitlement_basic spins up a repository with all default options,
// creates an entitlement with default options and verifies it exists and checks
// the name is set correctly. Then it changes the name and some of the limit
// variables, and verifies they've been set correctly before tearing down the
// resources and verifying deletion.
func TestAccEntitlement_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccEntitlementCheckDestroy("cloudsmith_entitlement.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccEntitlementConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccEntitlementCheckExists("cloudsmith_entitlement.test"),
					resource.TestCheckResourceAttr("cloudsmith_entitlement.test", "name", "Test Entitlement"),
					resource.TestCheckResourceAttr("cloudsmith_entitlement.test", "limit_num_downloads", "0"),
				),
			},
			{
				Config: testAccEntitlementConfigBasicUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccEntitlementCheckExists("cloudsmith_entitlement.test"),
					resource.TestCheckResourceAttr("cloudsmith_entitlement.test", "name", "Test Entitlement Update"),
					resource.TestCheckResourceAttr("cloudsmith_entitlement.test", "limit_num_downloads", "100"),
				),
			},
		},
	})
}

//nolint:goerr113
func testAccEntitlementCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
		repository := resourceState.Primary.Attributes["repository"]
		entitlement := resourceState.Primary.ID

		_, resp, err := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, entitlement, nil)
		if err != nil {
			if err.Error() != errMessage404 {
				return fmt.Errorf("unable to verify entitlement deletion: %w", err)
			}
		}
		defer resp.Body.Close()

		_, resp, err = pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, repository)
		if err != nil {
			if err.Error() != errMessage404 {
				return fmt.Errorf("unable to verify repository deletion: %w", err)
			}
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:goerr113
func testAccEntitlementCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
		repository := resourceState.Primary.Attributes["repository"]
		entitlement := resourceState.Primary.ID

		_, resp, err := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, entitlement, nil)
		if err != nil {
			return fmt.Errorf("unable to verify entitlement existence: %w", err)
		}
		defer resp.Body.Close()

		return nil
	}
}

var testAccEntitlementConfigBasic = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-ent"
	namespace = "%s"
}

resource "cloudsmith_entitlement" "test" {
    name       = "Test Entitlement"
    namespace  = "${cloudsmith_repository.test.namespace}"
    repository = "${cloudsmith_repository.test.slug_perm}"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccEntitlementConfigBasicUpdate = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-ent"
	namespace = "%s"
}

resource "cloudsmith_entitlement" "test" {
	name                = "Test Entitlement Update"
    limit_num_downloads = 100
    namespace           = "${cloudsmith_repository.test.namespace}"
    repository          = "${cloudsmith_repository.test.slug_perm}"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
