// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

// Package cloudsmith implements a Terraform provider for interacting with Cloudsmith.
package provider

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	apikeyrule "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/apikeyrule"
	entitlement "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/entitlement"
	entitlementcontrol "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/entitlement/control"
	namespace "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/namespace"
	oidc "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/oidc"
	organization "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/organization"
	organizationmemberdetails "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/organization/memberdetails"
	organizationmembers "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/organization/members"
	cloudsmithpackage "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/pkg"
	policy "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/policy"
	policyaction "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/policy/action"
	licensepolicy "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/policy/license"
	packagedeny "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/policy/packagedeny"
	vulnerabilitypolicy "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/policy/vulnerability"
	repository "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository"
	repositoryconnected "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository/connected"
	repositorygeoip "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository/geoip"
	repositoryprivilege "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository/privilege"
	repositoryretention "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository/retention"
	repositoryupstream "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository/upstream"
	saml "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/saml"
	samlauth "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/saml/auth"
	service "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/service"
	servicedetails "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/service/details"
	team "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/team"
	teammanage "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/team/manage"
	teammembers "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/team/members"
	usagelimits "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/usagelimits"
	userself "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/userself"
	webhook "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/webhook"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/util"
)

// Provider returns a terraform.ResourceProvider.
func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:          schema.TypeString,
				Description:   "The API key for authenticating with the Cloudsmith API. Conflicts with oidc. When oidc is unset and CLOUDSMITH_USE_OIDC is not set, the provider reads CLOUDSMITH_API_KEY.",
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{"oidc"},
			},
			"oidc": {
				Type:          schema.TypeList,
				Description:   "OIDC login using an HCP Terraform workload identity token. The identity token comes from the workspace environment (TFC_WORKLOAD_IDENTITY_TOKEN_CLOUDSMITH or TFC_WORKLOAD_IDENTITY_TOKEN), not this block. Conflicts with api_key. Ignores leftover CLOUDSMITH_API_KEY. You can enable the same path with CLOUDSMITH_USE_OIDC=true and no block.",
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"api_key"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"organization": {
							Type:        schema.TypeString,
							Description: "Cloudsmith organization slug. Defaults to CLOUDSMITH_ORG when unset.",
							Optional:    true,
							DefaultFunc: schema.EnvDefaultFunc("CLOUDSMITH_ORG", nil),
						},
						"service_slug": {
							Type:        schema.TypeString,
							Description: "OIDC service account slug. Defaults to CLOUDSMITH_SERVICE_SLUG when unset.",
							Optional:    true,
							DefaultFunc: schema.EnvDefaultFunc("CLOUDSMITH_SERVICE_SLUG", nil),
						},
					},
				},
			},
			"api_host": {
				Type:        schema.TypeString,
				Description: "The API host to connect to (mostly useful for testing).",
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSMITH_API_HOST", "https://api.cloudsmith.io/v1"),
			},
			"headers": {
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Additional HTTP headers to include in API requests",
				Optional:    true,
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"cloudsmith_namespace":                 namespace.DataSourceNamespace(),
			"cloudsmith_oidc":                      oidc.DataSourceOidc(),
			"cloudsmith_organization":              organization.DataSourceOrganization(),
			"cloudsmith_package":                   cloudsmithpackage.DataSourcePackage(),
			"cloudsmith_package_list":              cloudsmithpackage.DataSourcePackageList(),
			"cloudsmith_repository":                repository.DataSourceRepository(),
			"cloudsmith_repository_connected_list": repositoryconnected.DataSourceRepositoryConnectedList(),
			"cloudsmith_repository_privileges":     repositoryprivilege.DataSourceRepositoryPrivileges(),
			"cloudsmith_package_deny_policy":       packagedeny.DataSourcePackageDenyPolicy(),
			"cloudsmith_policy":                    policy.DataSourcePolicy(),
			"cloudsmith_policy_list":               policy.DataSourcePolicyList(),
			"cloudsmith_entitlement_list":          entitlement.DataSourceEntitlementList(),
			"cloudsmith_list_org_members":          organizationmembers.DataSourceOrganizationMembersList(),
			"cloudsmith_org_member_details":        organizationmemberdetails.DataSourceMemberDetails(),
			"cloudsmith_user_self":                 userself.DataSourceUserSelf(),
			"cloudsmith_team_list":                 team.DataSourceTeamList(),
			"cloudsmith_team_members":              teammembers.DataSourceTeamMembers(),
			"cloudsmith_service_list":              service.DataSourceServiceList(),
			"cloudsmith_service_details":           servicedetails.DataSourceServiceDetails(),
			"cloudsmith_usage_limits":              usagelimits.DataSourceUsageLimits(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"cloudsmith_api_key_rule":              apikeyrule.ResourceAPIKeyRule(),
			"cloudsmith_entitlement":               entitlement.ResourceEntitlement(),
			"cloudsmith_license_policy":            licensepolicy.ResourceLicensePolicy(),
			"cloudsmith_repository":                repository.ResourceRepository(),
			"cloudsmith_repository_connected":      repositoryconnected.ResourceRepositoryConnected(),
			"cloudsmith_repository_geo_ip_rules":   repositorygeoip.ResourceRepositoryGeoIpRules(),
			"cloudsmith_repository_privileges":     repositoryprivilege.ResourceRepositoryPrivileges(),
			"cloudsmith_repository_upstream":       repositoryupstream.ResourceRepositoryUpstream(),
			"cloudsmith_service":                   service.ResourceService(),
			"cloudsmith_team":                      team.ResourceTeam(),
			"cloudsmith_vulnerability_policy":      vulnerabilitypolicy.ResourceVulnerabilityPolicy(),
			"cloudsmith_webhook":                   webhook.ResourceWebhook(),
			"cloudsmith_package_deny_policy":       packagedeny.PackageDenyPolicy(),
			"cloudsmith_oidc":                      oidc.ResourceOIDC(),
			"cloudsmith_policy":                    policy.ResourcePolicy(),
			"cloudsmith_policy_action":             policyaction.ResourcePolicyAction(),
			"cloudsmith_manage_team":               teammanage.ResourceManageTeam(),
			"cloudsmith_saml":                      saml.ResourceSAML(),
			"cloudsmith_saml_auth":                 samlauth.ResourceSAMLAuth(),
			"cloudsmith_repository_retention_rule": repositoryretention.ResourceRepoRetentionRule(),
			"cloudsmith_entitlement_control":       entitlementcontrol.ResourceEntitlementControl(),
			"cloudsmith_usage_limits":              usagelimits.ResourceUsageLimits(),
		},
	}

	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}

		apiHost := util.RequiredString(d, "api_host")
		userAgent := fmt.Sprintf("(%s %s) Terraform/%s", runtime.GOOS, runtime.GOARCH, terraformVersion)
		headers := d.Get("headers").(map[string]interface{})

		cred, err := parseCredential(authSpecFromResourceData(d), os.Getenv)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		tokens, err := tokenSourceFromCredential(cred, apiHost, headers, userAgent, os.Getenv, nil)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return newProviderConfig(ctx, apiHost, tokens, headers, userAgent)
	}

	return p
}
