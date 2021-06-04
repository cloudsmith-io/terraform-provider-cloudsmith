package cloudsmith

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	"github.com/cloudsmith-io/cloudsmith-api-go"
)

func retrievePackagesPage(pc *providerConfig, namespace string, repository string, query string, pageSize int64, pageCount int64) ([]cloudsmith.Package, int64, error) {
	optional := cloudsmith.PackagesListOpts{Page: optional.NewInt64(pageCount), PageSize: optional.NewInt64(pageSize), Query: optional.NewString(query)}
	packagesPage, httpResponse, err := pc.APIClient.PackagesApi.PackagesList(pc.Auth, namespace, repository, &optional)
	if err != nil {
		return nil, 0, err
	}
	pageTotal, err := strconv.ParseInt(httpResponse.Header.Get("X-Pagination-Pagetotal"), 10, 64)
	if err != nil {
		return nil, 0, err
	}
	return packagesPage, pageTotal, nil
}

func retrievePackagesPages(pc *providerConfig, namespace string, repository string, query string, pageSize int64, pageCount int64) ([]cloudsmith.Package, error) {

	var pageCurrentCount int64 = 1

	// A negative or zero count is assumed to mean retrieve the largest size page
	packagesList := []cloudsmith.Package{}
	if pageSize == -1 || pageSize == 0 {
		pageSize = 100
	}

	// If no count is supplied assmumed to mean retrieve all pages
	// we have to retreive a page to get this count
	if pageCount == -1 || pageCount == 0 {
		var packagesPage []cloudsmith.Package
		var err error
		packagesPage, pageCount, err = retrievePackagesPage(pc, namespace, repository, query, pageSize, 1)
		if err != nil {
			return nil, err
		}
		packagesList = append(packagesList, packagesPage...)
		pageCurrentCount++
	}

	for pageCurrentCount <= pageCount {
		packagesPage, _, err := retrievePackagesPage(pc, namespace, repository, query, pageSize, pageCount)
		if err != nil {
			return nil, err
		}
		packagesList = append(packagesList, packagesPage...)
		pageCurrentCount++

	}
	return packagesList, nil
}

func buildQueryString(set *schema.Set, packageGroup string) string {
	var query strings.Builder
	for _, v := range set.List() {
		query.WriteString(v.(string))
		query.WriteString(" ")
	}
	if packageGroup != "" {
		query.WriteString("name:^")
		query.WriteString(packageGroup)
		query.WriteString("$")
	}
	return query.String()
}

func dataSourcePackagesRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := d.Get("namespace").(string)
	repository := d.Get("repository").(string)
	query := buildQueryString(d.Get("filters").(*schema.Set), d.Get("package_group").(string))
	mostRecent := d.Get("most_recent").(bool)
	var pageCount, pageSize int64 = -1, -1
	if mostRecent {
		pageCount = 1
		pageSize = 1
	}
	packagesList, err := retrievePackagesPages(pc, namespace, repository, query, pageSize, pageCount)
	if err != nil {
		return err
	}
	packages := flattenPackages(&packagesList)
	if err := d.Set("packages", packages); err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}

func flattenPackages(packages *[]cloudsmith.Package) []interface{} {
	if packages != nil {
		pkgs := make([]interface{}, len(*packages), len(*packages))
		for i, packageItem := range *packages {
			log.Printf("[DEBUG] package: %s", packageItem.Name)
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
			"package_group": {
				Type:        schema.TypeString,
				Description: "The namespace of the package",
				Optional:    true,
			},
			"filters": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"most_recent": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "Only return the most recent package",
				Optional:    true,
			},
			"packages": &schema.Schema{
				Type:     schema.TypeList,
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
