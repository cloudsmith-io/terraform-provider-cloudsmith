package cloudsmith

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	"github.com/cloudsmith-io/cloudsmith-api-go"
)

func dataSourcePackagesRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := d.Get("namespace").(string)
	repository := d.Get("repository").(string)
	optional := cloudsmith.PackagesListOpts{}

	packagesList, _, err := pc.APIClient.PackagesApi.PackagesList(pc.Auth, namespace, repository, &optional)
	if err != nil {
		return err
	}

	packages := flattenPackages(&packagesList)
	if err := d.Set("package", packages); err != nil {
		return err
	}

	return nil
}

func flattenPackages(packages *[]cloudsmith.Package) []interface{} {
	if packages != nil {
		pkgs := make([]interface{}, len(*packages), len(*packages))
		for i, packageItem := range *packages {
			pkg := make(map[string]interface{})
			pkg["repository"] = packageItem.Repository
			pkg["namespace"] = packageItem.Namespace
			pkg["name"] = packageItem.Name
			pkg["slug"] = packageItem.Slug
			pkg["slug_perm"] = packageItem.SlugPerm
			pkg["format"] = packageItem.Format
			pkg["version"] = packageItem.Version
			pkg["is_sync_awaiting"] = packageItem.IsSyncAwaiting
			pkg["is_sync_completed"] = packageItem.IsSyncCompleted
			pkg["is_sync_failed"] = packageItem.IsSyncFailed
			pkg["is_sync_in_progress"] = packageItem.IsSyncInProgress
			pkg["is_sync_in_flight"] = packageItem.IsSyncInFlight
			pkgs[i] = pkg
		}

		return pkgs
	}
	return make([]interface{}, 0)
}

func dataSourcePackages() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePackagesRead,

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:         schema.TypeString,
				Description:  "The repository of the package",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "The namespace of the package",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"package": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"repository": {
							Type:        schema.TypeString,
							Description: "The repository of the package",
							Computed:    true,
						},
						"namespace": {
							Type:        schema.TypeString,
							Description: "The namespace of the package",
							Computed:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Description: "A descriptive name for the package.",
							Computed:    true,
						},
						"slug": {
							Type:        schema.TypeString,
							Description: "The slug identifies the package in URIs.",
							Computed:    true,
						},
						"slug_perm": {
							Type: schema.TypeString,
							Description: "The slug_perm immutably identifies the package. " +
								"It will never change once a package has been created.",
							Computed: true,
						},
						"format": {
							Type:        schema.TypeString,
							Description: "The format of the package",
							Computed:    true,
						},
						"version": {
							Type:        schema.TypeString,
							Description: "The version of the package",
							Computed:    true,
						},
						"is_sync_awaiting": {
							Type:        schema.TypeBool,
							Description: "Is the package awaiting synchronisation",
							Computed:    true,
						},
						"is_sync_completed": {
							Type:        schema.TypeBool,
							Description: "Has the package synchronisation completed",
							Computed:    true,
						},
						"is_sync_failed": {
							Type:        schema.TypeBool,
							Description: "Has the package synchronisation failed",
							Computed:    true,
						},
						"is_sync_in_progress": {
							Type:        schema.TypeBool,
							Description: "Is the package synchronisation currently in-progress",
							Computed:    true,
						},
						"is_sync_in_flight": {
							Type:        schema.TypeBool,
							Description: "Is the package synchronisation currently in-flight",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
