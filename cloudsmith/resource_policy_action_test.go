package cloudsmith

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-go-v2/models/apierrors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPolicyAction_setPackageState(t *testing.T) {
	t.Parallel()

	parentName := testAccUniquePolicyName("TF Acc Policy For SetState")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccPolicyActionCheckDestroy("cloudsmith_policy_action.test"),
			testAccPolicyCheckDestroy("cloudsmith_policy.parent"),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyActionConfigSetPackageState(parentName, "QUARANTINED"),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyActionCheckExists("cloudsmith_policy_action.test"),
					resource.TestCheckResourceAttrSet("cloudsmith_policy_action.test", "slug_perm"),
					resource.TestCheckResourceAttrSet("cloudsmith_policy_action.test", "created_at"),
					resource.TestCheckResourceAttr("cloudsmith_policy_action.test", "set_package_state.0.package_state", "QUARANTINED"),
				),
			},
			{
				Config: testAccPolicyActionConfigSetPackageState(parentName, "DELETED"),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyActionCheckExists("cloudsmith_policy_action.test"),
					resource.TestCheckResourceAttr("cloudsmith_policy_action.test", "set_package_state.0.package_state", "DELETED"),
				),
			},
		},
	})
}

func TestAccPolicyAction_setPackageStateAvailable(t *testing.T) {
	t.Parallel()

	parentName := testAccUniquePolicyName("TF Acc Policy For Available")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccPolicyActionCheckDestroy("cloudsmith_policy_action.test"),
			testAccPolicyCheckDestroy("cloudsmith_policy.parent"),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyActionConfigSetPackageState(parentName, "AVAILABLE"),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyActionCheckExists("cloudsmith_policy_action.test"),
					resource.TestCheckResourceAttr("cloudsmith_policy_action.test", "set_package_state.0.package_state", "AVAILABLE"),
				),
			},
		},
	})
}

func TestAccPolicyAction_addPackageTags(t *testing.T) {
	t.Parallel()

	parentName := testAccUniquePolicyName("TF Acc Policy For AddTags")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccPolicyActionCheckDestroy("cloudsmith_policy_action.test"),
			testAccPolicyCheckDestroy("cloudsmith_policy.parent"),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyActionConfigAddPackageTags(parentName, []string{"needs-review", "imported"}),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyActionCheckExists("cloudsmith_policy_action.test"),
					resource.TestCheckResourceAttr("cloudsmith_policy_action.test", "add_package_tags.0.tags.#", "2"),
				),
			},
			{
				Config: testAccPolicyActionConfigAddPackageTags(parentName, []string{"needs-review"}),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyActionCheckExists("cloudsmith_policy_action.test"),
					resource.TestCheckResourceAttr("cloudsmith_policy_action.test", "add_package_tags.0.tags.#", "1"),
				),
			},
		},
	})
}

func TestAccPolicyAction_removePackageTags(t *testing.T) {
	t.Parallel()

	parentName := testAccUniquePolicyName("TF Acc Policy For RemoveTags")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccPolicyActionCheckDestroy("cloudsmith_policy_action.test"),
			testAccPolicyCheckDestroy("cloudsmith_policy.parent"),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyActionConfigRemovePackageTags(parentName, []string{"needs-review"}),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyActionCheckExists("cloudsmith_policy_action.test"),
					resource.TestCheckResourceAttr("cloudsmith_policy_action.test", "remove_package_tags.0.tags.#", "1"),
				),
			},
		},
	})
}

