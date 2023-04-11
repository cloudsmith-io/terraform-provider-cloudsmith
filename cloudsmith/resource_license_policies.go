package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	CreatedAt             string = "created_at"
	Description           string = "description"
	Name                  string = "name"
	AllowUnknownLicenses  string = "allow_unknown_licenses"
	OnViolationQuarantine string = "on_violation_quarantine"
	SlugPerm              string = "slug_perm"
	SpdxIdentifiers       string = "spdx_identifiers"
	UpdatedAt             string = "updated_at"
	Organization          string = "organization"
)

func importLicensePolicies(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>, got: %s", d.Id(),
		)
	}

	d.Set(Organization, idParts[0])
	d.SetId(idParts[1])
	return []*schema.ResourceData{d}, nil
}

func resourceLicensePoliciesCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, Organization)

	req := pc.APIClient.OrgsApi.OrgsLicensePolicyCreate(pc.Auth, org)
	req = req.Data(cloudsmith.OrganizationPackageLicensePolicyRequest{
		AllowUnknownLicenses:  optionalBool(d, AllowUnknownLicenses),
		Description:           nullableString(d, Description),
		Name:                  requiredString(d, Name),
		OnViolationQuarantine: optionalBool(d, OnViolationQuarantine),
		SpdxIdentifiers:       expandStrings(d, SpdxIdentifiers),
	})

	licensePolicy, resp, err := pc.APIClient.OrgsApi.OrgsLicensePolicyCreateExecute(req)
	if err != nil {
		if resp.StatusCode == http.StatusUnprocessableEntity {
			return fmt.Errorf("invalid spdx_identifiers: %v", expandStrings(d, SpdxIdentifiers))
		}
		return err
	}

	d.SetId(licensePolicy.GetSlugPerm())

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsLicensePolicyRead(pc.Auth, org, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsLicensePolicyReadExecute(req); err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return err
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for license policy (%s) to be created: %s", d.Id(), err)
	}

	return resourceLicensePoliciesRead(d, m)
}

func resourceLicensePoliciesUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, Organization)

	req := pc.APIClient.OrgsApi.OrgsLicensePolicyPartialUpdate(pc.Auth, org, d.Id())
	req = req.Data(cloudsmith.OrganizationPackageLicensePolicyRequestPatch{
		AllowUnknownLicenses:  optionalBool(d, AllowUnknownLicenses),
		Description:           nullableString(d, Description),
		Name:                  optionalString(d, Name),
		OnViolationQuarantine: optionalBool(d, OnViolationQuarantine),
		SpdxIdentifiers:       expandStrings(d, SpdxIdentifiers),
	})
	licensePolicy, resp, err := pc.APIClient.OrgsApi.OrgsLicensePolicyPartialUpdateExecute(req)
	if err != nil {
		if resp.StatusCode == http.StatusUnprocessableEntity {
			return fmt.Errorf("invalid spdx_identifiers: %v", expandStrings(d, SpdxIdentifiers))
		}
		return err
	}

	d.SetId(licensePolicy.GetSlugPerm())

	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for a
		// policy being updated (changes incoming on the API side)
		time.Sleep(time.Second * 5)
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for license policy (%s) to be updated: %w", d.Id(), err)
	}

	return resourceLicensePoliciesRead(d, m)
}

func resourceLicensePoliciesDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, Organization)

	req := pc.APIClient.OrgsApi.OrgsLicensePolicyDelete(pc.Auth, org, d.Id())
	_, err := pc.APIClient.OrgsApi.OrgsLicensePolicyDeleteExecute(req)
	if err != nil {
		return err
	}

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsLicensePolicyRead(pc.Auth, org, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsLicensePolicyReadExecute(req); err != nil {
			if is404(resp) {
				return nil
			}
			return err
		}
		return errKeepWaiting
	}
	if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
		return fmt.Errorf("error waiting for license policy (%s) to be deleted: %w", d.Id(), err)
	}

	return nil
}

func resourceLicensePoliciesRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, Organization)

	req := pc.APIClient.OrgsApi.OrgsLicensePolicyRead(pc.Auth, org, d.Id())

	licensePolicies, resp, err := pc.APIClient.OrgsApi.OrgsLicensePolicyReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return err
	}

	_ = d.Set(CreatedAt, licensePolicies.GetCreatedAt().String())
	_ = d.Set(Description, licensePolicies.GetDescription())
	_ = d.Set(Name, licensePolicies.GetName())
	_ = d.Set(OnViolationQuarantine, licensePolicies.GetOnViolationQuarantine())
	_ = d.Set(SlugPerm, licensePolicies.GetSlugPerm())
	_ = d.Set(SpdxIdentifiers, flattenStrings(licensePolicies.GetSpdxIdentifiers()))
	_ = d.Set(UpdatedAt, licensePolicies.GetUpdatedAt().String())

	// organization is not returned from the read
	// endpoint, so we can use the values stored in resource state. We rely on
	// ForceNew to ensure if either changes a new resource is created.
	_ = d.Set(Organization, org)

	return nil
}

//nolint:funlen
func resourceLicensePolicies() *schema.Resource {
	return &schema.Resource{
		Create: resourceLicensePoliciesCreate,
		Read:   resourceLicensePoliciesRead,
		Update: resourceLicensePoliciesUpdate,
		Delete: resourceLicensePoliciesDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importLicensePolicies,
		},

		Schema: map[string]*schema.Schema{
			CreatedAt: {
				Type:        schema.TypeString,
				Description: "The time the policy was created at.",
				Computed:    true,
			},
			Description: {
				Type:         schema.TypeString,
				Description:  "The description of the license policy.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			Name: {
				Type:         schema.TypeString,
				Description:  "The name of the license policy.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			OnViolationQuarantine: {
				Type:        schema.TypeBool,
				Description: "On violation of the license policy, quarantine violating packages.",
				Optional:    true,
				Default:     true,
			},
			SlugPerm: {
				Type:        schema.TypeString,
				Description: "Slug-perm of the license policy",
				Computed:    true,
			},
			SpdxIdentifiers: {
				Type:        schema.TypeSet,
				Description: "The licenses to deny.",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				Required: true,
			},
			UpdatedAt: {
				Type:        schema.TypeString,
				Description: "The time the policy last updated at.",
				Computed:    true,
			},
			Organization: {
				Type:         schema.TypeString,
				Description:  "Organization to which this policy belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}
