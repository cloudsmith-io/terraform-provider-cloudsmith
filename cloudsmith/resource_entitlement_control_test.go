// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:testpackage
package cloudsmith

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccEntitlementControl_basic spins up a repository and uses its default entitlement token,
// creates an entitlement control with the token enabled, verifies it exists and checks
// the enabled state is set correctly. Then it changes the enabled state to false,
// and verifies it's been set correctly before tearing down the resources and
// verifying deletion.
func TestAccEntitlementControl_basic(t *testing.T) {
	t.Parallel()

	repositoryName := testAccUniqueRepositoryName("terraform-acc-test-ent-ctrl")

	tc := testAccEntitlementControlTestCase(t.Context(), repositoryName, testAccProvider)
	tc.PreCheck = func() { testAccPreCheck(t) }
	resource.Test(t, tc)
}

func testAccEntitlementControlTestCase(ctx context.Context, repositoryName string, provider *schema.Provider) resource.TestCase {
	var namespace, repository string
	var controlCreated bool
	return resource.TestCase{
		Providers: map[string]*schema.Provider{"cloudsmith": provider},
		CheckDestroy: func(s *terraform.State) error {
			if s == nil || s.RootModule() == nil {
				return fmt.Errorf("entitlement fixture has no root state")
			}
			if _, ok := s.RootModule().Resources["cloudsmith_entitlement_control.test"]; !ok && !controlCreated && repository != "" {
				// A failed repository-only readiness check has no control to destroy.
				// Verify that Terraform still cleaned up the exact owned repository.
				pc := provider.Meta().(*providerConfig)
				cleanupCtx, cancel := context.WithTimeout(pc.Auth, time.Minute)
				defer cancel()
				_, resp, err := pc.APIClient.ReposApi.ReposRead(cleanupCtx, namespace, repository).Execute()
				if resp != nil {
					defer resp.Body.Close()
				}
				if is404(resp) {
					return nil
				}
				if err != nil {
					return fmt.Errorf("unable to verify entitlement fixture cleanup: %w", err)
				}
				return fmt.Errorf("entitlement fixture repository still exists: %s/%s", namespace, repository)
			}
			return testAccEntitlementControlCheckDestroy(provider, "cloudsmith_entitlement_control.test")(s)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccEntitlementControlConfigRepository(repositoryName),
				Check: func(s *terraform.State) error {
					if s == nil || s.RootModule() == nil {
						return fmt.Errorf("entitlement fixture has no root state")
					}
					rs, ok := s.RootModule().Resources["cloudsmith_repository.test"]
					if !ok || rs == nil || rs.Primary == nil || rs.Primary.ID == "" {
						return fmt.Errorf("entitlement fixture repository not created")
					}
					attrs := rs.Primary.Attributes
					if attrs["namespace"] == "" || attrs["slug_perm"] != rs.Primary.ID || attrs["name"] != repositoryName {
						return fmt.Errorf("entitlement fixture repository identity is incomplete")
					}
					namespace, repository = attrs["namespace"], attrs["slug_perm"]
					waitCtx, cancel := context.WithTimeout(ctx, time.Minute)
					defer cancel()
					return testAccWaitForDefaultEntitlement(waitCtx, provider.Meta().(*providerConfig), namespace, repository, time.Second)
				},
			},
			{
				Config: testAccEntitlementControlConfigBasic(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						controlCreated = true
						return testAccEntitlementControlCheckExists(provider, "cloudsmith_entitlement_control.test")(s)
					},
					func(s *terraform.State) error {
						resourceState, ok := s.RootModule().Resources["cloudsmith_entitlement_control.test"]
						if !ok {
							return fmt.Errorf("resource not found: %s", "cloudsmith_entitlement_control.test")
						}
						if resourceState.Primary.ID == "" {
							return fmt.Errorf("resource id not set")
						}
						pc := provider.Meta().(*providerConfig)
						namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
						repository := resourceState.Primary.Attributes["repository"]
						identifier := resourceState.Primary.ID
						return waitForEntitlementControlEnabled(pc, namespace, repository, identifier, true, 15)
					},
					resource.TestCheckResourceAttr("cloudsmith_entitlement_control.test", "namespace", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_entitlement_control.test", "enabled", "true"),
				),
			},
			{
				Config: testAccEntitlementControlConfigBasicUpdate(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					testAccEntitlementControlCheckExists(provider, "cloudsmith_entitlement_control.test"),
					func(s *terraform.State) error {
						resourceState, ok := s.RootModule().Resources["cloudsmith_entitlement_control.test"]
						if !ok {
							return fmt.Errorf("resource not found: %s", "cloudsmith_entitlement_control.test")
						}
						if resourceState.Primary.ID == "" {
							return fmt.Errorf("resource id not set")
						}
						pc := provider.Meta().(*providerConfig)
						namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
						repository := resourceState.Primary.Attributes["repository"]
						identifier := resourceState.Primary.ID
						return waitForEntitlementControlEnabled(pc, namespace, repository, identifier, false, 15)
					},
					resource.TestCheckResourceAttr("cloudsmith_entitlement_control.test", "namespace", os.Getenv("CLOUDSMITH_NAMESPACE")),
					resource.TestCheckResourceAttr("cloudsmith_entitlement_control.test", "enabled", "false"),
				),
			},
			{
				ResourceName:      "cloudsmith_entitlement_control.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources["cloudsmith_entitlement_control.test"]
					return fmt.Sprintf(
						"%s.%s.%s",
						resourceState.Primary.Attributes["namespace"],
						resourceState.Primary.Attributes["repository"],
						resourceState.Primary.ID,
					), nil
				},
				ImportStateVerifyIgnore: []string{
					"identifier", // Ignore identifier as it's used for creation but not returned in read
				},
			},
		},
	}
}

