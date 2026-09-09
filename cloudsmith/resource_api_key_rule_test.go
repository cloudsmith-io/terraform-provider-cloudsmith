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

func TestAccOrgAPIKeyRule_userAccounts(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testOrgAPIKeyRuleCheckDestroy("cloudsmith_api_key_rule.test_user"),
		Steps: []resource.TestStep{
			{
				Config: testOrgAPIKeyRuleUserBasic,
				Check: resource.ComposeTestCheckFunc(
					testOrgAPIKeyRuleCheckExists("cloudsmith_api_key_rule.test_user"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_user", "rule_type", "User Accounts"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_user", "max_age_hours", "876000"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_user", "is_enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_user", "enforce_refresh", "false"),
					// check computed properties have been set correctly
					resource.TestCheckResourceAttrSet("cloudsmith_api_key_rule.test_user", "created_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_api_key_rule.test_user", "updated_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_api_key_rule.test_user", "slug"),
					resource.TestCheckResourceAttrSet("cloudsmith_api_key_rule.test_user", "slug_perm"),
				),
			},
			{
				Config:      testOrgAPIKeyRuleUserInvalidRuleType,
				ExpectError: regexp.MustCompile(`expected rule_type to be one of`),
			},
			{
				Config:      testOrgAPIKeyRuleUserBelowMinimumAge,
				ExpectError: regexp.MustCompile(`expected max_age_hours to be at least`),
			},
			{
				Config: testOrgAPIKeyRuleUserUpdate,
				Check: resource.ComposeTestCheckFunc(
					testOrgAPIKeyRuleCheckExists("cloudsmith_api_key_rule.test_user"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_user", "rule_type", "User Accounts"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_user", "max_age_hours", "876024"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_user", "is_enabled", "false"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_user", "enforce_refresh", "true"),
					resource.TestCheckResourceAttrSet("cloudsmith_api_key_rule.test_user", "updated_at"),
				),
			},
			{
				ResourceName:      "cloudsmith_api_key_rule.test_user",
				ImportState:       true,
				ImportStateIdFunc: testOrgAPIKeyRuleImportStateID("cloudsmith_api_key_rule.test_user"),
				ImportStateVerify: true,
				// last_applied_at is stamped by the expiry task, which can run
				// between the apply and the import.
				ImportStateVerifyIgnore: []string{"last_applied_at"},
			},
		},
	})
}

func TestAccOrgAPIKeyRule_serviceAccounts(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testOrgAPIKeyRuleCheckDestroy("cloudsmith_api_key_rule.test_service"),
		Steps: []resource.TestStep{
			{
				Config: testOrgAPIKeyRuleServiceBasic,
				Check: resource.ComposeTestCheckFunc(
					testOrgAPIKeyRuleCheckExists("cloudsmith_api_key_rule.test_service"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_service", "rule_type", "Service Accounts"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_service", "max_age_hours", "876000"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_service", "is_enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_service", "enforce_refresh", "false"),
					// check computed properties have been set correctly
					resource.TestCheckResourceAttrSet("cloudsmith_api_key_rule.test_service", "created_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_api_key_rule.test_service", "updated_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_api_key_rule.test_service", "slug"),
					resource.TestCheckResourceAttrSet("cloudsmith_api_key_rule.test_service", "slug_perm"),
				),
			},
			{
				Config: testOrgAPIKeyRuleServiceUpdate,
				Check: resource.ComposeTestCheckFunc(
					testOrgAPIKeyRuleCheckExists("cloudsmith_api_key_rule.test_service"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_service", "rule_type", "Service Accounts"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_service", "max_age_hours", "876024"),
					resource.TestCheckResourceAttr("cloudsmith_api_key_rule.test_service", "enforce_refresh", "true"),
					resource.TestCheckResourceAttrSet("cloudsmith_api_key_rule.test_service", "updated_at"),
				),
			},
			{
				ResourceName:      "cloudsmith_api_key_rule.test_service",
				ImportState:       true,
				ImportStateIdFunc: testOrgAPIKeyRuleImportStateID("cloudsmith_api_key_rule.test_service"),
				ImportStateVerify: true,
				// last_applied_at is stamped by the expiry task, which can run
				// between the apply and the import.
				ImportStateVerifyIgnore: []string{"last_applied_at"},
			},
		},
	})
}

//nolint:err113
func testOrgAPIKeyRuleCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.OrgsApi.OrgsApiKeyRulesRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.OrgsApi.OrgsApiKeyRulesReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify api key rule deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify api key rule deletion: still exists: %s/%s", os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:err113
func testOrgAPIKeyRuleCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.OrgsApi.OrgsApiKeyRulesRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.OrgsApi.OrgsApiKeyRulesReadExecute(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:err113
func testOrgAPIKeyRuleImportStateID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		return fmt.Sprintf(
			"%s.%s",
			resourceState.Primary.Attributes["organization"],
			resourceState.Primary.Attributes["slug_perm"],
		), nil
	}
}

var testOrgAPIKeyRuleUserBasic = fmt.Sprintf(`
resource "cloudsmith_api_key_rule" "test_user" {
	organization    = "%s"
	rule_type       = "User Accounts"
	max_age_hours   = 876000
	is_enabled      = true
	enforce_refresh = false
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testOrgAPIKeyRuleUserInvalidRuleType = fmt.Sprintf(`
resource "cloudsmith_api_key_rule" "test_user" {
	organization    = "%s"
	rule_type       = "All Accounts"
	max_age_hours   = 876000
	is_enabled      = true
	enforce_refresh = false
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testOrgAPIKeyRuleUserBelowMinimumAge = fmt.Sprintf(`
resource "cloudsmith_api_key_rule" "test_user" {
	organization    = "%s"
	rule_type       = "User Accounts"
	max_age_hours   = 1
	is_enabled      = true
	enforce_refresh = false
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testOrgAPIKeyRuleUserUpdate = fmt.Sprintf(`
resource "cloudsmith_api_key_rule" "test_user" {
	organization    = "%s"
	rule_type       = "User Accounts"
	max_age_hours   = 876024
	is_enabled      = false
	enforce_refresh = true
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testOrgAPIKeyRuleServiceBasic = fmt.Sprintf(`
resource "cloudsmith_api_key_rule" "test_service" {
	organization    = "%s"
	rule_type       = "Service Accounts"
	max_age_hours   = 876000
	is_enabled      = true
	enforce_refresh = false
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testOrgAPIKeyRuleServiceUpdate = fmt.Sprintf(`
resource "cloudsmith_api_key_rule" "test_service" {
	organization    = "%s"
	rule_type       = "Service Accounts"
	max_age_hours   = 876024
	is_enabled      = true
	enforce_refresh = true
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
