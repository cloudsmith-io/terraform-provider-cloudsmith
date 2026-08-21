package cloudsmith

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	dsPackageTestNamespace = os.Getenv("CLOUDSMITH_NAMESPACE")
)

func TestDownloadPackageUsesOIDCExchangeToken(t *testing.T) {
	t.Parallel()

	const (
		assertion      = "tfc-workload-identity-token"
		exchangedToken = "cloudsmith-oidc-token"
		packageBody    = "package contents"
	)

	var exchanges atomic.Int32
	authorization := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/openid/test-org/":
			exchanges.Add(1)
			var payload struct {
				OIDCToken   string `json:"oidc_token"`
				ServiceSlug string `json:"service_slug"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if payload.OIDCToken != assertion || payload.ServiceSlug != "terraform-cloud" {
				http.Error(w, "unexpected exchange payload", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"token":%q}`, exchangedToken)
		case r.Method == http.MethodGet && (strings.TrimSuffix(r.URL.Path, "/") == "/user/self" || strings.TrimSuffix(r.URL.Path, "/") == "/v1/user/self"):
			if r.Header.Get("X-Api-Key") != exchangedToken {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"email":"sa@example.com","name":"tfc","slug":"tfc","slug_perm":"tfc"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/package.tar.gz":
			authorization <- r.Header.Get("Authorization")
			fmt.Fprint(w, packageBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	tokens := &oidcTokenSource{
		identity: oidcIdentity{organization: "test-org", serviceSlug: "terraform-cloud"},
		apiHost:  server.URL,
		getenv: func(key string) string {
			if key == "TFC_WORKLOAD_IDENTITY_TOKEN" {
				return assertion
			}
			return ""
		},
		now: time.Now,
	}
	config, diags := newProviderConfig(context.Background(), server.URL, tokens, nil, "test-agent")
	if diags.HasError() {
		t.Fatalf("configure provider: %v", diags)
	}

	downloaded, err := downloadPackage(server.URL+"/package.tar.gz", t.TempDir(), config, false)
	if err != nil {
		t.Fatalf("download package: %v", err)
	}
	if got := <-authorization; got != "Token "+exchangedToken {
		t.Fatalf("Authorization = %q, want %q", got, "Token "+exchangedToken)
	}
	if got := exchanges.Load(); got != 1 {
		t.Fatalf("OIDC exchanges = %d, want 1", got)
	}
	contents, err := os.ReadFile(downloaded)
	if err != nil {
		t.Fatalf("read downloaded package: %v", err)
	}
	if got := string(contents); got != packageBody {
		t.Fatalf("downloaded contents = %q, want %q", got, packageBody)
	}
}

func TestAccPackage_data(t *testing.T) {
	t.Parallel()

	repositoryName := testAccUniqueRepositoryName("terraform-acc-test-package")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPackageDataSetup(repositoryName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudsmith_repository.test", "name", repositoryName),
					// Custom TestCheckFunc to upload the package and wait for sync after repository creation
					func(s *terraform.State) error {
						return uploadPackage(testAccProvider.Meta().(*providerConfig), repositoryName, false)
					},
				),
			},
			{
				Config: testAccPackageDataReadPackage(dsPackageTestNamespace, repositoryName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_package.test", "namespace", dsPackageTestNamespace),
					resource.TestCheckResourceAttr("data.cloudsmith_package.test", "repository", repositoryName),
				),
			},
			{
				Config: testAccPackageDataReadPackageDownload(dsPackageTestNamespace, repositoryName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_package.test", "namespace", dsPackageTestNamespace),
					resource.TestCheckResourceAttr("data.cloudsmith_package.test", "repository", repositoryName),
					// Custom TestCheckFunc to check if the file exists at the output path
					func(s *terraform.State) error {
						filePath := filepath.Join(os.TempDir(), "hello.txt")
						if _, err := os.Stat(filePath); os.IsNotExist(err) {
							return fmt.Errorf("file does not exist at path: %s", filePath)
						}
						defer func() {
							// Remove the file after the check is done
							if err := os.Remove(filePath); err != nil {
								fmt.Printf("Error removing file: %s\n", err)
							}
						}()
						expectedContent := "Hello world"
						if err := checkFileContent(filePath, expectedContent); err != nil {
							return fmt.Errorf("file content check failed: %w", err)
						}
						return nil
					},
					func(s *terraform.State) error {
						return uploadPackage(testAccProvider.Meta().(*providerConfig), repositoryName, true)
					},
				),
			},
			{
				Config: testAccPackageDataReadPackageDownloadRepublish(dsPackageTestNamespace, repositoryName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudsmith_package.test", "namespace", dsPackageTestNamespace),
					resource.TestCheckResourceAttr("data.cloudsmith_package.test", "repository", repositoryName),
					func(s *terraform.State) error {
						filePath := filepath.Join(os.TempDir(), "hello.txt")
						if _, err := os.Stat(filePath); os.IsNotExist(err) {
							return fmt.Errorf("file does not exist at path: %s", filePath)
						}

						expectedContent := "Hello world updated content"
						if err := checkFileContent(filePath, expectedContent); err != nil {
							return fmt.Errorf("file content check failed: %w", err)
						}

						return nil
					},
				),
			},
		},
	})
}
func checkFileContent(filePath string, expectedContent string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if string(content) != expectedContent {
		return fmt.Errorf("file content does not match expected. Got: %s, Expected: %s", content, expectedContent)
	}

	return nil
}

