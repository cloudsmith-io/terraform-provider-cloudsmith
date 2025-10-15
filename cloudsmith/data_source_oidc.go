package cloudsmith

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOidcRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	slugPerm := requiredString(d, "slug_perm")

	req := pc.APIClient.OrgsApi.OrgsOpenidConnectRead(pc.Auth, namespace, slugPerm)
	oidc, _, err := pc.APIClient.OrgsApi.OrgsOpenidConnectReadExecute(req)
	if err != nil {
		return err
	}

	d.Set("claims", oidc.GetClaims())
	d.Set("enabled", oidc.GetEnabled())
	d.Set("name", oidc.GetName())
	d.Set("namespace", namespace)
	d.Set("provider_url", oidc.GetProviderUrl())
	d.Set("service_accounts", oidc.GetServiceAccounts())
	d.Set("slug", oidc.GetSlug())
	d.Set("slug_perm", oidc.GetSlugPerm())

	// Handle mapping_claim properly - set null if empty
	var mappingClaim interface{} = nil
	if getter, ok := any(oidc).(interface{ GetMappingClaimOk() (*string, bool) }); ok {
		if mc, ok2 := getter.GetMappingClaimOk(); ok2 && mc != nil && *mc != "" {
			mappingClaim = *mc
		}
	}
	d.Set("mapping_claim", mappingClaim)

	// Retrieve dynamic mappings if available
	apiMappings, err := retrieveAllDynamicMappings(pc, namespace, slugPerm)
	if err != nil {
		return fmt.Errorf("error retrieving dynamic mappings: %w", err)
	}
	d.Set("dynamic_mappings", convertMappingsToSetElems(apiMappings))

	d.SetId(fmt.Sprintf("%s.%s", namespace, slugPerm))

	return nil
}

func dataSourceOidc() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceOidcRead,

		Schema: map[string]*schema.Schema{
			"claims": {
				Type:        schema.TypeMap,
				Description: "The claims associated with these provider settings.",
				Computed:    true,
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "Whether the provider settings should be used for incoming OIDC requests.",
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the provider settings are being configured for.",
				Computed:    true,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "Namespace (or organization) to which this OIDC config belongs.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"provider_url": {
				Type:        schema.TypeString,
				Description: "The URL from the provider that serves as the base for the OpenID configuration.",
				Computed:    true,
			},
			"service_accounts": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The service accounts associated with these provider settings (static providers only).",
				Computed:    true,
			},
			"mapping_claim": {
				Type:        schema.TypeString,
				Description: "The claim key whose values dynamically map to service accounts (dynamic providers only).",
				Computed:    true,
			},
			"dynamic_mappings": {
				Type:        schema.TypeSet,
				Description: "Set of dynamic claim value -> service account mappings.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"claim_value": {
							Type:        schema.TypeString,
							Description: "The value of the mapping claim.",
							Computed:    true,
						},
						"service_account": {
							Type:        schema.TypeString,
							Description: "Service account slug mapped to the claim value.",
							Computed:    true,
						},
					},
				},
			},
			"slug": {
				Type:        schema.TypeString,
				Description: "The slug identifies the oidc.",
				Computed:    true,
			},
			"slug_perm": {
				Type:         schema.TypeString,
				Description:  "The slug_perm identifies the oidc.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}
