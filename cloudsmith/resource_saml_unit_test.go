//nolint:testpackage
package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	cloudsmithapi "github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestSamlCreateLeavesEnabledUnmanagedWhenOmitted(t *testing.T) {
	t.Parallel()

	var toggleRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/orgs/example-org/saml-group-sync/":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"idp_key":"group","idp_value":"developers","role":"Member","slug_perm":"mapping","team":"developers"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/example-org/saml-group-sync/":
			writeSAMLListResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/example-org/saml-group-sync/status/":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"saml_group_sync_status":false}`)
		case r.Method == http.MethodPost && (r.URL.Path == "/orgs/example-org/saml-group-sync/enable/" || r.URL.Path == "/orgs/example-org/saml-group-sync/disable/"):
			toggleRequests.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	pc := samlTestProviderConfig(server)
	providerFactory := func() (*schema.Provider, error) {
		samlResource := resourceSAML()
		samlResource.Delete = func(_ *schema.ResourceData, _ interface{}) error { return nil }
		return &schema.Provider{
			ResourcesMap: map[string]*schema.Resource{"cloudsmith_saml": samlResource},
			ConfigureContextFunc: func(_ context.Context, _ *schema.ResourceData) (interface{}, diag.Diagnostics) {
				return pc, nil
			},
		}, nil
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: map[string]func() (*schema.Provider, error){"cloudsmith": providerFactory},
		Steps: []resource.TestStep{{
			Config: `
resource "cloudsmith_saml" "test" {
  organization = "example-org"
  idp_key       = "group"
  idp_value     = "developers"
  role          = "Member"
  team          = "developers"
}`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("cloudsmith_saml.test", "enabled", "false"),
				func(_ *terraform.State) error {
					if got := toggleRequests.Load(); got != 0 {
						return fmt.Errorf("toggle requests = %d, want 0", got)
					}
					return nil
				},
			),
		}},
	})
}

func TestSamlReadRetriesTransientStatusNotFound(t *testing.T) {
	t.Parallel()

	statusRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/example-org/saml-group-sync/":
			writeSAMLListResponse(w)
		case "/orgs/example-org/saml-group-sync/status/":
			statusRequests++
			if statusRequests == 1 {
				http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"saml_group_sync_status":false}`)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	d := schema.TestResourceDataRaw(t, resourceSAML().Schema, map[string]interface{}{
		"organization": "example-org",
	})
	d.SetId("mapping")

	if err := samlRead(d, samlTestProviderConfig(server)); err != nil {
		t.Fatalf("samlRead() error = %v", err)
	}
	if d.Id() != "mapping" {
		t.Fatalf("samlRead() cleared mapping ID after transient status 404")
	}
	if got := d.Get("enabled"); got != false {
		t.Fatalf("enabled = %v, want false", got)
	}
	if statusRequests != 2 {
		t.Fatalf("status requests = %d, want 2", statusRequests)
	}
}

func TestSamlSetEnabledRetriesTransientStatusNotFound(t *testing.T) {
	t.Parallel()

	var statusRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/orgs/example-org/saml-group-sync/enable/":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/orgs/example-org/saml-group-sync/status/":
			if statusRequests.Add(1) == 1 {
				http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"saml_group_sync_status":true}`)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	d := schema.TestResourceDataRaw(t, resourceSAML().Schema, map[string]interface{}{
		"organization": "example-org",
		"enabled":      true,
	})

	if err := samlSetEnabled(d, samlTestProviderConfig(server)); err != nil {
		t.Fatalf("samlSetEnabled() error = %v", err)
	}
	if got := statusRequests.Load(); got != 2 {
		t.Fatalf("status requests = %d, want 2", got)
	}
}

func TestSamlSetEnabledFormatsAPIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		enabled bool
		path    string
		action  string
	}{
		{name: "enable", enabled: true, path: "/orgs/example-org/saml-group-sync/enable/", action: "enabling"},
		{name: "disable", enabled: false, path: "/orgs/example-org/saml-group-sync/disable/", action: "disabling"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != test.path {
					http.Error(w, "unexpected request", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = fmt.Fprint(w, `{"detail":"Invalid input.","fields":{"enabled":["denied"]}}`)
			}))
			defer server.Close()

			d := schema.TestResourceDataRaw(t, resourceSAML().Schema, map[string]interface{}{
				"organization": "example-org",
				"enabled":      test.enabled,
			})

			err := samlSetEnabled(d, samlTestProviderConfig(server))
			if err == nil {
				t.Fatal("samlSetEnabled() error = nil, want API error")
			}
			for _, want := range []string{
				"error " + test.action + " SAML group sync for organization (example-org)",
				"Invalid input.",
				"enabled",
				"denied",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("samlSetEnabled() error = %q, want it to contain %q", err, want)
				}
			}
		})
	}
}

func samlTestProviderConfig(server *httptest.Server) *providerConfig {
	config := cloudsmithapi.NewConfiguration()
	config.Servers = cloudsmithapi.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()
	return &providerConfig{
		Auth:      context.Background(),
		APIClient: cloudsmithapi.NewAPIClient(config),
	}
}

func writeSAMLListResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(paginationCountHeader, "1")
	w.Header().Set(paginationPageHeader, "1")
	w.Header().Set(paginationPageTotalHeader, "1")
	w.Header().Set(paginationPageSizeHeader, "100")
	_, _ = fmt.Fprint(w, `[{"idp_key":"group","idp_value":"developers","role":"Member","slug_perm":"mapping","team":"developers"}]`)
}
