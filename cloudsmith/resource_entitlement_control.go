package cloudsmith

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func entitlementControlImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 3 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <namespace>.<repository>.<identifier>, got: %s", d.Id(),
		)
	}

	d.Set("namespace", idParts[0])
	d.Set("repository", idParts[1])
	d.SetId(idParts[2])
	return []*schema.ResourceData{d}, nil
}

func entitlementControlCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")
	identifier := requiredString(d, "identifier")
	enabled := requiredBool(d, "enabled")

	if enabled {
		req := pc.APIClient.EntitlementsApi.EntitlementsEnable(pc.Auth, namespace, repository, identifier)
		_, err := pc.APIClient.EntitlementsApi.EntitlementsEnableExecute(req)
		if err != nil {
			return err
		}
	} else {
		req := pc.APIClient.EntitlementsApi.EntitlementsDisable(pc.Auth, namespace, repository, identifier)
		_, err := pc.APIClient.EntitlementsApi.EntitlementsDisableExecute(req)
		if err != nil {
			return err
		}
	}

	d.SetId(identifier)
	return entitlementControlRead(d, m)
}

func entitlementControlRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, d.Id())
	entitlement, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req)

	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("enabled", entitlement.GetIsActive())
	return nil
}

func entitlementControlUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")
	enabled := requiredBool(d, "enabled")

	if enabled {
		req := pc.APIClient.EntitlementsApi.EntitlementsEnable(pc.Auth, namespace, repository, d.Id())
		_, err := pc.APIClient.EntitlementsApi.EntitlementsEnableExecute(req)
		if err != nil {
			return err
		}
	} else {
		req := pc.APIClient.EntitlementsApi.EntitlementsDisable(pc.Auth, namespace, repository, d.Id())
		_, err := pc.APIClient.EntitlementsApi.EntitlementsDisableExecute(req)
		if err != nil {
			return err
		}
	}

	return entitlementControlRead(d, m)
}

func entitlementControlDelete(d *schema.ResourceData, m interface{}) error {
	// We don't actually delete the entitlement, just disable it
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	req := pc.APIClient.EntitlementsApi.EntitlementsDisable(pc.Auth, namespace, repository, d.Id())
	_, err := pc.APIClient.EntitlementsApi.EntitlementsDisableExecute(req)
	if err != nil {
		return err
	}

	return nil
}

func resourceEntitlementControl() *schema.Resource {
	return &schema.Resource{
		Create: entitlementControlCreate,
		Read:   entitlementControlRead,
		Update: entitlementControlUpdate,
		Delete: entitlementControlDelete,

		Importer: &schema.ResourceImporter{
			StateContext: entitlementControlImport,
		},

		Schema: map[string]*schema.Schema{
			"namespace": {
				Type:         schema.TypeString,
				Description:  "Namespace to which this entitlement belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"repository": {
				Type:         schema.TypeString,
				Description:  "Repository to which this entitlement belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"identifier": {
				Type:         schema.TypeString,
				Description:  "The identifier (slug_perm) of the entitlement token.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "Whether the entitlement token is enabled or disabled.",
				Required:    true,
			},
		},
	}
}