func TestAccPolicyAction_typeSwitchForcesReplace(t *testing.T) {
	t.Parallel()

	parentName := testAccUniquePolicyName("TF Acc Policy For TypeSwitch")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccPolicyActionCheckDestroy("cloudsmith_policy_action.test"),
			testAccPolicyCheckDestroy("cloudsmith_policy.parent"),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyActionConfigSetPackageState(parentName, "QUARANTINED"),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyActionCheckExists("cloudsmith_policy_action.test"),
					resource.TestCheckResourceAttr("cloudsmith_policy_action.test", "set_package_state.0.package_state", "QUARANTINED"),
				),
			},
			{
				Config: testAccPolicyActionConfigAddPackageTags(parentName, []string{"needs-review", "imported"}),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyActionCheckExists("cloudsmith_policy_action.test"),
					resource.TestCheckResourceAttr("cloudsmith_policy_action.test", "add_package_tags.0.tags.#", "2"),
				),
			},
		},
	})
}

func TestAccPolicyAction_import(t *testing.T) {
	t.Parallel()

	parentName := testAccUniquePolicyName("TF Acc Policy For Import")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccPolicyActionCheckDestroy("cloudsmith_policy_action.test"),
			testAccPolicyCheckDestroy("cloudsmith_policy.parent"),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyActionConfigSetPackageState(parentName, "QUARANTINED"),
				Check:  testAccPolicyActionCheckExists("cloudsmith_policy_action.test"),
			},
			{
				ResourceName: "cloudsmith_policy_action.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["cloudsmith_policy_action.test"]
					return fmt.Sprintf(
						"%s.%s.%s",
						rs.Primary.Attributes["workspace"],
						rs.Primary.Attributes["policy_slug_perm"],
						rs.Primary.Attributes["slug_perm"],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPolicyActionCheckDestroy(resourceName string) resource.TestCheckFunc {
	return testAccRetry(10*time.Second, 500*time.Millisecond, func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		pc := testAccProvider.Meta().(*providerConfig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesActionsRetrieve(
			ctx,
			rs.Primary.ID,
			rs.Primary.Attributes["policy_slug_perm"],
			rs.Primary.Attributes["workspace"],
		)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verifying policy action deletion: %w", err)
		}
		if resp != nil && resp.PolicyAction != nil {
			return fmt.Errorf("policy action still exists: %s", rs.Primary.ID)
		}
		return nil
	})
}

func testAccPolicyActionCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		pc := testAccProvider.Meta().(*providerConfig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesActionsRetrieve(
			ctx,
			rs.Primary.ID,
			rs.Primary.Attributes["policy_slug_perm"],
			testAccNamespace(),
		)
		return err
	}
}

func testAccPolicyActionConfigParentBlock(name string) string {
	return fmt.Sprintf(`
resource "cloudsmith_policy" "parent" {
    workspace = "%s"
    name      = "%s"
    rego      = <<-EOT
        package cloudsmith.policy
        default allow := true
    EOT
}
`, testAccNamespace(), name)
}

func testAccPolicyActionConfigSetPackageState(parentName, state string) string {
	return testAccPolicyActionConfigParentBlock(parentName) + fmt.Sprintf(`
resource "cloudsmith_policy_action" "test" {
    workspace        = "%s"
    policy_slug_perm = cloudsmith_policy.parent.slug_perm

    set_package_state {
        package_state = "%s"
    }
}
`, testAccNamespace(), state)
}

func testAccPolicyActionConfigAddPackageTags(parentName string, tags []string) string {
	return testAccPolicyActionConfigParentBlock(parentName) + fmt.Sprintf(`
resource "cloudsmith_policy_action" "test" {
    workspace        = "%s"
    policy_slug_perm = cloudsmith_policy.parent.slug_perm

    add_package_tags {
        tags = %s
    }
}
`, testAccNamespace(), hclStringList(tags))
}

func testAccPolicyActionConfigRemovePackageTags(parentName string, tags []string) string {
	return testAccPolicyActionConfigParentBlock(parentName) + fmt.Sprintf(`
resource "cloudsmith_policy_action" "test" {
    workspace        = "%s"
    policy_slug_perm = cloudsmith_policy.parent.slug_perm

    remove_package_tags {
        tags = %s
    }
}
`, testAccNamespace(), hclStringList(tags))
}

func hclStringList(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
