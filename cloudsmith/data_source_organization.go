package cloudsmith

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOrganizationRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	slug := requiredString(d, "slug")

	req := pc.APIClient.OrgsApi.OrgsRead(pc.Auth, slug)
	organization, _, err := pc.APIClient.OrgsApi.OrgsReadExecute(req)
	if err != nil {
		return err
	}

	d.Set("country", organization.GetCountry())
	d.Set("created_at", timeToString(organization.GetCreatedAt()))
	d.Set("location", organization.GetLocation())
	d.Set("name", organization.GetName())
	d.Set("slug", organization.GetSlug())
	d.Set("slug_perm", organization.GetSlugPerm())
	d.Set("tagline", organization.GetTagline())

	d.SetId(organization.GetSlugPerm())

	return nil
}

//nolint:funlen
func dataSourceOrganization() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceOrganizationRead,

		Schema: map[string]*schema.Schema{
			"country": {
				Type:        schema.TypeString,
				Description: "Country in which the organization is based.",
				Computed:    true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "ISO 8601 timestamp at which the organization was created.",
				Computed:    true,
			},
			"location": {
				Type:        schema.TypeString,
				Description: "The city/town/area in which the organization is based.",
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "A descriptive name for the organization.",
				Computed:    true,
			},
			"slug": {
				Type:         schema.TypeString,
				Description:  "The slug identifies the organization in URIs.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"slug_perm": {
				Type: schema.TypeString,
				Description: "The slug_perm immutably identifies the organization. " +
					"It will never change once an organization has been created.",
				Computed: true,
			},
			"tagline": {
				Type:        schema.TypeString,
				Description: "A short public description for the organization.",
				Computed:    true,
			},
		},
	}
}
