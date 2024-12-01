//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	testAccProviders map[string]*schema.Provider
	testAccProvider  *schema.Provider
)

//nolint:gochecknoinits
func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"cloudsmith": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("CLOUDSMITH_API_KEY"); v == "" {
		t.Fatal("CLOUDSMITH_API_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSMITH_NAMESPACE"); v == "" {
		t.Fatal("CLOUDSMITH_NAMESPACE must be set for acceptance tests")
	}
}
func TestAccProvider_UserSelfValidation(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/self/" {
			w.Header().Set("Content-Type", "application/json")
			if r.Header.Get("X-Api-Key") == "valid-token" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"email": "test@example.com", "name": "Test User", "slug": "test-user", "slug_perm": "test-user"}`)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, `{"error": "invalid API credentials"}`)
			}
		}
	}))
	defer server.Close()

	tests := []struct {
		name   string
		apiKey string
	}{
		{
			name:   "ValidToken",
			apiKey: "valid-token",
		},
		{
			name:   "InvalidToken",
			apiKey: "invalid-token",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CLOUDSMITH_API_HOST", server.URL)
			t.Setenv("CLOUDSMITH_API_KEY", tc.apiKey)
			resource.Test(t, resource.TestCase{
				PreCheck:  func() { testAccPreCheck(t) },
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config:      selfConfig,
						ExpectError: regexp.MustCompile("invalid API credentials"),
						SkipFunc: func() (bool, error) {
							// Skip error check for valid token case
							return tc.apiKey == "valid-token", nil
						},
					},
				},
			})
		})
	}
}

var selfConfig string = `
data "cloudsmith_user_self" "this" {
}`
