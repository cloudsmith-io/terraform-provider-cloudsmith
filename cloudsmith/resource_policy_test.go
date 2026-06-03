package cloudsmith

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-go-v2/models/apierrors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPolicy_basic(t *testing.T) {
	t.Parallel()

	name := testAccUniquePolicyName("TF Acc Policy")
	updatedName := name + " (updated)"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPolicyCheckDestroy("cloudsmith_policy.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyConfigBasic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyCheckExists("cloudsmith_policy.test"),
					resource.TestCheckResourceAttrSet("cloudsmith_policy.test", "slug_perm"),
					resource.TestCheckResourceAttrSet("cloudsmith_policy.test", "created_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_policy.test", "updated_at"),
					resource.TestCheckResourceAttrSet("cloudsmith_policy.test", "version"),
					resource.TestCheckResourceAttr("cloudsmith_policy.test", "name", name),
					resource.TestCheckResourceAttr("cloudsmith_policy.test", "enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_policy.test", "is_terminal", "false"),
				),
			},
			{
				Config: testAccPolicyConfigBasicUpdate(updatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccPolicyCheckExists("cloudsmith_policy.test"),
					resource.TestCheckResourceAttr("cloudsmith_policy.test", "name", updatedName),
					resource.TestCheckResourceAttr("cloudsmith_policy.test", "enabled", "false"),
					resource.TestCheckResourceAttr("cloudsmith_policy.test", "is_terminal", "true"),
				),
			},
			{
				ResourceName: "cloudsmith_policy.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["cloudsmith_policy.test"]
					return fmt.Sprintf("%s.%s", rs.Primary.Attributes["workspace"], rs.Primary.Attributes["slug_perm"]), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPolicyCheckDestroy(resourceName string) resource.TestCheckFunc {
	return testAccRetry(10*time.Second, 500*time.Millisecond, func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		pc := testAccProvider.Meta().(*providerConfig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		workspace := resourceState.Primary.Attributes["workspace"]
		resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesRetrieve(
			ctx,
			resourceState.Primary.ID,
			workspace,
		)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verifying policy deletion: %w", err)
		}
		if resp != nil && resp.Policy != nil {
			return fmt.Errorf("policy still exists: %s/%s", workspace, resourceState.Primary.ID)
		}
		return nil
	})
}

func testAccPolicyCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		pc := testAccProvider.Meta().(*providerConfig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesRetrieve(
			ctx,
			resourceState.Primary.ID,
			testAccNamespace(),
		)
		return err
	}
}

func testAccPolicyConfigBasic(name string) string {
	return fmt.Sprintf(`
resource "cloudsmith_policy" "test" {
    workspace   = "%s"
    name        = "%s"
    description = "Created by terraform acceptance tests."
    rego        = <<-EOT
        package cloudsmith.policy
        default allow := true
    EOT
}
`, testAccNamespace(), name)
}

func testAccPolicyConfigBasicUpdate(name string) string {
	return fmt.Sprintf(`
resource "cloudsmith_policy" "test" {
    workspace   = "%s"
    name        = "%s"
    description = "Updated by terraform acceptance tests."
    enabled     = false
    is_terminal = true
    rego        = <<-EOT
        package cloudsmith.policy
        default allow := false
    EOT
}
`, testAccNamespace(), name)
}
