package cloudsmith

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	dsPackageTestNamespace  = os.Getenv("CLOUDSMITH_NAMESPACE")
	dsPackageTestRepository = "terraform-acc-test-package"
)

func TestAccPackage_data(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPackageDataSetup(dsPackageTestNamespace, dsPackageTestRepository),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "name", dsPackageTestRepository),
					// Custom TestCheckFunc to upload the package and wait for sync after repository creation
					func(s *terraform.State) error {
						return uploadPackage(testAccProvider.Meta().(*providerConfig))
					},
				),
			},
			{
				Config: testAccPackageDataReadPackage(dsPackageTestNamespace, dsPackageTestRepository),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_package.test", "namespace", dsPackageTestNamespace),
					resource.TestCheckResourceAttr("data.cloudsmith_package.test", "repository", dsPackageTestRepository),
				),
			},
			{
				Config: testAccPackageDataReadPackageDownload(dsPackageTestNamespace, dsPackageTestRepository),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_package.test", "namespace", dsPackageTestNamespace),
					resource.TestCheckResourceAttr("data.cloudsmith_package.test", "repository", dsPackageTestRepository),
					// Custom TestCheckFunc to check if the file exists at the output path
					func(s *terraform.State) error {
						filePath := "hello.txt" // Replace with the output path used in the test
						if _, err := os.Stat(filePath); os.IsNotExist(err) {
							return fmt.Errorf("file does not exist at path: %s", filePath)
						}
						return nil
					},
				),
			},
		},
	})
}

func uploadPackage(pc *providerConfig) error {
	fileContent := []byte("Hello world")

	initPayload := cloudsmith.PackageFileUploadRequest{
		Filename:       "hello.txt",
		Method:         cloudsmith.PtrString("put"),
		Sha256Checksum: cloudsmith.PtrString(fmt.Sprintf("%x", sha256.Sum256(fileContent))),
	}

	initRequest := pc.APIClient.FilesApi.FilesCreate(pc.Auth, dsPackageTestNamespace, dsPackageTestRepository)
	initRequest = initRequest.Data(initPayload)
	initResponse, _, err := initRequest.Execute()
	if err != nil {
		return fmt.Errorf("failed to initialize file upload: %w", err)
	}

	// Step 1: PUT (upload) the file
	request, err := http.NewRequest("PUT", initResponse.GetUploadUrl(), bytes.NewReader(fileContent))
	if err != nil {
		return err
	}

	request.SetBasicAuth("token", pc.GetAPIKey())
	for k, v := range initResponse.GetUploadHeaders() {
		request.Header.Set(k, v.(string))
	}

	response, err := pc.APIClient.GetConfig().HTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.New("Unable to upload file")
	}

	var rbodyStruct struct {
		Identifier string `json:"identifier"`
	}

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, response.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(buf.Bytes(), &rbodyStruct); err != nil {
		return err
	}

	// Step 2: finalize file upload and kick off package sync
	finalizePayload := cloudsmith.RawPackageUploadRequest{
		PackageFile: rbodyStruct.Identifier,
	}

	finalizeRequest := pc.APIClient.PackagesApi.PackagesUploadRaw(pc.Auth, dsPackageTestNamespace, dsPackageTestRepository)
	finalizeRequest = finalizeRequest.Data(finalizePayload)
	finalizeResponse, _, err := finalizeRequest.Execute()
	if err != nil {
		return fmt.Errorf("failed to finalize file upload: %w", err)
	}

	// Step 3: wait for package sync
	for {
		statusRequest := pc.APIClient.PackagesApi.PackagesStatus(
			pc.Auth, dsPackageTestNamespace, dsPackageTestRepository, finalizeResponse.GetSlugPerm(),
		)
		status, _, err := statusRequest.Execute()
		if err != nil {
			return err
		}
		if status.GetIsSyncFailed() {
			return errors.New("package sync failed")
		}
		if status.GetIsSyncCompleted() {
			return nil
		}

		time.Sleep(5 * time.Second)
	}
}

func testAccPackageDataSetup(namespace, repository string) string {
	return fmt.Sprintf(`
		resource "cloudsmith_repository" "test" {
			name      = "%s"
			namespace = "%s"
		}
		`, repository, namespace)
}

func testAccPackageDataReadPackage(namespace, repository string) string {
	return fmt.Sprintf(`
		resource "cloudsmith_repository" "test" {
			name      = "%s"
			namespace = "%s"
		}

		data "cloudsmith_package_list" "test" {
			repository       = "%s"
			namespace        = "%s"
		}

		data "cloudsmith_package" "test" {
			repository       = "%s"
			namespace        = "%s"
			identifier       = data.cloudsmith_package_list.test.packages[0].slug_perm
		}
		`, repository, namespace, repository, namespace, repository, namespace)
}

func testAccPackageDataReadPackageDownload(namespace, repository string) string {
	return fmt.Sprintf(`
		resource "cloudsmith_repository" "test" {
			name      = "%s"
			namespace = "%s"
		}

		data "cloudsmith_package_list" "test" {
			repository       = "%s"
			namespace        = "%s"
		}

		data "cloudsmith_package" "test" {
			repository       = "%s"
			namespace        = "%s"
			identifier       = data.cloudsmith_package_list.test.packages[0].slug_perm
			download 		 = true
			output_path      = "."
		}
		`, repository, namespace, repository, namespace, repository, namespace)
}
