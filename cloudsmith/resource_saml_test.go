package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccSaml_basic(t *testing.T) {
	t.Parallel()

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
					resource.TestCheckResourceAttr("cloudsmith_saml.test", "team", "test-team"),
				),
			},
		},
	})
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
	team 		= cloudsmith_team.test.slug
}`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))
