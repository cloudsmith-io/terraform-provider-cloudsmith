// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package acctest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudsmith-io/cloudsmith-go-v2/models/apierrors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/util"
)

var (
	testAccRepositoryNameSequence atomic.Uint64
	testAccPolicyNameSequence     atomic.Uint64
)

const testAccRepositoryNameMaxLength = 50

func TestAccUniquePolicyName(base string) string {
	return fmt.Sprintf("%s %d-%d", base, time.Now().UnixMilli(), testAccPolicyNameSequence.Add(1))
}

func TestAccRetry(timeout, interval time.Duration, check resource.TestCheckFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		deadline := time.Now().Add(timeout)
		var lastErr error
		for {
			lastErr = check(s)
			if lastErr == nil {
				return nil
			}
			if time.Now().After(deadline) {
				return lastErr
			}
			time.Sleep(interval)
		}
	}
}

// TestAccUniqueRepositoryName keeps repository names unique across acceptance
// test runs while respecting Cloudsmith's 50 character repository name limit.
func TestAccUniqueRepositoryName(base string) string {
	suffix := fmt.Sprintf("-%d-%d", time.Now().UnixMilli(), testAccRepositoryNameSequence.Add(1))
	maxBaseLength := testAccRepositoryNameMaxLength - len(suffix)

	if maxBaseLength < 1 {
		maxBaseLength = 1
	}

	base = strings.Trim(base, "-")
	if len(base) > maxBaseLength {
		base = strings.TrimRight(base[:maxBaseLength], "-")
	}

	if base == "" {
		base = "tfacc"
		if len(base) > maxBaseLength {
			base = base[:maxBaseLength]
		}
	}

	return base + suffix
}

func TestAccNamespace() string {
	return os.Getenv("CLOUDSMITH_NAMESPACE")
}

//nolint:err113
func TestAccServiceCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := TestAccProvider.Meta().(*util.ProviderConfig)

		req := pc.APIClient.OrgsApi.OrgsServicesRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.OrgsApi.OrgsServicesReadExecute(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	}
}

func TestAccPolicyCheckDestroy(resourceName string) resource.TestCheckFunc {
	return TestAccRetry(10*time.Second, 500*time.Millisecond, func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		pc := TestAccProvider.Meta().(*util.ProviderConfig)
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

//nolint:err113
func TestAccRepositoryCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := TestAccProvider.Meta().(*util.ProviderConfig)

		req := pc.APIClient.ReposApi.ReposRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req)
		if err != nil && !util.Is404(resp) {
			return fmt.Errorf("unable to verify repository deletion: %w", err)
		} else if util.Is200(resp) {
			return fmt.Errorf("unable to verify repository deletion: still exists: %s/%s", os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:err113
func TestAccRepositoryCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := TestAccProvider.Meta().(*util.ProviderConfig)

		req := pc.APIClient.ReposApi.ReposRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:err113
func TestAccTeamCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := TestAccProvider.Meta().(*util.ProviderConfig)

		req := pc.APIClient.OrgsApi.OrgsTeamsRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.OrgsApi.OrgsTeamsReadExecute(req)
		if err != nil && !util.Is404(resp) {
			return fmt.Errorf("unable to verify team deletion: %w", err)
		} else if util.Is200(resp) {
			return fmt.Errorf("unable to verify team deletion: still exists: %s/%s", os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		}
		defer resp.Body.Close()

		return nil
	}
}

//nolint:err113
func TestAccTeamCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := TestAccProvider.Meta().(*util.ProviderConfig)

		req := pc.APIClient.OrgsApi.OrgsTeamsRead(pc.Auth, os.Getenv("CLOUDSMITH_NAMESPACE"), resourceState.Primary.ID)
		_, resp, err := pc.APIClient.OrgsApi.OrgsTeamsReadExecute(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return nil
	}
}
