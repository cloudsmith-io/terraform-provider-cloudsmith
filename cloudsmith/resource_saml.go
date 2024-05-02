package cloudsmith

import (
	"context"
	"fmt"
	"strconv"
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

	return samlRead(d, m)
}

func retrieveSAMLSyncListPage(pc *providerConfig, organization string, pageSize int64, pageCount int64) ([]cloudsmith.OrganizationGroupSync, int64, error) {
	req := pc.APIClient.OrgsApi.OrgsSamlGroupSyncList(pc.Auth, organization)
	req = req.Page(pageCount)
	req = req.PageSize(pageSize)

	samlPage, resp, err := pc.APIClient.OrgsApi.OrgsSamlGroupSyncListExecute(req)
	if err != nil {
		if is404(resp) {
			return nil, 0, nil
		}
		return nil, 0, err
	}

	pageTotal, err := strconv.ParseInt(resp.Header.Get("X-Pagination-Pagetotal"), 10, 64)
	if err != nil {
		return nil, 0, err
	}

	return samlPage, pageTotal, nil

}

func retrieveSAMLSyncListPages(pc *providerConfig, organization string, pageSize int64, pageCount int64) ([]cloudsmith.OrganizationGroupSync, error) {
	var pageCurrentCount int64 = 1

	// A negative or zero count is assumed to mean retrieve the largest size page
	samlList := []cloudsmith.OrganizationGroupSync{}
	if pageSize == -1 || pageSize == 0 {
		pageSize = 100
	}

	// If no count is supplied assumed to mean retrieve all pages
	// we have to retrieve a page to get this count
	if pageCount == -1 || pageCount == 0 {
		var samlPage []cloudsmith.OrganizationGroupSync
		var err error
		samlPage, pageCount, err = retrieveSAMLSyncListPage(pc, organization, pageSize, 1)
		if err != nil {
			return nil, err
		}
		samlList = append(samlList, samlPage...)
		pageCurrentCount++
	}

	for pageCurrentCount <= pageCount {
		samlPage, _, err := retrieveSAMLSyncListPage(pc, organization, pageSize, pageCount)
		if err != nil {
			return nil, err
		}
		samlList = append(samlList, samlPage...)
		pageCurrentCount++
	}

	return samlList, nil
}

func samlRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")

	var pageCount, pageSize int64 = -1, -1
	samlList, err := retrieveSAMLSyncListPages(pc, organization, pageSize, pageCount)
	if err != nil {
		return err
	}

	// Iterate over the saml array to find the matching item
	for _, item := range samlList {
		if item.GetSlugPerm() == d.Id() {
			d.Set("idp_key", item.IdpKey)
			d.Set("idp_value", item.IdpValue)
			d.Set("role", item.Role)
			d.Set("team", item.Team)
			d.Set("slug_perm", item.SlugPerm)

			// namespace is not returned from the saml group endpoint so we rely on the input value
			d.Set("organization", organization)
			return nil
		}
	}

	// If no matching item is found, unset the ID and return
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
