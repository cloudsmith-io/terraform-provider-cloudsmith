package cloudsmith

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// getPackageURL calls the API with function PackagesListExecute
func getPackageURL(pc *providerConfig, organization string, repository string, query string, packageName string) (string, string, string, error) {
	req := pc.APIClient.PackagesApi.PackagesList(pc.Auth, organization, repository)

	req = req.Query(query)
	req = req.PageSize(1)

	packagesList, _, err := pc.APIClient.PackagesApi.PackagesListExecute(req)
	if err != nil {
		return "", "", "", err
	}

	if len(packagesList) > 0 && packagesList[0].GetName() == packageName {
		return packagesList[0].GetCdnUrl(), packagesList[0].GetFilename(), packagesList[0].GetSlugPerm(), nil
	}

	return "", "", "", errors.New("package not found")
}

func dataSourcePackageDownloadRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := d.Get("organization").(string)
	repository := d.Get("repository").(string)
	query := d.Get("query").(string)
	packageName := d.Get("package_name").(string)
	destinationPath := d.Get("destination_path").(string)

	// Add this check for destinationPath
	if destinationPath == "" {
		return errors.New("destination_path must be provided")
	}

	cdnURL, filename, slugPerm, err := getPackageURL(pc, organization, repository, query, packageName)
	if err != nil {
		return err
	}

	apiKey := pc.GetAPIKey()

	// Use the filename variable when constructing the destination file path
	destinationFilepath := filepath.Join(destinationPath, filename)

	// Download the file to the destination path
	err = downloadFile(destinationFilepath, cdnURL, apiKey)
	if err != nil {
		return err
	}

	d.SetId(slugPerm)
	return nil
}

func downloadFile(filepath string, url string, apiKey string) error {
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

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func dataSourcePackageDownload() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePackageDownloadRead,

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:         schema.TypeString,
				Description:  "The repository to which the package belongs.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"organization": {
				Type:         schema.TypeString,
				Description:  "The organization to which the package belongs.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"query": {
				Type:        schema.TypeString,
				Description: "The query to filter packages.",
				Optional:    true,
				Default:     "",
			},
			"package_name": {
				Type:         schema.TypeString,
				Description:  "The name of the package to download.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"cdn_url": {
				Type:        schema.TypeString,
				Description: "The CDN URL of the package to download.",
				Computed:    true,
			},
			"destination_path": {
				Type:         schema.TypeString,
				Description:  "The path to download the package to.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}
