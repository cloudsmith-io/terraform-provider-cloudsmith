package cloudsmith

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/cloudsmith-io/cloudsmith-api-go"
)

func retrievePackageListPage(pc *providerConfig, namespace string, repository string, query string, pageSize int64, pageCount int64) ([]cloudsmith.PackageResponse, int64, error) {
	req := pc.APIClient.PackagesApi.PackagesList(pc.Auth, namespace, repository)
	req = req.Page(pageCount)
	req = req.PageSize(pageSize)
	req = req.Query(query)

	packagesPage, httpResponse, err := pc.APIClient.PackagesApi.PackagesListExecute(req)
	if err != nil {
		return nil, 0, err
	}
	pageTotal, err := strconv.ParseInt(httpResponse.Header.Get("X-Pagination-Pagetotal"), 10, 64)
	if err != nil {
		return nil, 0, err
	}
	return packagesPage, pageTotal, nil
}

func retrievePackageListPages(pc *providerConfig, namespace string, repository string, query string, pageSize int64, pageCount int64) ([]cloudsmith.PackageResponse, error) {

	var pageCurrentCount int64 = 1

	// A negative or zero count is assumed to mean retrieve the largest size page
	packagesList := []cloudsmith.PackageResponse{}
	if pageSize == -1 || pageSize == 0 {
		pageSize = 100
	}

	// If no count is supplied assmumed to mean retrieve all pages
	// we have to retreive a page to get this count
	if pageCount == -1 || pageCount == 0 {
		var packagesPage []cloudsmith.PackageResponse
		var err error
		packagesPage, pageCount, err = retrievePackageListPage(pc, namespace, repository, query, pageSize, 1)
		if err != nil {
			return nil, err
		}
		packagesList = append(packagesList, packagesPage...)
		pageCurrentCount++
	}

	for pageCurrentCount <= pageCount {
		packagesPage, _, err := retrievePackageListPage(pc, namespace, repository, query, pageSize, pageCount)
		if err != nil {
			return nil, err
		}
		packagesList = append(packagesList, packagesPage...)
		pageCurrentCount++

	}
	return packagesList, nil
}

func buildQueryString(set *schema.Set) string {
	var query strings.Builder
	for _, v := range set.List() {
		query.WriteString(v.(string))
		query.WriteString(" ")
	}
	return query.String()
}

func dataSourcePackageListRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")
	query := buildQueryString(d.Get("filters").(*schema.Set))
	mostRecent := requiredBool(d, "most_recent")

	var pageCount, pageSize int64 = -1, -1
	if mostRecent {
		pageCount = 1
		pageSize = 1
	}
	packagesList, err := retrievePackageListPages(pc, namespace, repository, query, pageSize, pageCount)
	if err != nil {
		return err
	}
	packages := flattenPackages(packagesList)
	if err := d.Set("packages", packages); err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}

func flattenPackages(packages []cloudsmith.PackageResponse) []interface{} {
	pkgs := make([]interface{}, len(packages))
	for i, packageItem := range packages {
		log.Printf("[DEBUG] package: %s", packageItem.GetName())
		pkg := make(map[string]interface{})
		pkg["repository"] = packageItem.GetRepository()
		pkg["namespace"] = packageItem.GetNamespace()
		pkg["name"] = packageItem.GetName()
		pkg["slug"] = packageItem.GetSlug()
		pkg["slug_perm"] = packageItem.GetSlugPerm()
		pkg["format"] = packageItem.GetFormat()
		pkg["version"] = packageItem.GetVersion()
		pkg["is_sync_awaiting"] = packageItem.GetIsSyncAwaiting()
		pkg["is_sync_completed"] = packageItem.GetIsSyncCompleted()
		pkg["is_sync_failed"] = packageItem.GetIsSyncFailed()
		pkg["is_sync_in_progress"] = packageItem.GetIsSyncInProgress()
		pkg["is_sync_in_flight"] = packageItem.GetIsSyncInFlight()
		pkgs[i] = pkg
	}

	return pkgs
}

func dataSourcePackageList() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePackageListRead,

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:         schema.TypeString,
				Description:  "The repository to which the packages belong.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "The namespace to which the packages belong.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"filters": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"most_recent": {
				Type:        schema.TypeBool,
				Description: "Only return the most recent package",
				Optional:    true,
			},
			"packages": {
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
