// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:testpackage
package repository_test

import (
	"fmt"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/acctest"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/util"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccRepositoryDelete_afterPrivilegesRevoked reproduces the ordering from
// issue #214: a repository with default_privilege = "None" (so the acceptance
// test identity has no access to it except what's granted explicitly), a
// team, and a cloudsmith_repository_privileges resource granting the
// acceptance test identity (and the team) Admin on the repository.
//
// Terraform's dependency graph destroys cloudsmith_repository_privileges
// before cloudsmith_repository (since privileges depends on the repository),
// creating the lockout risk before the repository's own Delete is called. The
// privileges Delete must preserve the authenticated account's Admin grant
// while removing the managed team grant, so the subsequent repository DELETE
// remains authorized and actually removes the repository.
//
// The acceptance-test credential may authenticate as either a service account
// or an organization member depending on the environment (e.g. a service
// account locally vs. a user in CI), so the account kind is resolved via the
// API before building the config, mirroring authenticatedAccountAdminPrivilege
// in resource_repository_privileges.go.
func TestAccRepositoryDelete_afterPrivilegesRevoked(t *testing.T) {
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
	}
	acctest.TestAccPreCheck(t)
	t.Parallel()

	repositoryName := acctest.TestAccUniqueRepositoryName("terraform-acc-test-del")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.TestAccPreCheck(t) },
		Providers:    acctest.TestAccProviders,
		CheckDestroy: acctest.TestAccRepositoryCheckDestroy("cloudsmith_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryDeleteConfigPrivilegesRevoked(t, repositoryName),
				Check: resource.ComposeTestCheckFunc(
					acctest.TestAccRepositoryCheckExists("cloudsmith_repository.test"),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "default_privilege", "None"),
				),
			},
		},
		// No further steps: resource.Test destroys everything created above at
		// the end of the test, which exercises the exact ordering (privileges
		// destroyed before the repository) that triggered the bug. If destroy
		// returns an error, the test fails here.
	})
}

// testAccAuthenticatedAccountPrivilegeBlock resolves whether the acceptance
// test credential is a service account or an organization member, and
// returns the matching cloudsmith_repository_privileges HCL block ("service"
// or "user") granting it Admin.
func testAccAuthenticatedAccountPrivilegeBlock(t *testing.T, pc *util.ProviderConfig, organization, slug string) string {
	serviceReq := pc.APIClient.OrgsApi.OrgsServicesRead(pc.Auth, organization, slug)
	_, serviceResp, serviceErr := pc.APIClient.OrgsApi.OrgsServicesReadExecute(serviceReq)
	if serviceErr == nil {
		return fmt.Sprintf(`
	service {
		privilege = "Admin"
		slug      = %q
	}
`, slug)
	}
	if !util.Is404(serviceResp) {
		t.Fatalf("error determining whether acceptance test account is a service: %v", serviceErr)
	}

	memberReq := pc.APIClient.OrgsApi.OrgsMembersRead(pc.Auth, organization, slug)
	_, memberResp, memberErr := pc.APIClient.OrgsApi.OrgsMembersReadExecute(memberReq)
	if memberErr == nil {
		return fmt.Sprintf(`
	user {
		privilege = "Admin"
		slug      = %q
	}
`, slug)
	}
	if util.Is404(memberResp) {
		t.Fatalf("acceptance test account slug %q is neither a service nor an organization member", slug)
	}
	t.Fatalf("error determining whether acceptance test account is an organization member: %v", memberErr)

	return ""
}

func testAccRepositoryDeleteConfigPrivilegesRevoked(t *testing.T, repositoryName string) string {
	pc := acctest.TestAccAPIClient(t)
	organization := os.Getenv("CLOUDSMITH_NAMESPACE")

	userReq := pc.APIClient.UserApi.UserSelf(pc.Auth)
	userSelf, _, err := pc.APIClient.UserApi.UserSelfExecute(userReq)
	if err != nil {
		t.Fatalf("error retrieving authenticated account for acceptance test: %v", err)
	}

	privilegeBlock := testAccAuthenticatedAccountPrivilegeBlock(t, pc, organization, userSelf.GetSlug())

	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name               = "%s"
	namespace          = "%s"
	default_privilege  = "None"
}

resource "cloudsmith_team" "test" {
	name         = "%s-team"
	organization = cloudsmith_repository.test.namespace
}

resource "cloudsmith_repository_privileges" "test" {
	organization = cloudsmith_repository.test.namespace
	repository   = cloudsmith_repository.test.slug

	# Grants the acceptance-test identity (standing in for the "provisioning
	# SA" from the issue) Admin on the repository. Since default_privilege is
	# "None" above, this is the *only* source of access this identity has to
	# the repository, so destroying this resource before the repository
	# reproduces "account lost visibility after privileges revoked" from #214.
%s
	team {
		privilege = "Admin"
		slug      = cloudsmith_team.test.slug
	}
}
`, repositoryName, organization, repositoryName, privilegeBlock)
}
