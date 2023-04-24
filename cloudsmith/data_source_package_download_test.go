package cloudsmith

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	apiBaseURL        = "https://api.cloudsmith.io/v1"
	uploadBaseURL     = "https://upload.cloudsmith.io"
	apiPackageBaseURL = "https://api-prd.cloudsmith.io/v1"
)

func TestAccPackageDownload_data(t *testing.T) {
	t.Parallel()

	namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
	apiKey := os.Getenv("CLOUDSMITH_API_KEY")
	userName := "token"
	repoName := "terraform-acc-download"
	fileName := "hello.txt"
	packageName := "hello"
	packageVersion := "1.0"
	destinationPath := "."

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					repoSlug := createRepository(apiKey, namespace, repoName, userName)
					createHelloWorldFile(fileName, packageVersion)
					uploadPackageToCloudsmith(apiKey, namespace, repoName, fileName, packageName, packageVersion, userName)
					slugPerm := getPackageSlugPerm(apiKey, namespace, repoName, fileName, userName)
					client := &http.Client{}
					waitForPackageSync(client, apiKey, namespace, repoName, slugPerm, userName)

					defer deleteRepository(apiKey, namespace, repoSlug, userName) // Delete repository after test
				},
				Config: testAccPackageDownloadData(namespace, repoName, packageName, packageVersion, destinationPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "organization", namespace),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "repository", repoName),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "package_name", packageName),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "destination_path", destinationPath),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "query", fmt.Sprintf("version:%s", packageVersion)),
				),
			},
		},
	})
}

func createRepository(apiKey, namespace, repoName, userName string) string {
	client := &http.Client{}

	createRepoURL := fmt.Sprintf("%s/repos/%s/", apiBaseURL, namespace)
	repoDetails := map[string]interface{}{
		"name":                repoName,
		"repository_type_str": "Private",
		"content_kind":        "Standard",
		"copy_packages":       "Read",
		"default_privilege":   "None",
		"delete_packages":     "Admin",
		"move_packages":       "Admin",
		"replace_packages":    "Write",
		"resync_packages":     "Admin",
		"scan_packages":       "Read",
		"storage_region":      "default",
		"view_statistics":     "Read",
	}

	repoDetailsJSON, err := json.Marshal(repoDetails)
	if err != nil {
		panic(err)
	}

	request, err := http.NewRequest("POST", createRepoURL, bytes.NewReader(repoDetailsJSON))
	if err != nil {
		panic(err)
	}
	request.SetBasicAuth(userName, apiKey)

	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		panic(fmt.Errorf("failed to create repository: %s", response.Status))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	var repoResponse map[string]interface{}
	err = json.Unmarshal(body, &repoResponse)
	if err != nil {
		panic(err)
	}

	return repoResponse["slug_perm"].(string)
}

func deleteRepository(apiKey, namespace, repoSlug, userName string) {
	client := &http.Client{}

	deleteRepoURL := fmt.Sprintf("%s/repos/%s/%s/", apiBaseURL, namespace, repoSlug)
	request, err := http.NewRequest("DELETE", deleteRepoURL, nil)
	if err != nil {
		panic(err)
	}
	request.SetBasicAuth(userName, apiKey)

	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		panic(fmt.Errorf("failed to delete repository: %s", response.Status))
	}
}

func createHelloWorldFile(filename string, fileVersion string) {
	content := []byte(fmt.Sprintf("Hello world v%s", fileVersion))
	err := os.WriteFile(filename, content, 0644)
	if err != nil {
		panic(err)
	}
}

func uploadPackageToCloudsmith(apiKey, namespace, repoName, fileName, packageName, packageVersion string, userName string) {
	client := &http.Client{}

	// Calculate the SHA-256 checksum of the file
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}
	sha256Checksum := fmt.Sprintf("%x", sha256.Sum256(fileData))

	// Step 1: PUT (upload) the file
	uploadURL := fmt.Sprintf("%s/%s/%s/%s", uploadBaseURL, namespace, repoName, fileName)
	request, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(fileData))
	if err != nil {
		panic(err)
	}
	request.SetBasicAuth(userName, apiKey)
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("Content-Sha256", sha256Checksum)

	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		panic(fmt.Errorf("failed to upload package: %s", response.Status))
	}

	// Read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	// Parse the response JSON
	var uploadResponse map[string]string
	err = json.Unmarshal(body, &uploadResponse)
	if err != nil {
		panic(err)
	}
	identifier := uploadResponse["identifier"]

	// Step 2: POST the package details
	createPackageURL := fmt.Sprintf("%s/packages/%s/%s/upload/raw/", apiPackageBaseURL, namespace, repoName)
	packageDetails := map[string]string{
		"package_file": identifier,
		"name":         packageName,
		"description":  "Test package for Terraform provider",
		"summary":      "Test Package",
		"version":      packageVersion,
	}
	packageDetailsJSON, err := json.Marshal(packageDetails)
	if err != nil {
		panic(err)
	}

	request, err = http.NewRequest("POST", createPackageURL, bytes.NewReader(packageDetailsJSON))
	if err != nil {
		panic(err)
	}
	request.SetBasicAuth(userName, apiKey)

	request.Header.Set("Content-Type", "application/json")

	response, err = client.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		panic(fmt.Errorf("failed to create package: %s", response.Status))
	}
	err = os.Remove(fileName)
	if err != nil {
		panic(err)
	}
}

func waitForPackageSync(client *http.Client, apiKey, namespace, repoName, identifier, userName string) {
	statusURL := fmt.Sprintf("%s/packages/%s/%s/%s/status/", apiBaseURL, namespace, repoName, identifier)
	request, err := http.NewRequest("GET", statusURL, nil)
	if err != nil {
		panic(err)
	}
	request.SetBasicAuth(userName, apiKey)

	for {
		response, err := client.Do(request)
		if err != nil {
			panic(err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			panic(fmt.Errorf("failed to get package status: %s", response.Status))
		}

		body, err := io.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}

		var statusResponse map[string]interface{}
		err = json.Unmarshal(body, &statusResponse)
		if err != nil {
			panic(err)
		}

		stageStr, ok := statusResponse["stage_str"].(string)
		if ok && stageStr == "Fully Synchronised" {
			break
		}

		time.Sleep(5 * time.Second)
	}
}

func getPackageSlugPerm(apiKey, namespace, repoName, fileName, userName string) string {
	client := &http.Client{}

	// Prepare API request
	url := fmt.Sprintf("%s/packages/%s/%s/?page_size=1&query=filename:%s", apiBaseURL, namespace, repoName, fileName)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	request.SetBasicAuth(userName, apiKey)

	// Send API request
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		panic(fmt.Errorf("failed to get package list: %s", response.Status))
	}

	// Read and parse API response
	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	var packages []map[string]interface{}
	err = json.Unmarshal(body, &packages)
	if err != nil {
		panic(err)
	}

	if len(packages) == 0 {
		panic(fmt.Errorf("no package found with filename: %s", fileName))
	}

	slugPerm, ok := packages[0]["slug_perm"].(string)
	if !ok {
		panic(fmt.Errorf("failed to get slug_perm from package: %v", packages[0]))
	}

	return slugPerm
}

func testAccPackageDownloadData(namespace, repository, packageName, packageVersion, destinationPath string) string {
	return fmt.Sprintf(`
data "cloudsmith_package_download" "test" {
	organization     = "%s"
	repository       = "%s"
	package_name     = "%s"
	query            = "version:%s"
	destination_path = "%s"
}
`, namespace, repository, packageName, packageVersion, destinationPath)
}
