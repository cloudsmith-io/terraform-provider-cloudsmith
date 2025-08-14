package cloudsmith

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func oidcImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<oidc_slug_perm>, got: %s", d.Id(),
		)
	}

	d.Set("namespace", idParts[0])
	d.SetId(idParts[1])
	return []*schema.ResourceData{d}, nil
}

func oidcCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	reqBuilder := pc.APIClient.OrgsApi.OrgsOpenidConnectCreate(pc.Auth, namespace)

	mappingClaim, hasMappingClaim := d.GetOk("mapping_claim")
	dynMappingsRaw, hasDynMappings := d.GetOk("dynamic_mappings")
	staticSvcAcctsRaw, hasServiceAccounts := d.GetOk("service_accounts")

	base := cloudsmith.NewProviderSettingsWriteRequest(
		d.Get("claims").(map[string]interface{}),
		requiredBool(d, "enabled"),
		requiredString(d, "name"),
		requiredString(d, "provider_url"),
	)

	if hasMappingClaim || hasDynMappings { // dynamic
		if hasMappingClaim && mappingClaim.(string) != "" {
			base.SetMappingClaim(mappingClaim.(string))
		}
		if hasDynMappings {
			dmObjects := buildDynamicMappingObjectsFromSet(dynMappingsRaw.(*schema.Set))
			base.SetDynamicMappings(dmObjects)
		}
	} else if hasServiceAccounts { // static
		svcList := convertInterfaceListToStrings(staticSvcAcctsRaw.([]interface{}))
		if len(svcList) > 0 {
			base.SetServiceAccounts(svcList)
		}
	}

	reqBuilder = reqBuilder.Data(*base)
	oidc, _, err := pc.APIClient.OrgsApi.OrgsOpenidConnectCreateExecute(reqBuilder)
	if err != nil {
		return err
	}
	d.SetId(oidc.GetSlugPerm())

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsOpenidConnectRead(pc.Auth, namespace, d.Id())
		_, resp, err := pc.APIClient.OrgsApi.OrgsOpenidConnectReadExecute(req)
		if err != nil {
			if resp != nil {
				if is404(resp) {
					return errKeepWaiting
				}
				if resp.StatusCode == 422 {
					return fmt.Errorf("service account does not exist, please check that the service account exist")
				}
			}
			return err
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for OIDC config (%s) to be updated: %w", d.Id(), err)
	}
	return oidcRead(d, m)
}

func oidcRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	req := pc.APIClient.OrgsApi.OrgsOpenidConnectRead(pc.Auth, namespace, d.Id())
	oidc, resp, err := pc.APIClient.OrgsApi.OrgsOpenidConnectReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", oidc.GetName())
	d.Set("enabled", oidc.GetEnabled())
	d.Set("provider_url", oidc.GetProviderUrl())
	if sa := oidc.GetServiceAccounts(); len(sa) > 0 {
		sort.Strings(sa)
		d.Set("service_accounts", sa)
	} else {
		d.Set("service_accounts", []string{})
	}
	d.Set("claims", oidc.GetClaims())
	d.Set("slug_perm", oidc.GetSlugPerm())
	d.Set("slug", oidc.GetSlug())

	// mapping_claim: use API value only
	mappingClaim := ""
	if getter, ok := any(oidc).(interface{ GetMappingClaimOk() (*string, bool) }); ok {
		if mc, ok2 := getter.GetMappingClaimOk(); ok2 && mc != nil {
			mappingClaim = *mc
		}
	}
	d.Set("mapping_claim", mappingClaim)

	apiMappings, err := retrieveAllDynamicMappings(pc, namespace, d.Id())
	if err != nil {
		return fmt.Errorf("error retrieving dynamic mappings: %w", err)
	}
	// Set dynamic mappings as returned by API only
	if len(apiMappings) > 0 || mappingClaim != "" {
		d.Set("dynamic_mappings", convertMappingsToSetElems(apiMappings))
		if len(apiMappings) > 0 {
			d.Set("service_accounts", []string{})
		}
	} else {
		d.Set("dynamic_mappings", []interface{}{})
	}

	d.SetId(oidc.GetSlugPerm())
	return nil
}

func oidcUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	reqBuilder := pc.APIClient.OrgsApi.OrgsOpenidConnectPartialUpdate(pc.Auth, namespace, d.Id())
	patch := cloudsmith.NewProviderSettingsWriteRequestPatch()

	patch.SetClaims(d.Get("claims").(map[string]interface{}))
	if v, ok := d.GetOkExists("enabled"); ok {
		patch.SetEnabled(v.(bool))
	}
	if v, ok := d.GetOkExists("name"); ok {
		patch.SetName(v.(string))
	}
	if v, ok := d.GetOkExists("provider_url"); ok {
		patch.SetProviderUrl(v.(string))
	}

	mappingClaim, hasMappingClaim := d.GetOk("mapping_claim")
	dynMappingsRaw, hasDynMappings := d.GetOk("dynamic_mappings")
	svcAcctsRaw, hasServiceAccounts := d.GetOk("service_accounts")

	if hasMappingClaim || hasDynMappings { // dynamic target state
		if hasMappingClaim && mappingClaim.(string) != "" {
			patch.SetMappingClaim(mappingClaim.(string))
		} else {
			patch.SetMappingClaimNil()
		}
		dmObjects := []cloudsmith.DynamicMapping{}
		if hasDynMappings {
			dmObjects = buildDynamicMappingObjectsFromSet(dynMappingsRaw.(*schema.Set))
		}
		patch.SetDynamicMappings(dmObjects)
		patch.SetServiceAccounts([]string{})
	} else if hasServiceAccounts { // static target state
		svcList := convertInterfaceListToStrings(svcAcctsRaw.([]interface{}))
		patch.SetServiceAccounts(svcList)
		patch.SetDynamicMappings([]cloudsmith.DynamicMapping{})
		patch.SetMappingClaimNil()
	}

	reqBuilder = reqBuilder.Data(*patch)
	oidc, _, err := pc.APIClient.OrgsApi.OrgsOpenidConnectPartialUpdateExecute(reqBuilder)
	if err != nil {
		return err
	}
	d.SetId(oidc.GetSlugPerm())

	checkerFunc := func() error { time.Sleep(5 * time.Second); return nil }
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for OIDC config (%s) to be updated: %w", d.Id(), err)
	}
	return oidcRead(d, m)
}

func oidcDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")

	req := pc.APIClient.OrgsApi.OrgsOpenidConnectDelete(pc.Auth, namespace, d.Id())
	_, err := pc.APIClient.OrgsApi.OrgsOpenidConnectDeleteExecute(req)
	if err != nil {
		return err
	}

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsOpenidConnectRead(pc.Auth, namespace, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsOpenidConnectReadExecute(req); err != nil {
			if is404(resp) {
				return nil
			}
			return err
		}
		return errKeepWaiting
	}

	if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
		return fmt.Errorf("error waiting for OIDC config (%s) to be deleted: %w", d.Id(), err)
	}

	return nil
}

func resourceOIDC() *schema.Resource {
	return &schema.Resource{
		Create: oidcCreate,
		Read:   oidcRead,
		Update: oidcUpdate,
		Delete: oidcDelete,

		Importer: &schema.ResourceImporter{
			StateContext: oidcImport,
		},

		Schema: map[string]*schema.Schema{
			"claims": {
				Type:        schema.TypeMap,
				Description: "The claims associated with these provider settings",
				Required:    true,
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "Whether the provider settings should be used for incoming OIDC requests.",
				Required:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the provider settings are being configured for",
				Required:    true,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "Namespace to which this OIDC config belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"provider_url": {
				Type:         schema.TypeString,
				Description:  "The URL from the provider that serves as the base for the OpenID configuration.",
				Required:     true,
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"service_accounts": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Description:   "The service accounts associated with these provider settings (static providers only). Mutually exclusive with mapping_claim/dynamic_mappings.",
				Optional:      true,
				ConflictsWith: []string{"mapping_claim", "dynamic_mappings"},
			},
			"mapping_claim": {
				Type:          schema.TypeString,
				Description:   "The claim key whose values dynamically map to service accounts (dynamic providers only). Mutually exclusive with service_accounts.",
				Optional:      true,
				ConflictsWith: []string{"service_accounts"},
			},
			"dynamic_mappings": {
				Type:          schema.TypeSet,
				Description:   "Set of dynamic claim value -> service account mappings (order-insensitive, authoritative). Mutually exclusive with service_accounts.",
				Optional:      true,
				ConflictsWith: []string{"service_accounts"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"claim_value":     {Type: schema.TypeString, Required: true, Description: "The value of the mapping claim."},
						"service_account": {Type: schema.TypeString, Required: true, Description: "Service account slug mapped to the claim value."},
					},
				},
			},
			"slug": {
				Type:        schema.TypeString,
				Description: "The slug identifies the oidc.",
				Computed:    true,
			},
			"slug_perm": {
				Type:        schema.TypeString,
				Description: "The slug_perm identifies the oidc.",
				Computed:    true,
			},
		},
	}
}

// returns a list of maps for state.
func retrieveAllDynamicMappings(pc *providerConfig, namespace, slugPerm string) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	const pageSize int64 = 500
	var page int64 = 1
	for {
		req := pc.APIClient.OrgsApi.OrgsOpenidConnectDynamicMappingsList(pc.Auth, namespace, slugPerm).Page(page).PageSize(pageSize)
		dmList, resp, err := pc.APIClient.OrgsApi.OrgsOpenidConnectDynamicMappingsListExecute(req)
		if err != nil {
			if resp != nil && is404(resp) { // treat missing as none
				break
			}
			return nil, err
		}
		for _, dm := range dmList {
			result = append(result, map[string]interface{}{
				"claim_value":     dm.GetClaimValue(),
				"service_account": dm.GetServiceAccount(),
			})
		}
		if resp == nil {
			break
		}
		pageTotalStr := resp.Header.Get("X-Pagination-Pagetotal")
		if pageTotalStr == "" {
			break
		}
		totalPages, err := strconv.ParseInt(pageTotalStr, 10, 64)
		if err != nil || page >= totalPages {
			break
		}
		page++
	}
	return result, nil
}

// Helper utilities
func buildDynamicMappingObjectsFromSet(set *schema.Set) []cloudsmith.DynamicMapping {
	if set == nil {
		return nil
	}
	var out []cloudsmith.DynamicMapping
	for _, v := range set.List() {
		if v == nil {
			continue
		}
		m := v.(map[string]interface{})
		cv, _ := m["claim_value"].(string)
		sa, _ := m["service_account"].(string)
		if cv == "" || sa == "" {
			continue
		}
		out = append(out, *cloudsmith.NewDynamicMapping(cv, sa))
	}
	return out
}

func convertInterfaceListToStrings(in []interface{}) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = v.(string)
	}
	return out
}

func convertMappingsToSetElems(in []map[string]interface{}) []interface{} {
	out := make([]interface{}, len(in))
	for i, m := range in {
		out[i] = m
	}
	return out
}
