package cloudsmith

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
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
				Type:        schema.TypeString,
				Description: "A descriptive name for the namespace.",
				Computed:    true,
			},
			"slug": {
				Type:         schema.TypeString,
				Description:  "The slug identifies the namespace in URIs.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"slug_perm": {
				Type: schema.TypeString,
				Description: "The slug_perm immutably identifies the namespace. " +
					"It will never change once a namespace has been created.",
				Computed: true,
			},
			"type_name": {
				Type:        schema.TypeString,
				Description: "Is this a user or an organization namespace?",
				Computed:    true,
			},
		},
	}
}
