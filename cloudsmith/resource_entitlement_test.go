//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccEntitlement_basic spins up a repository with all default options,
// creates an entitlement with default options and verifies it exists and checks
// the name is set correctly. Then it changes the name and some of the limit
// variables, and verifies they've been set correctly before tearing down the
// resources and verifying deletion.
func TestAccEntitlement_basic(t *testing.T) {
	t.Parallel()

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
					resource.TestCheckResourceAttr("cloudsmith_entitlement.test", "limit_path_query", "/test-path"),
				),
			},
			{
				Config: testAccEntitlementConfigBasicUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccEntitlementCheckExists("cloudsmith_entitlement.test"),
					resource.TestCheckResourceAttr("cloudsmith_entitlement.test", "name", "Test Entitlement Update"),
					resource.TestCheckResourceAttr("cloudsmith_entitlement.test", "limit_num_downloads", "100"),
					resource.TestCheckResourceAttr("cloudsmith_entitlement.test", "limit_path_query", "/updated-path"),
				),
			},
			{
				ResourceName: "cloudsmith_entitlement.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources["cloudsmith_entitlement.test"]
					return fmt.Sprintf(
						"%s.%s.%s",
						resourceState.Primary.Attributes["namespace"],
						resourceState.Primary.Attributes["repository"],
						resourceState.Primary.ID,
					), nil
				},
				ImportStateVerify: true,
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

		req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, entitlement)
		_, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify entitlement deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify entitlement deletion: still exists: %s/%s/%s", namespace, repository, entitlement)
		}
		defer resp.Body.Close()

		rreq := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, repository)
		_, resp, err = pc.APIClient.ReposApi.ReposReadExecute(rreq)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify repository deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify repository deletion: still exists: %s/%s", namespace, repository)
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

		req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, entitlement)
		_, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req)
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
    name             = "Test Entitlement"
    limit_path_query = "/test-path"
    namespace        = "${cloudsmith_repository.test.namespace}"
    repository       = "${cloudsmith_repository.test.slug_perm}"
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
    limit_path_query    = "/updated-path"
    namespace           = "${cloudsmith_repository.test.namespace}"
    repository          = "${cloudsmith_repository.test.slug_perm}"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
