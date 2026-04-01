//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	testAccProviders map[string]*schema.Provider
	testAccProvider  *schema.Provider

	testAccRepositoryNameSequence atomic.Uint64
)

const testAccRepositoryNameMaxLength = 50

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

// testAccUniqueRepositoryName keeps repository names unique across acceptance
// test runs while respecting Cloudsmith's 50 character repository name limit.
func testAccUniqueRepositoryName(base string) string {
	suffix := fmt.Sprintf("-%d-%d", time.Now().UnixMilli(), testAccRepositoryNameSequence.Add(1))
	maxBaseLength := testAccRepositoryNameMaxLength - len(suffix)

	if maxBaseLength < 1 {
		maxBaseLength = 1
	}

	base = strings.Trim(base, "-")
	if len(base) > maxBaseLength {
		base = strings.TrimRight(base[:maxBaseLength], "-")
	}

	if base == "" {
		base = "tfacc"
		if len(base) > maxBaseLength {
			base = base[:maxBaseLength]
		}
	}

	return base + suffix
}

func TestUniqueRepositoryName_MaxLength(t *testing.T) {
	t.Parallel()

	name := testAccUniqueRepositoryName("terraform-acc-test-repository-geo-ip-rules")
	if len(name) > testAccRepositoryNameMaxLength {
		t.Fatalf("repository name too long: got %d chars: %q", len(name), name)
	}

	otherName := testAccUniqueRepositoryName("terraform-acc-test-repository-geo-ip-rules")
	if name == otherName {
		t.Fatalf("expected unique repository names, got duplicate %q", name)
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
