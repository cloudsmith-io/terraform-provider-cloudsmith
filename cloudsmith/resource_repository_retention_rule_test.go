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

	repositoryName := testAccUniqueRepositoryName("terraform-acc-repo-retention-rule")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryRetentionRuleConfigRepository(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
				),
			},
			{
				Config: testAccRepositoryRetentionRuleConfigBasic(repositoryName),
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
				Config: testAccRepositoryRetentionRuleConfigZero(repositoryName),
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

func testAccRepositoryRetentionRuleConfigRepository(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
	name      = "%s"
	namespace = "%s"
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccRepositoryRetentionRuleConfigBasic(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
	name      = "%s"
	namespace = "%s"
}

resource "cloudsmith_repository_retention_rule" "test" {
	namespace = cloudsmith_repository.test-retention.namespace
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
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccRepositoryRetentionRuleConfigZero(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
	name      = "%s"
	namespace = "%s"
}

resource "cloudsmith_repository_retention_rule" "test" {
	namespace = cloudsmith_repository.test-retention.namespace
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
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}
