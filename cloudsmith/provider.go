// Package cloudsmith implements a Terraform provider for interacting with Cloudsmith.
package cloudsmith

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
			"cloudsmith_namespace":                 dataSourceNamespace(),
			"cloudsmith_oidc":                      dataSourceOidc(),
			"cloudsmith_organization":              dataSourceOrganization(),
			"cloudsmith_package":                   dataSourcePackage(),
			"cloudsmith_package_list":              dataSourcePackageList(),
			"cloudsmith_repository":                dataSourceRepository(),
			"cloudsmith_repository_connected_list": dataSourceRepositoryConnectedList(),
			"cloudsmith_repository_privileges":     dataSourceRepositoryPrivileges(),
			"cloudsmith_package_deny_policy":       dataSourcePackageDenyPolicy(),
			"cloudsmith_policy":                    dataSourcePolicy(),
			"cloudsmith_policy_list":               dataSourcePolicyList(),
			"cloudsmith_entitlement_list":          dataSourceEntitlementList(),
			"cloudsmith_list_org_members":          dataSourceOrganizationMembersList(),
			"cloudsmith_org_member_details":        dataSourceMemberDetails(),
			"cloudsmith_user_self":                 dataSourceUserSelf(),
			"cloudsmith_team_list":                 dataSourceTeamList(),
			"cloudsmith_team_members":              dataSourceTeamMembers(),
			"cloudsmith_service_list":              dataSourceServiceList(),
			"cloudsmith_service_details":           dataSourceServiceDetails(),
			"cloudsmith_usage_limits":              dataSourceUsageLimits(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"cloudsmith_entitlement":               resourceEntitlement(),
			"cloudsmith_license_policy":            resourceLicensePolicy(),
			"cloudsmith_repository":                resourceRepository(),
			"cloudsmith_repository_connected":      resourceRepositoryConnected(),
			"cloudsmith_repository_geo_ip_rules":   resourceRepositoryGeoIpRules(),
			"cloudsmith_repository_privileges":     resourceRepositoryPrivileges(),
			"cloudsmith_repository_upstream":       resourceRepositoryUpstream(),
			"cloudsmith_service":                   resourceService(),
			"cloudsmith_team":                      resourceTeam(),
			"cloudsmith_vulnerability_policy":      resourceVulnerabilityPolicy(),
			"cloudsmith_webhook":                   resourceWebhook(),
			"cloudsmith_package_deny_policy":       packageDenyPolicy(),
			"cloudsmith_oidc":                      resourceOIDC(),
			"cloudsmith_policy":                    resourcePolicy(),
			"cloudsmith_policy_action":             resourcePolicyAction(),
			"cloudsmith_manage_team":               resourceManageTeam(),
			"cloudsmith_saml":                      resourceSAML(),
			"cloudsmith_saml_auth":                 resourceSAMLAuth(),
			"cloudsmith_repository_retention_rule": resourceRepoRetentionRule(),
			"cloudsmith_entitlement_control":       resourceEntitlementControl(),
			"cloudsmith_usage_limits":              resourceUsageLimits(),
		},
	}

	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}

		apiHost := requiredString(d, "api_host")
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
