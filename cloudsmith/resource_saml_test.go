package cloudsmith

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccSaml_basic(t *testing.T) {
	t.Parallel()

	var initialEnabled bool
	var initialEnabledCaptured bool
	var mappingID string

	t.Cleanup(func() {
		if !initialEnabledCaptured {
			return
		}
		if err := testAccSamlRestoreEnabled(initialEnabled); err != nil {
			t.Errorf("error restoring initial SAML Group Sync status: %v", err)
		}
	})

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccSamlCheckDestroy("cloudsmith_saml.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccSamlConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("cloudsmith_team.test"),
					testAccSamlCheckExists("cloudsmith_saml.test"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "organization", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "idp_key", "test-idp-key"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "idp_value", "test-idp-value"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "role", "Member"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "team", "test-team"),
					func(s *terraform.State) error {
						enabled, err := strconv.ParseBool(s.RootModule().Resources["cloudsmith_saml.test"].Primary.Attributes["enabled"])
						if err != nil {
							return fmt.Errorf("invalid SAML Group Sync status in state: %w", err)
						}
						initialEnabled = enabled
						initialEnabledCaptured = true
						return nil
					},
				),
			},
			{
				Config: testAccSamlConfigBasicUpdateRole,
				Check: resource.ComposeTestCheckFunc(
					testAccSamlCheckExists("cloudsmith_saml.test"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "organization", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "idp_key", "test-idp-key"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "idp_value", "test-idp-value"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "role", "Manager"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "team", "test-team"),
					func(s *terraform.State) error {
						got := s.RootModule().Resources["cloudsmith_saml.test"].Primary.Attributes["enabled"]
						want := strconv.FormatBool(initialEnabled)
						if got != want {
							return fmt.Errorf("omitting enabled changed SAML Group Sync status: got %q, want %q", got, want)
						}
						return nil
					},
				),
			},
			{
				Config: testAccSamlConfigBasicUpdateIDP,
				Check: resource.ComposeTestCheckFunc(
					testAccSamlCheckExists("cloudsmith_saml.test"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "organization", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "idp_key", "test-idp-key-updated"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "idp_value", "test-idp-value-updated"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "role", "Manager"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "enabled", "false"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "team", "test-team"),
					func(s *terraform.State) error {
						mappingID = s.RootModule().Resources["cloudsmith_saml.test"].Primary.ID
						return nil
					},
				),
			},
			{
				Config: testAccSamlConfigBasicUpdateEnabled,
				Check: resource.ComposeTestCheckFunc(
					testAccSamlCheckExists("cloudsmith_saml.test"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "organization", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "idp_key", "test-idp-key-updated"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "idp_value", "test-idp-value-updated"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "role", "Manager"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "team", "test-team"),
					func(s *terraform.State) error {
						got := s.RootModule().Resources["cloudsmith_saml.test"].Primary.ID
						if got != mappingID {
							return fmt.Errorf("SAML mapping was recreated during enabled-only update: got ID %q, want %q", got, mappingID)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccSamlRestoreEnabled(enabled bool) error {
	pc := testAccProvider.Meta().(*providerConfig)
	organization := os.Getenv("CLOUDSMITH_NAMESPACE")

	if enabled {
		req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncEnable(pc.Auth, organization)
		_, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncEnableExecute(req)
		return err
	}

	req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncDisable(pc.Auth, organization)
	_, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncDisableExecute(req)
	return err
}

func testAccSamlCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("saml resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("saml resource ID not set")
		}

		c := testAccProvider.Meta().(*providerConfig)
		samlResources, _, err := c.APIClient.OrgsApi.OrgsSamlGroupSyncList(c.Auth, rs.Primary.Attributes["organization"]).Execute()
		if err != nil {
			return fmt.Errorf("error checking saml resource: %w", err)
		}

		for _, samlResource := range samlResources {
			if *samlResource.SlugPerm == rs.Primary.ID {
				return fmt.Errorf("saml resource still exists: %s", rs.Primary.ID)
			}
		}

		return nil
	}
}

// testAccSamlCheckExists verifies the SAML resource exists
func testAccSamlCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("saml resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("saml resource ID not set")
		}

		c := testAccProvider.Meta().(*providerConfig)
		_, resp, err := c.APIClient.OrgsApi.OrgsSamlGroupSyncList(c.Auth, rs.Primary.Attributes["organization"]).Execute()
		if err != nil {
			return fmt.Errorf("error checking saml resource: %w", err)
		}

		if resp != nil && is404(resp) {
			return fmt.Errorf("saml resource not found: %s", rs.Primary.ID)
		}

		return nil
	}
}

// create configs

var testAccSamlConfigBasic = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	organization = "%s"
	name      = "test-team"
}

resource "cloudsmith_saml" "test" {
	organization = "%s"
	idp_key 	= "test-idp-key"
	idp_value 	= "test-idp-value"
	role 		= "Member"
	team 		= cloudsmith_team.test.slug
}`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccSamlConfigBasicUpdateRole = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	organization = "%s"
	name      = "test-team"
}

resource "cloudsmith_saml" "test" {
	organization = "%s"
	idp_key 	= "test-idp-key"
	idp_value 	= "test-idp-value"
	role 		= "Manager"
	team 		= cloudsmith_team.test.slug
}`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccSamlConfigBasicUpdateIDP = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	organization = "%s"
	name      = "test-team"
}

resource "cloudsmith_saml" "test" {
	organization = "%s"
	idp_key 	= "test-idp-key-updated"
	idp_value 	= "test-idp-value-updated"
	role 		= "Manager"
	enabled 	= false
	team 		= cloudsmith_team.test.slug
}`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccSamlConfigBasicUpdateEnabled = fmt.Sprintf(`
resource "cloudsmith_team" "test" {
	organization = "%s"
	name      = "test-team"
}

resource "cloudsmith_saml" "test" {
	organization = "%s"
	idp_key 	= "test-idp-key-updated"
	idp_value 	= "test-idp-value-updated"
	role 		= "Manager"
	enabled 	= true
	team 		= cloudsmith_team.test.slug
}`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))
