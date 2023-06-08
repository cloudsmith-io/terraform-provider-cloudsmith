package cloudsmith

import (
	"fmt"
	"io"
	"os"
	"path"

	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePackageRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")
	identifier := requiredString(d, "identifier")
	download := requiredBool(d, "download")
	outputPath := requiredString(d, "output_path")

	req := pc.APIClient.PackagesApi.PackagesRead(pc.Auth, namespace, repository, identifier)
	pkg, _, err := pc.APIClient.PackagesApi.PackagesReadExecute(req)
	if err != nil {
		return err
	}

	d.Set("cdn_url", pkg.GetCdnUrl())
	d.Set("format", pkg.GetFormat())
	d.Set("is_sync_awaiting", pkg.GetIsSyncAwaiting())
	d.Set("is_sync_completed", pkg.GetIsSyncCompleted())
	d.Set("is_sync_failed", pkg.GetIsSyncFailed())
	d.Set("is_sync_in_flight", pkg.GetIsSyncInFlight())
	d.Set("is_sync_in_progress", pkg.GetIsSyncInProgress())
	d.Set("name", pkg.GetName())
	d.Set("slug", pkg.GetSlug())
	d.Set("slug_perm", pkg.GetSlugPerm())
	d.Set("version", pkg.GetVersion())

	d.SetId(fmt.Sprintf("%s_%s_%s", namespace, repository, pkg.GetSlugPerm()))

	var computedOutputPath string

	if download {
		err := downloadPackage(pkg.GetCdnUrl(), outputPath, pc.GetAPIKey())
		if err != nil {
			return err
		}

		computedOutputPath = outputPath
	} else {
		computedOutputPath = pkg.GetCdnUrl()
	}

	d.Set("output_path", computedOutputPath)

	return nil
}

func downloadPackage(url string, outputPath string, apiKey string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s, status code: %d", url, resp.StatusCode)
	}

	// Extract filename from CDN URL
	filename := path.Base(url)
	outputPath = path.Join(outputPath, filename)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func dataSourcePackage() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePackageRead,

		Schema: map[string]*schema.Schema{
			"cdn_url": {
				Type:        schema.TypeString,
				Description: "The URL of the package to download.",
				Computed:    true,
			},
			"download": {
				Type:        schema.TypeBool,
				Description: "If set to true, download the package",
				Optional:    true,
				Default:     false,
			},
			"format": {
				Type:        schema.TypeString,
				Description: "The format of the package",
				Computed:    true,
			},
			"identifier": {
				Type:         schema.TypeString,
				Description:  "The identifier for this repository.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
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
			"is_sync_in_flight": {
				Type:        schema.TypeBool,
				Description: "Is the package synchronisation currently in-flight",
				Computed:    true,
			},
			"is_sync_in_progress": {
				Type:        schema.TypeBool,
				Description: "Is the package synchronisation currently in-progress",
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "A descriptive name for the package.",
				Computed:    true,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "The namespace of the package",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"output_path": {
				Type:        schema.TypeString,
				Description: "The path to save the downloaded package",
				Optional:    true,
				Default:     os.TempDir(),
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
			"repository": {
				Type:         schema.TypeString,
				Description:  "The repository of the package",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"version": {
				Type:        schema.TypeString,
				Description: "The version of the package",
				Computed:    true,
			},
		},
	}
}
