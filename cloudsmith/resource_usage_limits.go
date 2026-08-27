package cloudsmith

import (
	"context"
	"fmt"

	cloudsmithapi "github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	usageLimitsOrganization           = "organization"
	usageLimitsAllowOpenSourceOverage = "allow_open_source_overage"
	usageLimitsBandwidthOverageLimit  = "bandwidth_overage_limit"
	usageLimitsStorageOverageLimit    = "storage_overage_limit"
	usageLimitsBandwidthMaximum       = "bandwidth_maximum"
	usageLimitsStorageMaximum         = "storage_maximum"
)

func resourceUsageLimits() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUsageLimitsCreate,
		ReadContext:   resourceUsageLimitsRead,
		UpdateContext: resourceUsageLimitsUpdate,
		DeleteContext: resourceUsageLimitsDelete,

		Importer: &schema.ResourceImporter{StateContext: resourceUsageLimitsImport},

		Schema: map[string]*schema.Schema{
			usageLimitsOrganization: {
				Type:         schema.TypeString,
				Description:  "The slug of the Cloudsmith organization whose usage limits are managed.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			usageLimitsAllowOpenSourceOverage: {
				Type:        schema.TypeBool,
				Description: "Whether on-demand open source usage is allowed.",
				Required:    true,
			},
			usageLimitsBandwidthOverageLimit: {
				Type:         schema.TypeInt,
				Description:  "The package delivery on-demand limit in GB.",
				Required:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},
			usageLimitsStorageOverageLimit: {
				Type:         schema.TypeInt,
				Description:  "The artifact data on-demand limit in GB.",
				Required:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},
			usageLimitsBandwidthMaximum: {
				Type:        schema.TypeInt,
				Description: "The maximum package delivery overage allowed by the organization's plan, in GB.",
				Computed:    true,
			},
			usageLimitsStorageMaximum: {
				Type:        schema.TypeInt,
				Description: "The maximum artifact data overage allowed by the organization's plan, in GB.",
				Computed:    true,
			},
		},
	}
}

func resourceUsageLimitsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if diags := updateUsageLimits(d, m); diags != nil {
		return diags
	}

	d.SetId(requiredString(d, usageLimitsOrganization))
	return resourceUsageLimitsRead(ctx, d, m)
}

func resourceUsageLimitsRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)
	organization := requiredString(d, usageLimitsOrganization)

	limits, response, err := pc.APIClient.OrgsApi.OrgsRetrieveUsageLimits(pc.Auth, organization).Execute()
	if err != nil {
		if is404(response) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("error reading usage limits for organization %q: %w", organization, formatAPIError(err)))
	}

	if err := setUsageLimitsFields(d, organization, limits); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(organization)

	return nil
}

func setUsageLimitsFields(d *schema.ResourceData, organization string, limits *cloudsmithapi.OrgsRetrieveUsageLimits200Response) error {
	fields := map[string]interface{}{
		usageLimitsOrganization:           organization,
		usageLimitsAllowOpenSourceOverage: limits.GetAllowOpenSourceOverage(),
		usageLimitsBandwidthOverageLimit:  limits.GetBandwidthOverageLimit(),
		usageLimitsStorageOverageLimit:    limits.GetStorageOverageLimit(),
		usageLimitsBandwidthMaximum:       limits.GetBandwidthMaximum(),
		usageLimitsStorageMaximum:         limits.GetStorageMaximum(),
	}

	for name, value := range fields {
		if err := d.Set(name, value); err != nil {
			return fmt.Errorf("error setting usage limits field %q: %w", name, err)
		}
	}
	return nil
}

func resourceUsageLimitsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if diags := updateUsageLimits(d, m); diags != nil {
		return diags
	}

	return resourceUsageLimitsRead(ctx, d, m)
}

func updateUsageLimits(d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)
	organization := requiredString(d, usageLimitsOrganization)
	allowOpenSourceOverage := d.Get(usageLimitsAllowOpenSourceOverage).(bool)
	bandwidthOverageLimit := int64(d.Get(usageLimitsBandwidthOverageLimit).(int))
	storageOverageLimit := int64(d.Get(usageLimitsStorageOverageLimit).(int))

	request := cloudsmithapi.NewOrganizationUsageUpdateRequestPatch()
	request.SetAllowOpenSourceOverage(allowOpenSourceOverage)
	request.SetBandwidthOverageLimit(bandwidthOverageLimit)
	request.SetStorageOverageLimit(storageOverageLimit)

	_, _, err := pc.APIClient.OrgsApi.OrgsUpdateUsageLimits(pc.Auth, organization).Data(*request).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating usage limits for organization %q: %w", organization, formatAPIError(err)))
	}

	checker := func() error {
		limits, _, err := pc.APIClient.OrgsApi.OrgsRetrieveUsageLimits(pc.Auth, organization).Execute()
		if err != nil {
			return formatAPIError(err)
		}
		if limits.GetAllowOpenSourceOverage() != allowOpenSourceOverage ||
			limits.GetBandwidthOverageLimit() != bandwidthOverageLimit ||
			limits.GetStorageOverageLimit() != storageOverageLimit {
			return errKeepWaiting
		}
		return nil
	}

	if err := checker(); err != nil {
		if err != errKeepWaiting {
			return diag.FromErr(fmt.Errorf("error reading updated usage limits for organization %q: %w", organization, err))
		}
		if err := waiter(checker, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
			return diag.FromErr(fmt.Errorf("error waiting for usage limits for organization %q to update: %w", organization, err))
		}
	}

	return nil
}

func resourceUsageLimitsDelete(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// Cloudsmith always has a usage-limits configuration for an organization and
	// the API has no delete/reset operation. Forget the Terraform resource without
	// changing the organization's live settings.
	d.SetId("")
	return nil
}

func resourceUsageLimitsImport(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	organization := d.Id()
	if organization == "" {
		return nil, fmt.Errorf("usage limits import ID must be an organization slug")
	}

	if err := d.Set(usageLimitsOrganization, organization); err != nil {
		return nil, fmt.Errorf("error setting organization during usage limits import: %w", err)
	}
	d.SetId(organization)
	return []*schema.ResourceData{d}, nil
}
