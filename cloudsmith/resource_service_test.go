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

// TestAccService_basic spins up a service with all default options, verifies it
// exists and checks the attributes. Then it performs some updates and verifies
// them before tearing down the resources and verifying deletion.
//
// NOTE: It is not necessary to check properties that have been explicitly set
// as Terraform performs a drift/plan check after every step anyway. Only
// computed properties need explicitly checked.
// TestAccService_basic runs a series of tests for the cloudsmith_service resource.
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
					resource.TestMatchResourceAttr("cloudsmith_service.test", "slug", regexp.MustCompile("^tf-test-service.*$")),
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
					resource.TestMatchTypeSetElemNestedAttrs("cloudsmith_service.test", "team.*", map[string]*regexp.Regexp{
						"slug": regexp.MustCompile("^tf-test-team-svc(-[^2].*)?$"),
						"role": regexp.MustCompile("^Member$"),
					}),
				),
			},
			{
				Config: testAccServiceConfigBasicAddAnotherToTeam,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					resource.TestCheckResourceAttrSet("cloudsmith_service.test", "team.#"),

					resource.TestMatchTypeSetElemNestedAttrs("cloudsmith_service.test", "team.*", map[string]*regexp.Regexp{
						"slug": regexp.MustCompile("^tf-test-team-svc(-[^2].*)?$"),
						"role": regexp.MustCompile("^Member$"),
					}),

					resource.TestMatchTypeSetElemNestedAttrs("cloudsmith_service.test", "team.*", map[string]*regexp.Regexp{
						"slug": regexp.MustCompile("^tf-test-team-svc-2.*$"),
						"role": regexp.MustCompile("^Manager$"),
					}),
				),
			},
			{
				Config: testAccServiceConfigNoAPIKey,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					// check that the key attribute is explicitly an empty string
					resource.TestCheckResourceAttr("cloudsmith_service.test", "key", "**redacted**"),
				),
			},
			{
				Config: testAccServiceConfigRotateAPIKeyFirst,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					// key should be present in state when store_api_key is true
					resource.TestCheckResourceAttrSet("cloudsmith_service.test", "key"),
				),
			},
			{
				Config: testAccServiceConfigRotateAPIKeySecond,
				Check: resource.ComposeTestCheckFunc(
					// ensure the resource still exists after rotation
					testAccServiceCheckExists("cloudsmith_service.test"),
					// key should still be set after rotation; we don't assert the value
					resource.TestCheckResourceAttrSet("cloudsmith_service.test", "key"),
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
				ImportStateVerifyIgnore: []string{"key", "store_api_key", "rotate_api_key"},
			},
		},
	})
}

// TestAccService_rotate focuses specifically on exercising the rotate_api_key
// trigger to ensure that rotating a service account's API key works without
// involving team assignments or import behaviour.
func TestAccService_rotate(t *testing.T) {
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
					// initial key should be present when store_api_key is true (default)
					resource.TestCheckResourceAttrSet("cloudsmith_service.test", "key"),
				),
			},
			{
				Config: testAccServiceConfigRotateAPIKeyFirst,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					// key should still be set after first rotation
					resource.TestCheckResourceAttrSet("cloudsmith_service.test", "key"),
				),
			},
			{
				Config: testAccServiceConfigRotateAPIKeySecond,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceCheckExists("cloudsmith_service.test"),
					// key should remain set after subsequent rotations
					resource.TestCheckResourceAttrSet("cloudsmith_service.test", "key"),
				),
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

var testAccServiceConfigNoAPIKey = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
	name            = "TF Test Service No API Key"
	organization    = "%s"
	store_api_key  = false
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccServiceConfigBasic = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
	name         = "TF Test Service cs"
	organization = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccServiceConfigBasicUpdateName = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
	name         = "TF Test Service Updated"
	organization = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccServiceConfigRotateAPIKeyFirst = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
	name          = "TF Test Service cs"
	organization  = "%s"
	rotate_api_key = "first-rotation"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccServiceConfigRotateAPIKeySecond = fmt.Sprintf(`
resource "cloudsmith_service" "test" {
	name          = "TF Test Service cs"
	organization  = "%s"
	rotate_api_key = "second-rotation"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccServiceConfigBasicAddToTeam = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	name         = "TF Test Team Svc"
	organization = "%s"
}

resource "cloudsmith_service" "test" {
	name         = "TF Test Service cs"
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
	name         = "TF Test Service cs"
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
