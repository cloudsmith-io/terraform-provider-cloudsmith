package cloudsmith

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceUsageLimits() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceUsageLimitsRead,

		Schema: map[string]*schema.Schema{
			usageLimitsOrganization: {
				Type:         schema.TypeString,
				Description:  "The slug of the Cloudsmith organization whose usage limits are read.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			usageLimitsAllowOpenSourceOverage: {
				Type:        schema.TypeBool,
				Description: "Whether on-demand open source usage is allowed.",
				Computed:    true,
			},
			usageLimitsBandwidthOverageLimit: {
				Type:        schema.TypeInt,
				Description: "The package delivery on-demand limit in GB.",
				Computed:    true,
			},
			usageLimitsStorageOverageLimit: {
				Type:        schema.TypeInt,
				Description: "The artifact data on-demand limit in GB.",
				Computed:    true,
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

func dataSourceUsageLimitsRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)
	organization := requiredString(d, usageLimitsOrganization)

	limits, _, err := pc.APIClient.OrgsApi.OrgsRetrieveUsageLimits(pc.Auth, organization).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading usage limits for organization %q: %w", organization, formatAPIError(err)))
	}

	if err := setUsageLimitsFields(d, organization, limits); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(organization)
	return nil
}
