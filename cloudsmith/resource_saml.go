package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func samlImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<saml_slug_perm>, got: %s", d.Id(),
		)
	}

	d.Set("organization", idParts[0])
	d.SetId(idParts[1])
	return []*schema.ResourceData{d}, nil
}

func samlCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")
	req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncCreate(pc.Auth, organization)
	req = req.Data(cloudsmith.OrganizationGroupSyncRequest{
		IdpKey:       requiredString(d, "idp_key"),
		IdpValue:     requiredString(d, "idp_value"),
		Role:         optionalString(d, "role"), // default to Member
		Team:         requiredString(d, "team"),
		Organization: requiredString(d, "organization"),
	})

	saml, _, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncCreateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(saml.GetSlugPerm())

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncList(pc.Auth, organization)
		_, resp, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncListExecute(req)
		if err != nil {
			if resp != nil {
				if is404(resp) {
					return errKeepWaiting
				}
				if resp.StatusCode == 422 {
					return fmt.Errorf("team does not exist, please check that the team exist")
				}
			}
			return err
		}
		return nil
	}

	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for SAML group sync (%s) to be created: %w", d.Id(), err)
	}

	if samlEnabledConfigured(d) {
		if err := samlSetEnabled(d, m); err != nil {
			return err
		}
	}

	return samlRead(d, m)
}

// samlEnabledConfigured reports whether `enabled` is explicitly set in the
// configuration. It's an organization-wide setting shared by all SAML Group
// Sync mappings, so we only manage it when the user opts in.
func samlEnabledConfigured(d *schema.ResourceData) bool {
	rawConfig := d.GetRawConfig()
	if rawConfig.IsNull() || !rawConfig.IsKnown() {
		return false
	}

	return !rawConfig.GetAttr("enabled").IsNull()
}

func samlSetEnabled(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	organization := requiredString(d, "organization")
	enabled := requiredBool(d, "enabled")

	if enabled {
		req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncEnable(pc.Auth, organization)
		_, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncEnableExecute(req)
		if err != nil {
			return fmt.Errorf(
				"error enabling SAML group sync for organization (%s): %w", organization, formatAPIError(err),
			)
		}
	} else {
		req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncDisable(pc.Auth, organization)
		_, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncDisableExecute(req)
		if err != nil {
			return fmt.Errorf(
				"error disabling SAML group sync for organization (%s): %w", organization, formatAPIError(err),
			)
		}
	}

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncStatus(pc.Auth, organization)
		status, resp, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncStatusExecute(req)
		if err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return formatAPIError(err)
		}
		if status.GetSamlGroupSyncStatus() != enabled {
			return errKeepWaiting
		}
		return nil
	}

	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for SAML group sync (%s) to become enabled=%t: %w", organization, enabled, err)
	}

	return nil
}

func samlRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")

	exec := func(page, ps int64) ([]cloudsmith.OrganizationGroupSync, *http.Response, error) {
		req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncList(pc.Auth, organization).
			Page(page).
			PageSize(ps)
		results, resp, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncListExecute(req)
		if is404(resp) {
			return nil, resp, nil
		}
		return results, resp, err
	}
	samlList, err := PaginateAllHTTP[cloudsmith.OrganizationGroupSync](exec, PaginationOptions{})
	if err != nil {
		return err
	}

	for _, item := range samlList {
		if item.GetSlugPerm() == d.Id() {
			d.Set("idp_key", item.IdpKey)
			d.Set("idp_value", item.IdpValue)
			d.Set("role", item.Role)
			d.Set("team", item.Team)
			d.Set("slug_perm", item.SlugPerm)

			statusReq := pc.APIClient.OrgsApi.OrgsSamlGroupSyncStatus(pc.Auth, organization)
			status, resp, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncStatusExecute(statusReq)
			if err != nil {
				if is404(resp) {
					d.SetId("")
					return nil
				}
				return err
			}
			d.Set("enabled", status.GetSamlGroupSyncStatus())

			// namespace is not returned from the saml group endpoint so we rely on the input value
			d.Set("organization", organization)
			return nil
		}
	}

	d.SetId("")
	return nil
}

func samlDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	organization := requiredString(d, "organization")

	req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncDelete(pc.Auth, organization, d.Id())
	_, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncDeleteExecute(req)
	if err != nil {
		return err
	}

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncList(pc.Auth, organization)
		_, resp, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncListExecute(req)
		if err != nil {
			if resp != nil {
				if is404(resp) {
					return nil
				}
			}
			return err
		}
		return nil
	}

	if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
		return fmt.Errorf("error waiting for SAML group sync (%s) to be deleted: %w", d.Id(), err)
	}
	return nil
}

// This is a workaround for not having a proper update endpoint for SAML group sync, we are recreating the entry based on new+old values
func samlUpdate(d *schema.ResourceData, m interface{}) error {
	if !d.HasChanges("idp_key", "idp_value", "role", "team") {
		if d.HasChange("enabled") && samlEnabledConfigured(d) {
			if err := samlSetEnabled(d, m); err != nil {
				return err
			}
		}
		return samlRead(d, m)
	}

	if err := samlDelete(d, m); err != nil {
		return err
	}
	return samlCreate(d, m)
}

func resourceSAML() *schema.Resource {
	return &schema.Resource{
		Create: samlCreate,
		Read:   samlRead,
		Update: samlUpdate,
		Delete: samlDelete,
		Importer: &schema.ResourceImporter{
			StateContext: samlImport,
		},
		Schema: map[string]*schema.Schema{
			"organization": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"idp_key": {
				Type:     schema.TypeString,
				Required: true,
			},
			"idp_value": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "Member",
				ValidateFunc: validation.StringInSlice([]string{"Member", "Manager"}, false),
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "Whether SAML Group Sync is enabled for the organization.",
				Optional:    true,
				Computed:    true,
			},
			"team": {
				Type:     schema.TypeString,
				Required: true,
			},
			"slug_perm": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
