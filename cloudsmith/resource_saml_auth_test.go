package cloudsmith

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccSAMLAuth_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccSAMLAuthCheckDestroy("cloudsmith_saml_auth.test"),
		Steps: []resource.TestStep{
			{
				// Basic configuration with URL-based metadata
				Config: testAccSAMLAuthConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccSAMLAuthCheckExists("cloudsmith_saml_auth.test"),
					resource.TestCheckResourceAttr("cloudsmith_saml_auth.test", "saml_auth_enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_saml_auth.test", "saml_auth_enforced", "false"),
					resource.TestCheckResourceAttr("cloudsmith_saml_auth.test", "saml_metadata_url", "https://test.idp.example.com/metadata.xml"),
				),
			},
			{
				// Update to use inline metadata
				Config: testAccSAMLAuthConfigInlineMetadata,
				Check: resource.ComposeTestCheckFunc(
					testAccSAMLAuthCheckExists("cloudsmith_saml_auth.test"),
					resource.TestCheckResourceAttr("cloudsmith_saml_auth.test", "saml_metadata_inline", testSAMLMetadata),
					resource.TestCheckResourceAttr("cloudsmith_saml_auth.test", "saml_metadata_url", ""),
				),
			},
			{
				// Enable enforcement
				Config: testAccSAMLAuthConfigEnforced,
				Check: resource.ComposeTestCheckFunc(
					testAccSAMLAuthCheckExists("cloudsmith_saml_auth.test"),
					resource.TestCheckResourceAttr("cloudsmith_saml_auth.test", "saml_auth_enforced", "true"),
				),
			},
			{
				ResourceName: "cloudsmith_saml_auth.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return os.Getenv("CLOUDSMITH_NAMESPACE"), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSAMLAuthCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)
		organization := resourceState.Primary.Attributes["organization"]

		req := pc.APIClient.OrgsApi.OrgsSamlAuthenticationRead(pc.Auth, organization)
		samlAuth, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify SAML auth deletion: %w", err)
		}
		defer resp.Body.Close()

		// Resource is considered destroyed if SAML auth is disabled
		if samlAuth != nil && samlAuth.GetSamlAuthEnabled() {
			return fmt.Errorf("SAML authentication still enabled for organization: %s", organization)
		}

		return nil
	}
}

func testAccSAMLAuthCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)
		organization := resourceState.Primary.Attributes["organization"]

		req := pc.APIClient.OrgsApi.OrgsSamlAuthenticationRead(pc.Auth, organization)
		_, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationReadExecute(req)
		if err != nil {
			return fmt.Errorf("error checking SAML auth existence: %w", err)
		}
		defer resp.Body.Close()

		return nil
	}
}

// Sample SAML metadata for testing
var testSAMLMetadata = strings.TrimSpace(`<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata">
  <IDPSSODescriptor>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
                        Location="https://test.idp.example.com/sso"/>
  </IDPSSODescriptor>
</EntityDescriptor>`)

var testAccSAMLAuthConfigBasic = strings.TrimSpace(fmt.Sprintf(`
resource "cloudsmith_saml_auth" "test" {
    organization        = "%s"
    saml_auth_enabled  = true
    saml_auth_enforced = false
    saml_metadata_url  = "https://test.idp.example.com/metadata.xml"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE")))

var testAccSAMLAuthConfigInlineMetadata = strings.TrimSpace(fmt.Sprintf(`
resource "cloudsmith_saml_auth" "test" {
    organization        = "%s"
    saml_auth_enabled  = true
    saml_auth_enforced = false
    saml_metadata_inline = <<EOF
%s
EOF
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), testSAMLMetadata))

var testAccSAMLAuthConfigEnforced = strings.TrimSpace(fmt.Sprintf(`
resource "cloudsmith_saml_auth" "test" {
    organization        = "%s"
    saml_auth_enabled  = true
    saml_auth_enforced = true
    saml_metadata_inline = <<EOF
%s
EOF
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), testSAMLMetadata))
