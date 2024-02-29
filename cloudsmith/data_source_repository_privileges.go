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

// dataSourceRepositoryPrivilegesRead retrieves privileges information for the specified repository.
func dataSourceRepositoryPrivilegesRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := d.Get("organization").(string)
	repository := d.Get("repository").(string)

	req := pc.APIClient.ReposApi.ReposPrivilegesList(pc.Auth, organization, repository)
	// TODO: add a proper loop here to ensure we always get all privs,
	// regardless of how many are configured.
	req = req.Page(1)
	req = req.PageSize(1000)
	privileges, resp, err := pc.APIClient.ReposApi.ReposPrivilegesListExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("service", flattenRepositoryPrivilegeServices(privileges.GetPrivileges()))
	d.Set("team", flattenRepositoryPrivilegeTeams(privileges.GetPrivileges()))
	d.Set("user", flattenRepositoryPrivilegeUsers(privileges.GetPrivileges()))

	d.SetId(fmt.Sprintf("%s/%s", organization, repository))

	return nil
}
