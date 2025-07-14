package cloudsmith

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// waitForEntitlementControlEnabledResource polls until the entitlement token's enabled state matches wantEnabled or times out.
func waitForEntitlementControlEnabledResource(pc *providerConfig, namespace, repository, identifier string, wantEnabled bool, timeoutSec int) error {
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
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for entitlement control enabled=%v", wantEnabled)
		}
		time.Sleep(1 * time.Second)
	}
}

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
	if err := waitForEntitlementControlEnabledResource(pc, namespace, repository, identifier, enabled, 15); err != nil {
		return err
	}
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
	// Wait for the entitlement to reach the desired state
	if err := waitForEntitlementControlEnabledResource(pc, namespace, repository, d.Id(), enabled, 30); err != nil {
		return err
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
	// Wait for the entitlement to be disabled
	if err := waitForEntitlementControlEnabledResource(pc, namespace, repository, d.Id(), false, 30); err != nil {
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
