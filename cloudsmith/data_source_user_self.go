package cloudsmith

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUserSelfRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	req := pc.APIClient.UserApi.UserSelf(pc.Auth)
	userSelf, _, err := pc.APIClient.UserApi.UserSelfExecute(req)
	if err != nil {
		return err
	}

	d.Set("email", userSelf.GetEmail())
	d.Set("name", userSelf.GetName())
	d.Set("slug", userSelf.GetSlug())
	d.Set("slug_perm", userSelf.GetSlugPerm())

	d.SetId(userSelf.GetSlugPerm())

	return nil
}

// dataSourceUserSelf returns the schema and implementation for the data source
// that provides information about the currently authenticated user.
func dataSourceUserSelf() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceUserSelfRead,

		Schema: map[string]*schema.Schema{
			"email": {
				Type:        schema.TypeString,
				Description: "The user's email address",
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The user's full name",
				Computed:    true,
			},
			"slug": {
				Type:        schema.TypeString,
				Description: "The slug identifies the user in URIs",
				Computed:    true,
			},
			"slug_perm": {
				Type:        schema.TypeString,
				Description: "The slug_perm immutably identifies the user",
				Computed:    true,
			},
		},
	}
}
