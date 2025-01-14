//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccEntitlementControl_basic spins up a repository and uses its default entitlement token,
// creates an entitlement control with the token disabled, verifies it exists and checks
// the enabled state is set correctly. Then it changes the enabled state to true,
// and verifies it's been set correctly before tearing down the resources and
// verifying deletion.
func TestAccEntitlementControl_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccEntitlementControlCheckDestroy("cloudsmith_entitlement_control.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccEntitlementControlConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccEntitlementControlCheckExists("cloudsmith_entitlement_control.test"),
					resource.TestCheckResourceAttr("cloudsmith_entitlement_control.test", "namespace", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_entitlement_control.test", "enabled", "false"),
				),
			},
			{
				Config: testAccEntitlementControlConfigBasicUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccEntitlementControlCheckExists("cloudsmith_entitlement_control.test"),
					resource.TestCheckResourceAttr("cloudsmith_entitlement_control.test", "namespace", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_entitlement_control.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "cloudsmith_entitlement_control.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources["cloudsmith_entitlement_control.test"]
					return fmt.Sprintf(
						"%s.%s.%s",
						resourceState.Primary.Attributes["namespace"],
						resourceState.Primary.Attributes["repository"],
						resourceState.Primary.ID,
					), nil
				},
				ImportStateVerifyIgnore: []string{
					"identifier", // Ignore identifier as it's used for creation but not returned in read
				},
			},
		},
	})
}

//nolint:goerr113
func testAccEntitlementControlCheckDestroy(resourceName string) resource.TestCheckFunc {
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
		identifier := resourceState.Primary.ID

		req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, identifier)
		entitlement, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify entitlement control state: %w", err)
		} else if is200(resp) && entitlement.GetIsActive() {
			return fmt.Errorf("unable to verify entitlement control state: still enabled: %s/%s/%s", namespace, repository, identifier)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:goerr113
func testAccEntitlementControlCheckExists(resourceName string) resource.TestCheckFunc {
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
		identifier := resourceState.Primary.ID

		req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, identifier)
		_, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req)
		if err != nil {
			return fmt.Errorf("unable to verify entitlement control existence: %w", err)
		}
		defer resp.Body.Close()

		return nil
	}
}

var testAccEntitlementControlConfigBasic = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-ent-ctrl"
	namespace = "%s"
}

data "cloudsmith_entitlement_list" "test" {
    namespace  = resource.cloudsmith_repository.test.namespace
    repository = resource.cloudsmith_repository.test.slug_perm
    query      = ["name:Default"]
}

resource "cloudsmith_entitlement_control" "test" {
    namespace  = resource.cloudsmith_repository.test.namespace
    repository = resource.cloudsmith_repository.test.slug_perm
    identifier = data.cloudsmith_entitlement_list.test.entitlement_tokens[0].slug_perm
    enabled    = false
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccEntitlementControlConfigBasicUpdate = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-ent-ctrl"
	namespace = "%s"
}

data "cloudsmith_entitlement_list" "test" {
    namespace  = resource.cloudsmith_repository.test.namespace
    repository = resource.cloudsmith_repository.test.slug_perm
    query      = ["name:Default"]
}

resource "cloudsmith_entitlement_control" "test" {
    namespace  = resource.cloudsmith_repository.test.namespace
    repository = resource.cloudsmith_repository.test.slug_perm
    identifier = data.cloudsmith_entitlement_list.test.entitlement_tokens[0].slug_perm
    enabled    = true
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
