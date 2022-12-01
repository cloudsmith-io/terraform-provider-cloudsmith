package cloudsmith

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceNamespaceRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	slug := requiredString(d, "slug")

	req := pc.APIClient.NamespacesApi.NamespacesRead(pc.Auth, slug)
	namespace, _, err := pc.APIClient.NamespacesApi.NamespacesReadExecute(req)
	if err != nil {
		return err
	}

	d.SetId(namespace.GetSlugPerm())
	d.Set("name", namespace.GetName())
	d.Set("slug", namespace.GetSlug())
	d.Set("slug_perm", namespace.GetSlugPerm())
	d.Set("type_name", namespace.GetTypeName())

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
