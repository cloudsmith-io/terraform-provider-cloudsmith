package cloudsmith

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func dataSourceRepositoryRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	name := requiredString(d, "identifier")

	req := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, name)
	repository, _, err := pc.APIClient.ReposApi.ReposReadExecute(req)
	if err != nil {
		return err
	}

	d.Set("cdn_url", repository.GetCdnUrl())
	d.Set("created_at", repository.GetCreatedAt())
	d.Set("deleted_at", repository.GetDeletedAt())
	d.Set("description", repository.GetDescription())
	d.Set("index_files", repository.GetIndexFiles())
	d.Set("namespace_url", repository.GetNamespaceUrl())
	d.Set("repository_type", repository.GetRepositoryTypeStr())
	d.Set("self_html_url", repository.GetSelfHtmlUrl())
	d.Set("self_url", repository.GetSelfUrl())
	d.Set("slug", repository.GetSlug())
	d.Set("slug_perm", repository.GetSlugPerm())
	d.Set("storage_region", repository.GetStorageRegion())

	d.SetId(fmt.Sprintf("%s_%s", namespace, name))

	return nil
}

//nolint:funlen
func dataSourceRepository() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRepositoryRead,

		Schema: map[string]*schema.Schema{
			"cdn_url": {
				Type:        schema.TypeString,
				Description: "Base URL from which packages and other artifacts are downloaded.",
				Computed:    true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "ISO 8601 timestamp at which the repository was created.",
				Computed:    true,
			},
			"deleted_at": {
				Type: schema.TypeString,
				Description: "ISO 8601 timestamp at which the repository was deleted " +
					"(repositories are soft deleted temporarily to allow cancelling).",
				Computed: true,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "A description of the repository's purpose/contents.",
				Computed:    true,
			},
			"identifier": {
				Type:         schema.TypeString,
				Description:  "The identifier for this repository.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"index_files": {
				Type: schema.TypeBool,
				Description: "If checked, files contained in packages will be indexed, which increase the " +
					"synchronisation time required for packages. Note that it is recommended you keep this " +
					"enabled unless the synchronisation time is significantly impacted.",
				Computed: true,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "Namespace to which this repository belongs.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"namespace_url": {
				Type:        schema.TypeString,
				Description: "API endpoint where data about this namespace can be retrieved.",
				Computed:    true,
			},
			"repository_type": {
				Type: schema.TypeString,
				Description: "The repository type changes how it is accessed and billed. Private repositories " +
					"can only be used on paid plans, but are visible only to you or authorised delegates. Public " +
					"repositories are free to use on all plans and visible to all Cloudsmith users.",
				Computed: true,
			},
			"self_html_url": {
				Type:        schema.TypeString,
				Description: "Website URL for this repository.",
				Computed:    true,
			},
			"self_url": {
				Type:        schema.TypeString,
				Description: "API endpoint where data about this repository can be retrieved.",
				Computed:    true,
			},
			"slug": {
				Type:        schema.TypeString,
				Description: "The slug identifies the repository in URIs.",
				Computed:    true,
			},
			"slug_perm": {
				Type: schema.TypeString,
				Description: "The slug_perm immutably identifies the repository. " +
					"It will never change once a repository has been created.",
				Computed: true,
			},
			"storage_region": {
				Type:        schema.TypeString,
				Description: "The Cloudsmith region in which package files are stored.",
				Computed:    true,
			},
		},
	}
}
