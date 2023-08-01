//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccService_basic spins up a service with all default options, verifies it
// exists and checks the attributes. Then it performs some updates and verifies
// them before tearing down the resources and verifying deletion.
//
// NOTE: It is not necessary to check properties that have been explicitly set
// as Terraform performs a drift/plan check after every step anyway. Only
// computed properties need explicitly checked.
func TestAccService_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccServiceCheckDestroy("cloudsmith_service.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccServiceConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					// check a sample of computed properties have been set correctly
					resource.TestCheckResourceAttr("cloudsmith_service.test", "description", ""),
					resource.TestCheckResourceAttr("cloudsmith_service.test", "slug", "tf-test-service"),
					resource.TestCheckResourceAttrSet("cloudsmith_service.test", "key"),
					resource.TestCheckResourceAttr("cloudsmith_service.test", "role", "Member"),
					resource.TestCheckNoResourceAttr("cloudsmith_service.test", "team.#"),
				),
			},
			{
				Config: testAccServiceConfigBasicUpdateName,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
				),
			},
			{
				Config: testAccServiceConfigBasicAddToTeam,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					resource.TestCheckResourceAttrSet("cloudsmith_service.test", "team.#"),
					resource.TestCheckTypeSetElemNestedAttrs("cloudsmith_service.test", "team.*", map[string]string{
						"slug": "tf-test-team-svc",
						"role": "Member",
					}),
				),
			},
			{
				Config: testAccServiceConfigBasicAddAnotherToTeam,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					resource.TestCheckResourceAttrSet("cloudsmith_service.test", "team.#"),
					resource.TestCheckTypeSetElemNestedAttrs("cloudsmith_service.test", "team.*", map[string]string{
						"slug": "tf-test-team-svc",
						"role": "Member",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("cloudsmith_service.test", "team.*", map[string]string{
						"slug": "tf-test-team-svc-2",
						"role": "Manager",
					}),
				),
			},
			{
				ResourceName: "cloudsmith_service.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources["cloudsmith_service.test"]
					return fmt.Sprintf(
						"%s.%s",
						resourceState.Primary.Attributes["organization"],
						resourceState.Primary.Attributes["slug"],
					), nil
				},
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key", "warn_on_key_difference"}, // Exclude warn_on_key_difference from the verification
			},
		},
	})
}

//nolint:goerr113
func testAccServiceCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.OrgsApi.OrgsServicesRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.OrgsApi.OrgsServicesReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify service deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify service deletion: still exists: %s/%s", os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:goerr113
func testAccServiceCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.OrgsApi.OrgsServicesRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.OrgsApi.OrgsServicesReadExecute(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	}
}

var testAccServiceConfigBasic = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
	name         = "TF Test Service"
	organization = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccServiceConfigBasicUpdateName = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
	name         = "TF Test Service Updated"
	organization = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccServiceConfigBasicAddToTeam = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	name         = "TF Test Team Svc"
	organization = "%s"
}

resource "cloudsmith_service" "test" {
	name         = "TF Test Service"
	organization = "%s"
	role         = "Manager"

	team {
		slug = cloudsmith_team.test.slug
	}
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccServiceConfigBasicAddAnotherToTeam = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	name         = "TF Test Team Svc"
	organization = "%s"
}

resource "cloudsmith_team" "test2" {
	name         = "TF Test Team Svc 2"
	organization = "%s"
}

resource "cloudsmith_service" "test" {
	name         = "TF Test Service"
	organization = "%s"
	role         = "Manager"

	team {
		slug = cloudsmith_team.test.slug
	}

	team {
		role = "Manager"
		slug = cloudsmith_team.test2.slug
	}
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))
