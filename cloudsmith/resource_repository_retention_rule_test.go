//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// testCheckResourceAttrWithMessage enhances output for attribute checks
func testCheckResourceAttrWithMessage(resourceName, attrName, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		actual, ok := rs.Primary.Attributes[attrName]
		if !ok {
			return fmt.Errorf("Attribute '%s' not found in resource '%s'. State: %#v", attrName, resourceName, rs.Primary.Attributes)
		}
		if actual != expected {
			return fmt.Errorf("Attribute '%s' in resource '%s' expected '%s', got '%s'. Full state: %#v", attrName, resourceName, expected, actual, rs.Primary.Attributes)
		}
		return nil
	}
}

func TestAccRepositoryRetentionRule_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
				),
			},
			{
				Config: testAccRepositoryRetentionRuleConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_count_limit", "100"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_days_limit", "28"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_enabled", "false"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_group_by_name", "false"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_group_by_format", "false"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_group_by_package_type", "false"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_size_limit", "0"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_package_query_string", "name:test"),
				),
			},
			{
				Config: testAccRepositoryRetentionRuleConfigZero,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_count_limit", "0"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_days_limit", "0"),
					testCheckResourceAttrWithMessage("cloudsmith_repository_retention_rule.test", "retention_size_limit", "0"),
				),
			},
		},
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository.test-retention"),
	})
}

var testAccRepositoryConfig = fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
	name        = "terraform-acc-repo-retention-rule"
	namespace   = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccRepositoryRetentionRuleConfigBasic = fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
	name        = "terraform-acc-repo-retention-rule"
	namespace   = "%s"
}

resource "cloudsmith_repository_retention_rule" "test" {
	namespace = "%s"
	repository = cloudsmith_repository.test-retention.name
	retention_enabled = false
	retention_count_limit = 100
	retention_days_limit = 28
	retention_group_by_name = false
	retention_group_by_format = false
	retention_group_by_package_type = false
	retention_size_limit = 0
	retention_package_query_string = "name:test"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccRepositoryRetentionRuleConfigZero = fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
	name        = "terraform-acc-repo-retention-rule"
	namespace   = "%s"
}

resource "cloudsmith_repository_retention_rule" "test" {
	namespace = "%s"
	repository = cloudsmith_repository.test-retention.name
	retention_enabled = false
	retention_count_limit = 0
	retention_days_limit = 0
	retention_group_by_name = false
	retention_group_by_format = false
	retention_group_by_package_type = false
	retention_size_limit = 0
	retention_package_query_string = "name:test"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))
