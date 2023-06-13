package cloudsmith

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePackageRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")
	identifier := requiredString(d, "identifier")
	download := requiredBool(d, "download")
	downloadDir := requiredString(d, "download_dir")

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

	if download {
		outputPath, err := downloadPackage(pkg.GetCdnUrl(), downloadDir, pc)
		if err != nil {
			return err
		}
		d.Set("output_path", outputPath)
		d.Set("output_directory", downloadDir)
	} else {
		d.Set("output_path", pkg.GetCdnUrl())
		d.Set("output_directory", "")
	}

	return nil
}

func downloadPackage(url string, downloadDir string, pc *providerConfig) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", pc.GetAPIKey()))

	client := pc.APIClient.GetConfig().HTTPClient
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file: %s, status code: %d", url, resp.StatusCode)
	}

	// Extract filename from CDN URL
	filename := path.Base(url)
	outputPath := path.Join(downloadDir, filename)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, resp.Body)
	if err != nil {
		return "", err
	}

	return outputPath, nil
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
				Description: "The location of the package",
				Computed:    true,
			},
			"download_dir": {
				Type:        schema.TypeString,
				Description: "The directory where the file will be downloaded if download is set to true",
				Optional:    true,
				Default:     os.TempDir(),
			},
			"output_directory": {
				Type:        schema.TypeString,
				Description: "The directory where the file is downloaded",
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
