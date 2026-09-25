// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

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
		DeprecationMessage: "use cloudsmith_organization data source instead",

		Read: dataSourceNamespaceRead,
		Description: "!> **WARNING:** This data source is deprecated and will be removed in future. Use `cloudsmith_organization` instead.\n" +
			"The `namespace` data source allows fetching of metadata about a given Cloudsmith namespace. The fetched data can be used to resolve permanent identifiers from a namespace's user-facing name. These identifiers can then be passed to other resources to allow more consistent identification as user-facing names can change.",

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
				Description: "Whether this is a user or an organization namespace.",
				Computed:    true,
			},
		},
	}
}
