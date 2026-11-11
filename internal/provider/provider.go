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

	api_key_rule "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/api-key-rule"
	entitlement "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/entitlement"
	entitlement_control "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/entitlement-control"
	licence_policy "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/licence-policy"
	manage_team "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/manage-team"
	namespace "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/namespace"
	oidc "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/oidc"
	organization "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/organization"
	organization_member_details "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/organization-member-details"
	organization_members "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/organization-members"
	cloudsmith_package "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/package"
	package_deny_policy "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/package-deny-policy"
	policy "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/policy"
	policy_action "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/policy-action"
	repository "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository"
	repository_connected "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository-connected"
	repository_geo_ip_rules "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository-geo-ip-rules"
	repository_privilege "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository-privilege"
	repository_retention_rule "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository-retention-rule"
	repository_upstream "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/repository-upstream"
	saml "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/saml"
	saml_auth "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/saml-auth"
	service "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/service"
	service_details "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/service-details"
	team "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/team"
	team_members "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/team-members"
	usage_limits "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/usage-limits"
	user_self "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/user-self"
	vulnerability_policy "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/vulnerability-policy"
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
			"cloudsmith_package":                   cloudsmith_package.DataSourcePackage(),
			"cloudsmith_package_list":              cloudsmith_package.DataSourcePackageList(),
			"cloudsmith_repository":                repository.DataSourceRepository(),
			"cloudsmith_repository_connected_list": repository_connected.DataSourceRepositoryConnectedList(),
			"cloudsmith_repository_privileges":     repository_privilege.DataSourceRepositoryPrivileges(),
			"cloudsmith_package_deny_policy":       package_deny_policy.DataSourcePackageDenyPolicy(),
			"cloudsmith_policy":                    policy.DataSourcePolicy(),
			"cloudsmith_policy_list":               policy.DataSourcePolicyList(),
			"cloudsmith_entitlement_list":          entitlement.DataSourceEntitlementList(),
			"cloudsmith_list_org_members":          organization_members.DataSourceOrganizationMembersList(),
			"cloudsmith_org_member_details":        organization_member_details.DataSourceMemberDetails(),
			"cloudsmith_user_self":                 user_self.DataSourceUserSelf(),
			"cloudsmith_team_list":                 team.DataSourceTeamList(),
			"cloudsmith_team_members":              team_members.DataSourceTeamMembers(),
			"cloudsmith_service_list":              service.DataSourceServiceList(),
			"cloudsmith_service_details":           service_details.DataSourceServiceDetails(),
			"cloudsmith_usage_limits":              usage_limits.DataSourceUsageLimits(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"cloudsmith_api_key_rule":              api_key_rule.ResourceAPIKeyRule(),
			"cloudsmith_entitlement":               entitlement.ResourceEntitlement(),
			"cloudsmith_license_policy":            licence_policy.ResourceLicensePolicy(),
			"cloudsmith_repository":                repository.ResourceRepository(),
			"cloudsmith_repository_connected":      repository_connected.ResourceRepositoryConnected(),
			"cloudsmith_repository_geo_ip_rules":   repository_geo_ip_rules.ResourceRepositoryGeoIpRules(),
			"cloudsmith_repository_privileges":     repository_privilege.ResourceRepositoryPrivileges(),
			"cloudsmith_repository_upstream":       repository_upstream.ResourceRepositoryUpstream(),
			"cloudsmith_service":                   service.ResourceService(),
			"cloudsmith_team":                      team.ResourceTeam(),
			"cloudsmith_vulnerability_policy":      vulnerability_policy.ResourceVulnerabilityPolicy(),
			"cloudsmith_webhook":                   webhook.ResourceWebhook(),
			"cloudsmith_package_deny_policy":       package_deny_policy.PackageDenyPolicy(),
			"cloudsmith_oidc":                      oidc.ResourceOIDC(),
			"cloudsmith_policy":                    policy.ResourcePolicy(),
			"cloudsmith_policy_action":             policy_action.ResourcePolicyAction(),
			"cloudsmith_manage_team":               manage_team.ResourceManageTeam(),
			"cloudsmith_saml":                      saml.ResourceSAML(),
			"cloudsmith_saml_auth":                 saml_auth.ResourceSAMLAuth(),
			"cloudsmith_repository_retention_rule": repository_retention_rule.ResourceRepoRetentionRule(),
			"cloudsmith_entitlement_control":       entitlement_control.ResourceEntitlementControl(),
			"cloudsmith_usage_limits":              usage_limits.ResourceUsageLimits(),
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
