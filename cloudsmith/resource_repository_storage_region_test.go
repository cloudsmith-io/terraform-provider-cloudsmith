//nolint:testpackage
package cloudsmith

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cloudsmithapi "github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestResourceRepositoryStorageRegionUpdate_UsesResourceID(t *testing.T) {
	t.Parallel()

	var (
		requestPath string
		requestBody string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		requestBody = string(body)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := cloudsmithapi.NewConfiguration()
	config.Servers = cloudsmithapi.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	pc := &providerConfig{
		APIClient: cloudsmithapi.NewAPIClient(config),
		Auth: context.WithValue(
			context.Background(),
			cloudsmithapi.ContextAPIKeys,
			map[string]cloudsmithapi.APIKey{
				"apikey": {Key: "test-api-key"},
			},
		),
	}

	rd := schema.TestResourceDataRaw(t, resourceRepository().Schema, map[string]interface{}{
		"namespace":      "example-org",
		"name":           "display-name-should-not-be-used",
		"storage_region": "us-ohio",
	})
	rd.SetId("repo-id-fixture")

	if err := resourceRepositoryStorageRegionUpdate(rd, pc); err != nil {
		t.Fatalf("resourceRepositoryStorageRegionUpdate() error = %v", err)
	}

	expectedPath := "/repos/example-org/repo-id-fixture/transfer-region/"
	if requestPath != expectedPath {
		t.Fatalf("unexpected request path %q, expected %q", requestPath, expectedPath)
	}
	if !strings.Contains(requestBody, `"storage_region":"us-ohio"`) {
		t.Fatalf("unexpected request body %q", requestBody)
	}
}
