package cloudsmith

import (
	"context"
	"fmt"
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
	req := pc.APIClient.OrgsApi.OrgsOpenidConnectCreate(pc.Auth, namespace)
	serviceAccounts := d.Get("service_accounts").([]interface{})
	serviceAccountsStr := make([]string, len(serviceAccounts))
	for i, v := range serviceAccounts {
		serviceAccountsStr[i] = v.(string)
	}
	req = req.Data(cloudsmith.ProviderSettingsRequest{
		Claims:          d.Get("claims").(map[string]interface{}),
		Enabled:         requiredBool(d, "enabled"),
		Name:            requiredString(d, "name"),
		ProviderUrl:     requiredString(d, "provider_url"),
		ServiceAccounts: serviceAccountsStr,
	})

	oidc, _, err := pc.APIClient.OrgsApi.OrgsOpenidConnectCreateExecute(req)
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

	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
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
	d.Set("service_accounts", oidc.GetServiceAccounts())
	d.Set("claims", oidc.GetClaims())
	d.Set("slug_perm", oidc.GetSlugPerm())
	d.Set("slug", oidc.GetSlug())
	d.SetId(oidc.GetSlugPerm())
	return nil
}

func oidcUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")

	req := pc.APIClient.OrgsApi.OrgsOpenidConnectPartialUpdate(pc.Auth, namespace, d.Id())
	serviceAccounts := d.Get("service_accounts").([]interface{})
	serviceAccountsStr := make([]string, len(serviceAccounts))
	for i, v := range serviceAccounts {
		serviceAccountsStr[i] = v.(string)
	}
	req = req.Data(cloudsmith.ProviderSettingsRequestPatch{
		Claims:          d.Get("claims").(map[string]interface{}),
		Enabled:         optionalBool(d, "enabled"),
		Name:            optionalString(d, "name"),
		ProviderUrl:     optionalString(d, "provider_url"),
		ServiceAccounts: serviceAccountsStr,
	})

	oidc, _, err := pc.APIClient.OrgsApi.OrgsOpenidConnectPartialUpdateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(oidc.GetSlugPerm())

	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for a
		// oidc being updated (changes incoming on the API side)
		time.Sleep(time.Second * 5)
		return nil
	}

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
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The service accounts associated with these provider settings",
				Required:    true,
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
