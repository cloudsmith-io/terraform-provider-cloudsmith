// Package cloudsmith ...
package cloudsmith

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceNamespaceRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	slug := d.Get("slug").(string)

	namespace, _, err := pc.APIClient.NamespacesApi.NamespacesRead(pc.Auth, slug)
	if err != nil {
		return err
	}

	d.SetId(namespace.SlugPerm)
	d.Set("name", namespace.Name)
	d.Set("slug", namespace.Slug)
	d.Set("slug_perm", namespace.SlugPerm)
	d.Set("type_name", namespace.TypeName)

	return nil
}

func dataSourceNamespace() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNamespaceRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"slug": {
				Type:     schema.TypeString,
				Required: true,
			},
			"slug_perm": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
