// Copyright Cloudsmith 2026
// SPDX-License-Identifier: MPL-2.0

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

// TestAccRepository_basic spins up a repository with all default options,
// verifies it exists and checks the name is set correctly. Then it changes the
// name, and verifies it's been set correctly before tearing down the resource
// and verifying deletion.
//
// NOTE: It is not necessary to check properties that have been explicitly set
// as Terraform performs a drift/plan check after every step anyway. Only
// computed properties need explicitly checked.
func TestAccRepository_basic(t *testing.T) {
	t.Parallel()

	repositoryName := testAccUniqueRepositoryName("terraform-acc-test")
	repositoryUpdatedName := testAccUniqueRepositoryName("terraform-acc-test-update")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryConfigBasic(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test"),
					// check a sample of computed properties have been set correctly
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "contextual_auth_realm", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "copy_own", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "copy_packages", "Read"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "docker_refresh_tokens_enabled", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "is_private", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "is_public", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "replace_packages_by_default", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "use_vulnerability_scanning", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "broadcast_state", "Off"),
				),
			},
			{
				Config: testAccRepositoryConfigBasicUpdateName(repositoryUpdatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test"),
				),
			},
			{
				Config:      testAccRepositoryConfigBasicInvalidProp(repositoryUpdatedName),
				ExpectError: regexp.MustCompile("expected copy_packages to be one of"),
			},
			{
				Config:      testAccRepositoryConfigBasicInvalidBroadcastState(repositoryUpdatedName),
				ExpectError: regexp.MustCompile("expected broadcast_state to be one of"),
			},
			{
				Config: testAccRepositoryConfigBasicUpdateProps(repositoryUpdatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "tag_pre_releases_as_latest", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "manage_entitlements_privilege", "Write"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "use_entitlements_privilege", "Admin"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "broadcast_state", "Private"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "cosign_signing_enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "nuget_native_signing_enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "npm_upstream_tags_take_precedence", "true"),
				),
			},
			{
				ResourceName: "cloudsmith_repository.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources["cloudsmith_repository.test"]
					return fmt.Sprintf(
						"%s.%s",
						resourceState.Primary.Attributes["namespace"],
						resourceState.Primary.Attributes["slug"],
					), nil
				},
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"wait_for_deletion"},
			},
		},
	})
}

//nolint:err113
func testAccRepositoryCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.ReposApi.ReposRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify repository deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify repository deletion: still exists: %s/%s", os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:err113
func testAccRepositoryCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		req := pc.APIClient.ReposApi.ReposRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	}
}

func testAccRepositoryConfigBasic(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccRepositoryConfigBasicUpdateName(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccRepositoryConfigBasicInvalidProp(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"

	copy_packages = "Sudo"
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccRepositoryConfigBasicInvalidBroadcastState(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"

	broadcast_state = "InvalidState"
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccRepositoryConfigBasicUpdateProps(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"

	contextual_auth_realm             = false
	copy_packages                     = "Write"
	docker_refresh_tokens_enabled     = true
	replace_packages_by_default       = true
	use_vulnerability_scanning        = false
	tag_pre_releases_as_latest        = true
	manage_entitlements_privilege     = "Write"
	use_entitlements_privilege        = "Admin"
	broadcast_state                   = "Private"
	cosign_signing_enabled            = true
	nuget_native_signing_enabled      = true
	npm_upstream_tags_take_precedence = true
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

// TestAccRepositoryRetentionRule_subresource verifies the `retention_rule`
// sub-resource: that it is applied on create, updated in place, that only one
// rule can be declared per repository, and that removing the block disables
// retention.
func TestAccRepositoryRetentionRule_subresource(t *testing.T) {
	t.Parallel()

	repositoryName := testAccUniqueRepositoryName("terraform-acc-repo-retention")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository.test-retention"),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryConfigRetentionRule(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test-retention", "retention_rule.#", "1"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test-retention", "retention_rule.0.retention_enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test-retention", "retention_rule.0.retention_count_limit", "100"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test-retention", "retention_rule.0.retention_days_limit", "28"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test-retention", "retention_rule.0.retention_package_query_string", "name:test"),
				),
			},
			{
				Config: testAccRepositoryConfigRetentionRuleUpdate(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test-retention", "retention_rule.0.retention_count_limit", "0"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test-retention", "retention_rule.0.retention_days_limit", "0"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test-retention", "retention_rule.0.retention_size_limit", "0"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test-retention", "retention_rule.0.retention_group_by_name", "true"),
				),
			},
			{
				Config:      testAccRepositoryConfigRetentionRuleDuplicate(repositoryName),
				ExpectError: regexp.MustCompile(`No more than 1 "retention_rule" blocks are allowed`),
			},
			{
				Config: testAccRepositoryConfigBasicRetention(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test-retention", "retention_rule.#", "0"),
					testAccRepositoryCheckRetentionDisabled("cloudsmith_repository.test-retention"),
				),
			},
		},
	})
}

//nolint:err113
func testAccRepositoryCheckRetentionDisabled(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		pc := testAccProvider.Meta().(*providerConfig)

		rule, _, err := pc.APIClient.ReposApi.RepoRetentionRead(
			pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID,
		).Execute()
		if err != nil {
			return fmt.Errorf("unable to read repository retention rule: %w", err)
		}

		if rule.GetRetentionEnabled() {
			return fmt.Errorf("retention is still enabled for repository: %s", resourceState.Primary.ID)
		}

		return nil
	}
}

func testAccRepositoryConfigBasicRetention(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
	name      = "%s"
	namespace = "%s"
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccRepositoryConfigRetentionRule(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
	name      = "%s"
	namespace = "%s"

	retention_rule {
		retention_enabled               = true
		retention_count_limit           = 100
		retention_days_limit            = 28
		retention_group_by_name         = false
		retention_group_by_format       = false
		retention_group_by_package_type = false
		retention_size_limit            = 0
		retention_package_query_string  = "name:test"
	}
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccRepositoryConfigRetentionRuleUpdate(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
	name      = "%s"
	namespace = "%s"

	retention_rule {
		retention_enabled               = true
		retention_count_limit           = 0
		retention_days_limit            = 0
		retention_group_by_name         = true
		retention_group_by_format       = false
		retention_group_by_package_type = false
		retention_size_limit            = 0
		retention_package_query_string  = "name:test"
	}
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccRepositoryConfigRetentionRuleDuplicate(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
	name      = "%s"
	namespace = "%s"

	retention_rule {
		retention_enabled     = true
		retention_count_limit = 100
	}

	retention_rule {
		retention_enabled     = true
		retention_count_limit = 50
	}
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}
