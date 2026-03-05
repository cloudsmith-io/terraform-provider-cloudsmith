//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	testAccProviders                   map[string]*schema.Provider
	testAccProviderFactories           map[string]func() (*schema.Provider, error)
	testAccProvider                    *schema.Provider
	testAccNameSuffix                  = shortTestAccSuffix()
	testAccCheckProviderConfig         *providerConfig
	testAccCheckProviderConfigErr      error
	testAccCheckProviderConfigInitLock sync.Mutex
)

func testAccUniqueName(base string) string {
	const maxNameLen = 30
	hashSuffix := shortHash(base)
	tail := hashSuffix + testAccNameSuffix

	maxBaseLen := maxNameLen - len(tail) - 1
	if maxBaseLen < 1 {
		return tail
	}
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}

	return fmt.Sprintf("%s-%s", base, tail)
}

func shortTestAccSuffix() string {
	s := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(s) > 4 {
		return s[len(s)-4:]
	}
	return s
}

func shortHash(s string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	hash := strconv.FormatUint(uint64(h.Sum32()), 36)
	if len(hash) > 3 {
		return hash[:3]
	}
	return hash
}

//nolint:gochecknoinits
func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"cloudsmith": testAccProvider,
	}
	testAccProviderFactories = map[string]func() (*schema.Provider, error){
		"cloudsmith": func() (*schema.Provider, error) {
			return Provider(), nil
		},
	}
}

func testAccProviderConfigForChecks() (*providerConfig, error) {
	testAccCheckProviderConfigInitLock.Lock()
	defer testAccCheckProviderConfigInitLock.Unlock()

	if testAccCheckProviderConfig != nil {
		return testAccCheckProviderConfig, nil
	}
	if testAccCheckProviderConfigErr != nil {
		return nil, testAccCheckProviderConfigErr
	}

	apiKey := os.Getenv("CLOUDSMITH_API_KEY")
	if apiKey == "" {
		testAccCheckProviderConfigErr = fmt.Errorf("CLOUDSMITH_API_KEY must be set for acceptance checks")
		return nil, testAccCheckProviderConfigErr
	}

	apiHost := os.Getenv("CLOUDSMITH_API_HOST")
	if apiHost == "" {
		apiHost = "https://api.cloudsmith.io/v1"
	}

	config, diags := newProviderConfig(apiHost, apiKey, map[string]interface{}{}, "terraform-provider-cloudsmith-acc-tests")
	if diags.HasError() {
		msg := "failed to initialize acceptance check provider config"
		if len(diags) > 0 && diags[0].Summary != "" {
			msg = fmt.Sprintf("%s: %s", msg, diags[0].Summary)
		}
		testAccCheckProviderConfigErr = fmt.Errorf(msg)
		return nil, testAccCheckProviderConfigErr
	}

	testAccCheckProviderConfig = config
	return testAccCheckProviderConfig, nil
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
				PreCheck:          func() { testAccPreCheck(t) },
				ProviderFactories: testAccProviderFactories,
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
