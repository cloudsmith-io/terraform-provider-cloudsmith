package cloudsmith

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	if saml == nil || saml.GetSlugPerm() == "" {
		return fmt.Errorf("SAML group sync create returned no permanent slug")
	}

	d.SetId(saml.GetSlugPerm())

	observed, err := samlWaitForMapping(pc, organization, d.Id(), true, defaultCreationTimeout, defaultCreationInterval)
	if err != nil {
		return fmt.Errorf("error waiting for SAML group sync (%s) to be created: %w", d.Id(), err)
	}

	if samlEnabledConfigured(d) {
		if err := samlSetEnabled(d, m); err != nil {
			return err
		}
	}

	// Keep the response that satisfied the waiter: another list may hit a stale replica.
	return samlSetState(d, pc, observed)
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
		if status == nil || !status.HasSamlGroupSyncStatus() {
			return fmt.Errorf("SAML group sync status response is missing its status")
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

func samlReadEnabled(pc *providerConfig, organization string) (bool, error) {
	readStatus := func() (*cloudsmith.OrganizationGroupSyncStatus, *http.Response, error) {
		req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncStatus(pc.Auth, organization)
		status, resp, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncStatusExecute(req)
		if err == nil && (status == nil || !status.HasSamlGroupSyncStatus()) {
			err = fmt.Errorf("SAML group sync status response is missing its status")
		}
		return status, resp, err
	}

	status, resp, err := readStatus()
	if err == nil {
		return status.GetSamlGroupSyncStatus(), nil
	}
	if !is404(resp) {
		return false, fmt.Errorf(
			"error reading SAML group sync status for organization (%s): %w", organization, formatAPIError(err),
		)
	}

	var enabled bool
	checkerFunc := func() error {
		status, resp, err := readStatus()
		if err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return formatAPIError(err)
		}
		enabled = status.GetSamlGroupSyncStatus()
		return nil
	}

	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return false, fmt.Errorf(
			"error waiting for SAML group sync status for organization (%s): %w", organization, err,
		)
	}

	return enabled, nil
}

func samlFindMapping(ctx context.Context, pc *providerConfig, organization, id string) (*cloudsmith.OrganizationGroupSync, error) {
	exec := func(page, ps int64) ([]cloudsmith.OrganizationGroupSync, *http.Response, error) {
		req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncList(ctx, organization).
			Page(page).
			PageSize(ps)
		results, resp, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncListExecute(req)
		if is404(resp) {
			return nil, resp, nil
		}
		return results, resp, formatAPIError(err)
	}
	samlList, err := PaginateAllHTTP[cloudsmith.OrganizationGroupSync](exec, PaginationOptions{})
	if err != nil {
		return nil, err
	}

	for _, item := range samlList {
		if item.GetSlugPerm() == id {
			return &item, nil
		}
	}
	return nil, nil
}

func samlWaitForMapping(pc *providerConfig, organization, id string, present bool, timeout, interval time.Duration) (*cloudsmith.OrganizationGroupSync, error) {
	ctx, cancel := context.WithTimeout(pc.Auth, timeout)
	defer cancel()
	for {
		item, err := samlFindMapping(ctx, pc, organization, id)
		if err != nil {
			return nil, err
		}
		if (item != nil) == present {
			return item, nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			if pc.Auth.Err() != nil {
				return nil, pc.Auth.Err()
			}
			return nil, errTimedOut
		case <-timer.C:
		}
	}
}

func samlSetState(d *schema.ResourceData, pc *providerConfig, item *cloudsmith.OrganizationGroupSync) error {
	enabled, err := samlReadEnabled(pc, requiredString(d, "organization"))
	if err != nil {
		return err
	}
	d.Set("idp_key", item.IdpKey)
	d.Set("idp_value", item.IdpValue)
	d.Set("role", item.Role)
	d.Set("team", item.Team)
	d.Set("slug_perm", item.SlugPerm)
	d.Set("enabled", enabled)
	return nil
}

func samlRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	// An absent list entry can be replication lag even after creation was observed.
	// Confirm absence for a bounded window, then honor legitimate external deletion.
	item, err := samlWaitForMapping(pc, requiredString(d, "organization"), d.Id(), true, defaultCreationTimeout, defaultCreationInterval)
	if errors.Is(err, errTimedOut) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}
	return samlSetState(d, pc, item)
}

func samlDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	organization := requiredString(d, "organization")

	req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncDelete(pc.Auth, organization, d.Id())
	resp, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncDeleteExecute(req)
	if is404(resp) {
		return nil
	}
	if err != nil {
		return err
	}

	if _, err := samlWaitForMapping(pc, organization, d.Id(), false, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
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
