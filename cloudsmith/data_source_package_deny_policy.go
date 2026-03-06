package cloudsmith

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePackageDenyPolicyRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	slugPerm := requiredString(d, "slug_perm")

	req := pc.APIClient.OrgsApi.OrgsDenyPolicyRead(pc.Auth, namespace, slugPerm)
	packageDenyPolicy, _, err := pc.APIClient.OrgsApi.OrgsDenyPolicyReadExecute(req)
	if err != nil {
		return err
	}

	d.Set("name", packageDenyPolicy.GetName())
	d.Set("description", packageDenyPolicy.GetDescription())
	d.Set("package_query", packageDenyPolicy.GetPackageQueryString())
	d.Set("enabled", packageDenyPolicy.GetEnabled())
	d.Set("namespace", namespace)
	d.Set("slug_perm", packageDenyPolicy.GetSlugPerm())
	d.SetId(fmt.Sprintf("%s.%s", namespace, packageDenyPolicy.GetSlugPerm()))

	return nil
}

//nolint:funlen
func dataSourcePackageDenyPolicy() *schema.Resource {
	return &schema.Resource{
		Read:        dataSourcePackageDenyPolicyRead,
		Description: "Get an existing package deny policy in a Cloudsmith namespace.",

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "A descriptive name for the package deny policy.",
				Computed:    true,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "Description of the package deny policy.",
				Computed:    true,
			},
			"package_query": {
				Type:        schema.TypeString,
				Description: "The query to match the packages to be blocked.",
				Computed:    true,
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "Is the package deny policy enabled?.",
				Computed:    true,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "Namespace to which this package deny policy belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"slug_perm": {
				Type:         schema.TypeString,
				Description:  "Identifier of the package deny policy.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}
