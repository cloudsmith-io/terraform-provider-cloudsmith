package cloudsmith

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPackageDownload_data(t *testing.T) {
	t.Parallel()

	namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
	apiKey := os.Getenv("CLOUDSMITH_API_KEY")
	userName := "token"
	repoName := "terraform-download-test"
	repoConfig := testAccDataSourcePackageDownload_repositoryConfig(repoName, namespace)
	fileName := "hello.txt"
	packageName := "hello"
	packageVersion := "1.0"
	destinationPath := "."

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: repoConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "name", repoName),
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "namespace", namespace),
				),
			},
			{
				PreConfig: func() {
					createHelloWorldFile(fileName)
					uploadPackageToCloudsmith(apiKey, namespace, repoName, fileName, packageName, packageVersion, userName)
				},
				Config: testAccPackageDownloadData(namespace, repoName, packageName, packageVersion, destinationPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "namespace", namespace),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "repository", repoName),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "package_name", packageName),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "package_version", packageVersion),
					resource.TestCheckResourceAttr("data.cloudsmith_package_download.test", "destination_path", destinationPath),
					checkDownloadedFile(namespace, repoName, packageName, packageVersion, destinationPath, "Hello world"),
				),
			},
		},
	})
}

func testAccDataSourcePackageDownload_repositoryConfig(repoName string, namespace string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "test" {
	name      = "%s"
	namespace = "%s"
}
`, repoName, namespace)
}

func createHelloWorldFile(filename string) {
	content := []byte("Hello world")
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
	uploadURL := fmt.Sprintf("https://upload.cloudsmith.io/%s/%s/%s", namespace, repoName, fileName)
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
	createPackageURL := fmt.Sprintf("https://api-prd.cloudsmith.io/v1/packages/%s/%s/upload/raw/", namespace, repoName)
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
	time.Sleep(50 * time.Second)
}

func checkDownloadedFile(namespace, repository, packageName, packageVersion, destinationPath, expectedContent string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		filename := fmt.Sprintf("%s.txt", packageName)
		filepath := filepath.Join(destinationPath, filename)

		// Check if file exists
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filepath)
		}

		// Read the file content
		content, err := os.ReadFile(filepath)
		if err != nil {
			return fmt.Errorf("error reading file: %s", err.Error())
		}

		// Check if the content matches the expected content
		if string(content) != expectedContent {
			return fmt.Errorf("file content does not match expected content")
		}

		return nil
	}
}

func testAccPackageDownloadData(namespace, repository, packageName, packageVersion, destinationPath string) string {
	return fmt.Sprintf(`
data "cloudsmith_package_download" "test" {
	namespace        = "%s"
	repository       = "%s"
	package_name     = "%s"
	package_version  = "%s"
	destination_path = "%s"
}
`, namespace, repository, packageName, packageVersion, destinationPath)
}
