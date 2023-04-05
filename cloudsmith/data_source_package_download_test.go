package cloudsmith

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudsmith-io/cloudsmith-api-go"
)

func TestDownloadPackage(t *testing.T) {
	// Read environment variables
	namespace := os.Getenv("CLOUDSMITH_NAMESPACE")
	repository := os.Getenv("CLOUDSMITH_REPOSITORY")
	packageName := os.Getenv("CLOUDSMITH_PACKAGE_NAME")
	packageVersion := os.Getenv("CLOUDSMITH_PACKAGE_VERSION")
	destinationPath := os.Getenv("CLOUDSMITH_DESTINATION_PATH")
	apiKey := os.Getenv("CLOUDSMITH_API_KEY")

	// Initialize provider configuration
	pc := providerConfig{
		APIClient: cloudsmith.NewAPIClient(cloudsmith.NewConfiguration()),
		Auth:      context.Background(),
	}
	pc.Auth = context.WithValue(pc.Auth, cloudsmith.ContextAPIKeys, map[string]cloudsmith.APIKey{
		"apikey": {Key: apiKey},
	})

	// Get package URL
	cdnURL, filename, err := getPackageURL(&pc, namespace, repository, "", packageName, packageVersion)
	if err != nil {
		t.Fatalf("Error getting package URL: %s", err)
	}

	// Use the filename variable when constructing the destination file path
	destinationFilepath := filepath.Join(destinationPath, filename)

	// Download the file
	err = downloadFile(destinationFilepath, cdnURL, apiKey)
	if err != nil {
		t.Fatalf("Error downloading package: %s", err)
	}

	fmt.Printf("Package %s has been downloaded to %s\n", packageName, destinationFilepath)
}
