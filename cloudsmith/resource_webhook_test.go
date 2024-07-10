//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccWebhook_basic spins up a repository with all default options,
// creates a webhook with default options and verifies it exists and checks
// the name is set correctly. Then it changes the name and some of the
// options and verifies they've been set correctly before tearing down the
// resources and verifying deletion.
func TestAccWebhook_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccWebhookCheckDestroy("cloudsmith_webhook.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("cloudsmith_webhook.test"),
					resource.TestCheckResourceAttr("cloudsmith_webhook.test", "request_body_format", "JSON Object"),
				),
			},
			{
				Config: testAccWebhookConfigBasicUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("cloudsmith_webhook.test"),
				),
			},
			{
				Config: testAccWebhookConfigBasicUpdateWithTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("cloudsmith_webhook.test"),
					resource.TestCheckResourceAttr("cloudsmith_webhook.test", "template.0.event", "package.created"),
					resource.TestCheckResourceAttr("cloudsmith_webhook.test", "template.1.template", "flop"),
				),
			},
			{
				Config: testAccWebhookConfigBasicUpdateWithTemplateChange,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("cloudsmith_webhook.test"),
					resource.TestCheckResourceAttr("cloudsmith_webhook.test", "template.1.template", "flap"),
				),
			},
			{
				ResourceName: "cloudsmith_webhook.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources["cloudsmith_webhook.test"]
					return fmt.Sprintf(
						"%s.%s.%s",
						resourceState.Primary.Attributes["namespace"],
						resourceState.Primary.Attributes["repository"],
						resourceState.Primary.ID,
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

//nolint:goerr113
func testAccWebhookCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
		repository := resourceState.Primary.Attributes["repository"]
		webhook := resourceState.Primary.ID

		req := pc.APIClient.WebhooksApi.WebhooksRead(pc.Auth, namespace, repository, webhook)
		_, resp, err := pc.APIClient.WebhooksApi.WebhooksReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify webhook deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify webhook deletion: still exists: %s/%s/%s", namespace, repository, webhook)
		}
		defer resp.Body.Close()

		rreq := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, repository)
		_, resp, err = pc.APIClient.ReposApi.ReposReadExecute(rreq)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify repository deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify repository deletion: still exists: %s/%s", namespace, repository)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:goerr113
func testAccWebhookCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
		repository := resourceState.Primary.Attributes["repository"]
		webhook := resourceState.Primary.ID

		req := pc.APIClient.WebhooksApi.WebhooksRead(pc.Auth, namespace, repository, webhook)
		_, resp, err := pc.APIClient.WebhooksApi.WebhooksReadExecute(req)
		if err != nil {
			return fmt.Errorf("unable to verify webhook existence: %w", err)
		}
		defer resp.Body.Close()

		return nil
	}
}

var testAccWebhookConfigBasic = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-webhook"
	namespace = "%s"
}

resource "cloudsmith_webhook" "test" {
    namespace  = "${cloudsmith_repository.test.namespace}"
    repository = "${cloudsmith_repository.test.slug_perm}"

	events     = ["package.created", "package.deleted", "package.failed", "package.security_scanned", "package.synced", "package.syncing", "package.tags_updated"]
	target_url = "https://example.com"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccWebhookConfigBasicUpdate = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-webhook"
	namespace = "%s"
}

resource "cloudsmith_webhook" "test" {
    namespace  = "${cloudsmith_repository.test.namespace}"
    repository = "${cloudsmith_repository.test.slug_perm}"

	events     = ["package.created", "package.deleted"]
	target_url = "https://example.com"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccWebhookConfigBasicUpdateWithTemplate = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-webhook"
	namespace = "%s"
}

resource "cloudsmith_webhook" "test" {
    namespace  = "${cloudsmith_repository.test.namespace}"
    repository = "${cloudsmith_repository.test.slug_perm}"

	events              = ["package.created", "package.deleted"]
	request_body_format = "Handlebars Template"
	target_url          = "https://example.com"

	template {
		event = "package.created"
		template = "flip"
	}

	template {
		event = "package.deleted"
		template = "flop"
	}
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccWebhookConfigBasicUpdateWithTemplateChange = fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "terraform-acc-test-webhook"
	namespace = "%s"
}

resource "cloudsmith_webhook" "test" {
    namespace  = "${cloudsmith_repository.test.namespace}"
    repository = "${cloudsmith_repository.test.slug_perm}"

	events              = ["package.created", "package.deleted"]
	request_body_format = "Handlebars Template"
	target_url          = "https://example.com"

	template {
		event = "package.created"
		template = "flip"
	}

	template {
		event = "package.deleted"
		template = "flap"
	}
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))