//nolint:err113
func testAccEntitlementControlCheckDestroy(provider *schema.Provider, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok || resourceState == nil || resourceState.Primary == nil {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := provider.Meta().(*providerConfig)

		namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
		repository := resourceState.Primary.Attributes["repository"]
		identifier := resourceState.Primary.ID

		req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, identifier)
		entitlement, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify entitlement control state: %w", err)
		} else if is200(resp) && entitlement.GetIsActive() {
			return fmt.Errorf("unable to verify entitlement control state: still enabled: %s/%s/%s", namespace, repository, identifier)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:err113
func testAccEntitlementControlCheckExists(provider *schema.Provider, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := provider.Meta().(*providerConfig)

		namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
		repository := resourceState.Primary.Attributes["repository"]
		identifier := resourceState.Primary.ID

		req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, identifier)
		_, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req)
		if err != nil {
			return fmt.Errorf("unable to verify entitlement control existence: %w", err)
		}
		defer resp.Body.Close()

		return nil
	}
}

func waitForEntitlementControlEnabled(pc *providerConfig, namespace, repository, identifier string, wantEnabled bool, timeoutSec int) error {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for {
		req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, identifier)
		entitlement, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err == nil {
			if entitlement.GetIsActive() == wantEnabled {
				return nil
			}
		} else if is404(resp) {
			return fmt.Errorf("entitlement not found while waiting for enabled=%v", wantEnabled)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for entitlement control enabled=%v", wantEnabled)
		}
		time.Sleep(1 * time.Second)
	}
}

// Only the newly created acceptance fixture is expected to acquire a Default
// token. Repository readiness does not establish entitlement-list readiness;
// keep this bounded precondition out of the production data source.
func testAccWaitForDefaultEntitlement(ctx context.Context, pc *providerConfig, namespace, repository string, interval time.Duration) error {
	// Keep SDK authentication while using the fixture's cancellation/deadline.
	ctx = context.WithValue(ctx, cloudsmith.ContextAPIKeys, pc.Auth.Value(cloudsmith.ContextAPIKeys))
	query := buildQueryString(schema.NewSet(schema.HashString, []interface{}{"name:Default"}))
	last := "no entitlement response"
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("waiting for Default entitlement in %s/%s (%s): %w", namespace, repository, last, err)
		}
		tokens, resp, err := pc.APIClient.EntitlementsApi.EntitlementsList(ctx, namespace, repository).
			Page(1).PageSize(DefaultPageSize).ShowTokens(false).Active(false).Query(query).Execute()
		if resp != nil {
			resp.Body.Close()
		}
		switch {
		case err == nil:
			if len(tokens) == 1 && tokens[0].GetDefault() && tokens[0].GetName() == "Default" && tokens[0].GetSlugPerm() != "" {
				return nil
			}
			if len(tokens) != 0 {
				return fmt.Errorf("unexpected Default entitlement result in %s/%s: expected one default token with a slug_perm", namespace, repository)
			}
			last = "empty entitlement list"
		case is404(resp):
			last = "entitlement list returned 404"
		default:
			return fmt.Errorf("reading Default entitlement in %s/%s: %w", namespace, repository, err)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("waiting for Default entitlement in %s/%s (%s): %w", namespace, repository, last, ctx.Err())
		case <-timer.C:
		}
	}
}

func testAccEntitlementControlConfigRepository(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccEntitlementControlConfigBasic(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}

data "cloudsmith_entitlement_list" "test" {
    namespace  = resource.cloudsmith_repository.test.namespace
    repository = resource.cloudsmith_repository.test.slug_perm
    query      = ["name:Default"]
}

resource "cloudsmith_entitlement_control" "test" {
    namespace  = resource.cloudsmith_repository.test.namespace
    repository = resource.cloudsmith_repository.test.slug_perm
    identifier = data.cloudsmith_entitlement_list.test.entitlement_tokens[0].slug_perm
    enabled    = true
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func testAccEntitlementControlConfigBasicUpdate(repositoryName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}

data "cloudsmith_entitlement_list" "test" {
    namespace  = resource.cloudsmith_repository.test.namespace
    repository = resource.cloudsmith_repository.test.slug_perm
    query      = ["name:Default"]
}

resource "cloudsmith_entitlement_control" "test" {
    namespace  = resource.cloudsmith_repository.test.namespace
    repository = resource.cloudsmith_repository.test.slug_perm
    identifier = data.cloudsmith_entitlement_list.test.entitlement_tokens[0].slug_perm
    enabled    = false
}
`, repositoryName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}
