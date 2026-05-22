//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const ConnectedResourceName string = "cloudsmith_repository_connected.test"

// TestAccRepositoryConnected_basic creates a source repository and a target
// repository, connects them, verifies the connection exists, updates the
// priority/is_active fields, imports the resource and finally verifies the
// connection is cleaned up.
func TestAccRepositoryConnected_basic(t *testing.T) {
	t.Parallel()

	sourceName := testAccUniqueRepositoryName("tf-acc-connected-src")
	targetName := testAccUniqueRepositoryName("tf-acc-connected-tgt")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryConnectedCheckDestroy(ConnectedResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryConnectedConfigCreate(sourceName, targetName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryConnectedCheckExists(ConnectedResourceName),
					resource.TestCheckResourceAttr(ConnectedResourceName, "is_active", "true"),
					resource.TestCheckResourceAttr(ConnectedResourceName, "priority", "1"),
					resource.TestCheckResourceAttrSet(ConnectedResourceName, "slug_perm"),
					resource.TestCheckResourceAttrSet(ConnectedResourceName, "created_at"),
				),
			},
			{
				Config: testAccRepositoryConnectedConfigUpdate(sourceName, targetName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryConnectedCheckExists(ConnectedResourceName),
					resource.TestCheckResourceAttr(ConnectedResourceName, "is_active", "false"),
					resource.TestCheckResourceAttr(ConnectedResourceName, "priority", "5"),
				),
			},
			{
				ResourceName: ConnectedResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[ConnectedResourceName]
					return fmt.Sprintf(
						"%s.%s.%s",
						resourceState.Primary.Attributes["namespace"],
						resourceState.Primary.Attributes["repository"],
						resourceState.Primary.Attributes["slug_perm"],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRepositoryConnectedConfigCreate(sourceName, targetName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "source" {
    name      = "%s"
    namespace = "%s"
}

resource "cloudsmith_repository" "target" {
    name      = "%s"
    namespace = "%s"
}

resource "cloudsmith_repository_connected" "test" {
    namespace         = resource.cloudsmith_repository.source.namespace
    repository        = resource.cloudsmith_repository.source.slug_perm
    target_repository = resource.cloudsmith_repository.target.slug
    is_active         = true
    priority          = 1
}
`, sourceName, testAccNamespace(), targetName, testAccNamespace())
}

func testAccRepositoryConnectedConfigUpdate(sourceName, targetName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "source" {
    name      = "%s"
    namespace = "%s"
}

resource "cloudsmith_repository" "target" {
    name      = "%s"
    namespace = "%s"
}

resource "cloudsmith_repository_connected" "test" {
    namespace         = resource.cloudsmith_repository.source.namespace
    repository        = resource.cloudsmith_repository.source.slug_perm
    target_repository = resource.cloudsmith_repository.target.slug
    is_active         = false
    priority          = 5
}
`, sourceName, testAccNamespace(), targetName, testAccNamespace())
}

//nolint:err113
func testAccRepositoryConnectedCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		namespace := resourceState.Primary.Attributes["namespace"]
		repository := resourceState.Primary.Attributes["repository"]
		slugPerm := resourceState.Primary.Attributes["slug_perm"]

		req := pc.APIClient.ReposApi.ReposConnectedRead(pc.Auth, namespace, repository, slugPerm)
		_, resp, err := pc.APIClient.ReposApi.ReposConnectedReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify connected repository deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify connected repository deletion: still exists: %s/%s/%s", namespace, repository, slugPerm)
		}
		if resp != nil {
			defer resp.Body.Close()
		}

		return nil
	}
}

//nolint:err113
func testAccRepositoryConnectedCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		namespace := resourceState.Primary.Attributes["namespace"]
		repository := resourceState.Primary.Attributes["repository"]
		slugPerm := resourceState.Primary.ID

		req := pc.APIClient.ReposApi.ReposConnectedRead(pc.Auth, namespace, repository, slugPerm)
		_, resp, err := pc.APIClient.ReposApi.ReposConnectedReadExecute(req)
		if err != nil {
			return fmt.Errorf("unable to verify connected repository existence: %w", err)
		}
		defer resp.Body.Close()

		return nil
	}
}