func uploadPackage(pc *providerConfig, repository string, republish bool) error {

	var (
		fileContent []byte
	)

	if republish {
		fileContent = []byte("Hello world updated content")
	} else {
		fileContent = []byte("Hello world")
	}

	initPayload := cloudsmith.PackageFileUploadRequest{
		Filename:       "hello.txt",
		Method:         cloudsmith.PtrString("put"),
		Sha256Checksum: cloudsmith.PtrString(fmt.Sprintf("%x", sha256.Sum256(fileContent))),
	}

	initRequest := pc.APIClient.FilesApi.FilesCreate(pc.Auth, dsPackageTestNamespace, repository)
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

	apiKey, err := pc.GetAPIKey()
	if err != nil {
		return err
	}
	request.SetBasicAuth("token", apiKey)
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

	finalizeRequest := pc.APIClient.PackagesApi.PackagesUploadRaw(pc.Auth, dsPackageTestNamespace, repository)
	finalizeRequest = finalizeRequest.Data(finalizePayload)
	finalizeResponse, _, err := finalizeRequest.Execute()
	if err != nil {
		return fmt.Errorf("failed to finalize file upload: %w", err)
	}

	// Step 3: wait for package sync
	for {
		statusRequest := pc.APIClient.PackagesApi.PackagesStatus(
			pc.Auth, dsPackageTestNamespace, repository, finalizeResponse.GetSlugPerm(),
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

func testAccPackageDataSetup(repository string) string {
	return fmt.Sprintf(`
		resource "cloudsmith_repository" "test" {
			name                        = "%s"
			namespace                   = "%s"
			replace_packages_by_default = true
		}
		`, repository, dsPackageTestNamespace)
}

func testAccPackageDataReadPackage(namespace, repository string) string {
	return fmt.Sprintf(`
		resource "cloudsmith_repository" "test" {
			name                        = "%s"
			namespace                   = "%s"
			replace_packages_by_default = true
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
			name                        = "%s"
			namespace                   = "%s"
			replace_packages_by_default = true
		}

		data "cloudsmith_package_list" "test" {
			repository = "%s"
			namespace  = "%s"
		}

		data "cloudsmith_package" "test" {
			repository = "%s"
			namespace  = "%s"
			identifier = data.cloudsmith_package_list.test.packages[0].slug_perm
			download   = true
		}
		`, repository, namespace, repository, namespace, repository, namespace)
}

func testAccPackageDataReadPackageDownloadRepublish(namespace, repository string) string {
	return fmt.Sprintf(`
		resource "cloudsmith_repository" "test" {
			name                        = "%s"
			namespace                   = "%s"
			replace_packages_by_default = true
		}

		data "cloudsmith_package_list" "test" {
			repository = "%s"
			namespace  = "%s"
		}

		data "cloudsmith_package" "test" {
			repository = "%s"
			namespace  = "%s"
			identifier = data.cloudsmith_package_list.test.packages[0].slug_perm
			download   = true
		}
		`, repository, namespace, repository, namespace, repository, namespace)
}
