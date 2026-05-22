package cloudsmith

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// dataSourceRepositoryPrivileges returns the data source schema and read function.
func dataSourceRepositoryPrivileges() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRepositoryPrivilegesRead,

		Schema: map[string]*schema.Schema{
			"organization": {
				Type:        schema.TypeString,
				Description: "Organization to which this repository belongs.",
				Required:    true,
			},
			"repository": {
				Type:        schema.TypeString,
				Description: "Repository to fetch privileges information.",
				Required:    true,
			},
			"service": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"privilege": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"slug": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"team": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"privilege": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"slug": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"user": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"privilege": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"slug": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceRepositoryPrivilegesRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := d.Get("organization").(string)
	repository := d.Get("repository").(string)

	all, notFound, err := retrieveRepositoryPrivilegePages(pc, organization, repository)
	if err != nil {
		return err
	}
	if notFound {
		d.SetId("")
		return nil
	}

	d.Set("service", flattenRepositoryPrivilegeServices(all))
	d.Set("team", flattenRepositoryPrivilegeTeams(all))
	d.Set("user", flattenRepositoryPrivilegeUsers(all))

	d.SetId(fmt.Sprintf("%s/%s", organization, repository))

	return nil
}
