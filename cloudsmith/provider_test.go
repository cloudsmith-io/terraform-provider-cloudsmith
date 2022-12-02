//nolint:testpackage
package cloudsmith

import (
	"os"
	"testing"

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
