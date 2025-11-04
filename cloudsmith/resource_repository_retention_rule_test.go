//nolint:testpackage
package cloudsmith

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestAccRepositoryRetentionRule_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
				),
			},
			{
				Config: testAccRepositoryRetentionRuleConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_count_limit", "100"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_days_limit", "28"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_enabled", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_group_by_name", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_group_by_format", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_group_by_package_type", "false"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_size_limit", "0"),
				),
			},
			{
				Config: testAccRepositoryRetentionRuleConfigZeroValues,
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryCheckExists("cloudsmith_repository.test-retention"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_count_limit", "0"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_days_limit", "0"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_enabled", "true"),
					resource.TestCheckResourceAttr("cloudsmith_repository_retention_rule.test", "retention_size_limit", "0"),
				),
			},
		},
		CheckDestroy: testAccRepositoryCheckDestroy("cloudsmith_repository.test-retention"),
	})
}

var testAccRepositoryConfig = fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
  name        = "terraform-acc-repo-retention-rule"
  namespace   = "%s"
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccRepositoryRetentionRuleConfigBasic = fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
  name        = "terraform-acc-repo-retention-rule"
  namespace   = "%s"
}

resource "cloudsmith_repository_retention_rule" "test" {
  namespace = "%s"
  repository = cloudsmith_repository.test-retention.name
  retention_enabled = false
  retention_count_limit = 100
  retention_days_limit = 28
  retention_group_by_name = false
  retention_group_by_format = false
  retention_group_by_package_type = false
  retention_size_limit = 0
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

var testAccRepositoryRetentionRuleConfigZeroValues = fmt.Sprintf(`
resource "cloudsmith_repository" "test-retention" {
  name        = "terraform-acc-repo-retention-rule"
  namespace   = "%s"
}

resource "cloudsmith_repository_retention_rule" "test" {
  namespace = "%s"
  repository = cloudsmith_repository.test-retention.name
  retention_enabled = true
  retention_count_limit = 0
  retention_days_limit = 0
  retention_group_by_name = false
  retention_group_by_format = false
  retention_group_by_package_type = false
  retention_size_limit = 0
}
`, os.Getenv("CLOUDSMITH_NAMESPACE"), os.Getenv("CLOUDSMITH_NAMESPACE"))

// Unit tests for retention rule zero value handling
// These tests verify that explicitly set zero values are handled correctly

func TestRepositoryRetentionRule_ZeroValues(t *testing.T) {
	// Create a ResourceData instance with the actual schema
	resource := resourceRepoRetentionRule()
	d := resource.TestResourceData()

	// Set all the required fields
	d.Set("namespace", "test-namespace")
	d.Set("repository", "test-repository")
	d.Set("retention_enabled", true)

	// The key test: explicitly set retention_count_limit to 0
	// This was the original problem - when users set this to 0, it wasn't working
	d.Set("retention_count_limit", 0)

	// Test that our fixed optionalInt64 function correctly handles the explicit zero
	countLimit := optionalInt64(d, "retention_count_limit")
	if countLimit == nil {
		t.Fatal("retention_count_limit = 0 returned nil - the fix didn't work!")
	}
	if *countLimit != 0 {
		t.Fatalf("retention_count_limit = 0 returned %d instead of 0", *countLimit)
	}

	// Test the same for retention_days_limit
	d.Set("retention_days_limit", 0)
	daysLimit := optionalInt64(d, "retention_days_limit")
	if daysLimit == nil {
		t.Fatal("retention_days_limit = 0 returned nil - the fix didn't work!")
	}
	if *daysLimit != 0 {
		t.Fatalf("retention_days_limit = 0 returned %d instead of 0", *daysLimit)
	}

	// Test the same for retention_size_limit  
	d.Set("retention_size_limit", 0)
	sizeLimit := optionalInt64(d, "retention_size_limit")
	if sizeLimit == nil {
		t.Fatal("retention_size_limit = 0 returned nil - the fix didn't work!")
	}
	if *sizeLimit != 0 {
		t.Fatalf("retention_size_limit = 0 returned %d instead of 0", *sizeLimit)
	}

	// Verify that unset values still return nil (expected behavior)
	d2 := resource.TestResourceData()
	unsetValue := optionalInt64(d2, "retention_count_limit")
	if unsetValue != nil {
		t.Errorf("Expected unset value to return nil, got %v", unsetValue)
	}
}

func TestRepositoryRetentionRule_GetOkVsGetOkExists(t *testing.T) {
	// This test demonstrates the difference between GetOk() and GetOkExists()
	// for handling zero values with default values

	// Create a ResourceData instance with a field that has a default
	testSchema := map[string]*schema.Schema{
		"field_with_default": {
			Type:     schema.TypeInt,
			Optional: true,
			Default:  100, // This default would cause issues with GetOk()
		},
	}
	resource := &schema.Resource{Schema: testSchema}
	d := resource.TestResourceData()

	// Set the field to 0 explicitly
	d.Set("field_with_default", 0)

	// Using GetOkExists (our fix) should work
	if value, ok := d.GetOkExists("field_with_default"); ok {
		intValue := value.(int)
		if intValue != 0 {
			t.Errorf("GetOkExists: Expected 0, got %d", intValue)
		}
	} else {
		t.Error("GetOkExists should return true for explicitly set 0 value")
	}

	// Using GetOk (the old way) demonstrates the issue
	if value, ok := d.GetOk("field_with_default"); ok {
		intValue := value.(int)
		t.Logf("GetOk returned: %d", intValue)
	} else {
		t.Log("GetOk returned false - this demonstrates the issue with zero values")
	}
}
