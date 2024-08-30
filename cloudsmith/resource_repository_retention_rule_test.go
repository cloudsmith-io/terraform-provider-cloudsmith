//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

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
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_count_limit", "100"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_days_limit", "28"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_enabled", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_group_by_name", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_group_by_format", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_group_by_package_type", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_size_limit", "0"),
				),
			},
			{
				Config: testAccRepositoryRetentionRuleConfigBasicUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_count_limit", "11"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_days_limit", "20"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_group_by_name", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_size_limit", "20000000"),
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
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccRepositoryRetentionRuleConfigBasicUpdate = fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
  name        = "terraform-acc-repo-retention-rule"
  namespace   = "%s"
}

resource "cloudsmith_repository_retention_rule" "test" {
  namespace = "%s"
  repository = cloudsmith_repository.test-retention.name
  retention_count_limit = 11
  retention_days_limit = 20
  retention_size_limit = 20000000
  retention_enabled = true
  retention_group_by_name = false
  retention_group_by_format = true
  retention_group_by_package_type = true
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))
