package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestAccPolicyListDataSource_basic(t *testing.T) {
	t.Parallel()

	seedName := testAccUniquePolicyName("TF Acc List Seed")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPolicyCheckDestroy("cloudsmith_policy.seed"),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyListDataSourceConfigBasic(seedName),
				Check: testAccRetry(15*time.Second, 500*time.Millisecond,
					resource.TestCheckResourceAttr("data.cloudsmith_policy_list.all", "policies.#", "1"),
				),
			},
		},
	})
}

func TestAccPolicyListDataSource_filter(t *testing.T) {
	t.Parallel()

	seedName := testAccUniquePolicyName("TF Acc List Filter")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPolicyCheckDestroy("cloudsmith_policy.seed"),
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyListDataSourceConfigFilter(seedName),
				Check: testAccRetry(15*time.Second, 500*time.Millisecond,
					resource.TestCheckResourceAttr("data.cloudsmith_policy_list.filtered", "policies.#", "1"),
				),
			},
		},
	})
}

func TestPolicyListDataSource_ReturnsInitialListError(t *testing.T) {
	t.Parallel()
	v1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/self/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"email": "test@example.com", "name": "Test User", "slug": "test-user", "slug_perm": "test-user"}`)
	}))
	defer v1Server.Close()

	v2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"detail": "temporary failure"}`)
	}))
	defer v2Server.Close()

	pc, diags := newProviderConfig(v1Server.URL, v2Server.URL, "valid-token", map[string]interface{}{}, "test-agent")
	if diags.HasError() {
		t.Fatalf("unexpected provider config diagnostics: %v", diags)
	}

	d := schema.TestResourceDataRaw(t, dataSourcePolicyList().Schema, map[string]interface{}{
		"workspace": "test-workspace",
		"query":     "name:test-policy",
	})

	diags = dataSourcePolicyListRead(context.Background(), d, pc)
	if !diags.HasError() {
		t.Fatal("expected policy list API error to be returned")
	}
	if got := diags[0].Summary; !strings.Contains(got, "listing policies in workspace") {
		t.Fatalf("expected contextual error, got %q", got)
	}
}

func TestPolicyListDataSource_PaginatesAcrossAllPages(t *testing.T) {
	// Not parallel: modifies policyListPageSize global.
	const totalPages = 10

	old := policyListPageSize
	policyListPageSize = 1
	defer func() { policyListPageSize = old }()

	var (
		mu           sync.Mutex
		pagesVisited []int
	)

	v1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/self/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"email": "test@example.com", "name": "Test User", "slug": "test-user", "slug_perm": "test-user"}`)
	}))
	defer v1Server.Close()

	v2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.URL.Query().Get("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 || page > totalPages {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		pagesVisited = append(pagesVisited, page)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"results": [{
				"created_at": "2025-01-01T00:00:00Z",
				"name": "policy-%d",
				"rego": "package cloudsmith.policy\ndefault allow := true",
				"read_only": false,
				"slug_perm": "slug-%d",
				"updated_at": "2025-01-01T00:00:00Z",
				"version": 1
			}],
			"pagetotal": %d
		}`, page, page, totalPages)
	}))
	defer v2Server.Close()

	pc, diags := newProviderConfig(v1Server.URL, v2Server.URL, "valid-token", map[string]interface{}{}, "test-agent")
	if diags.HasError() {
		t.Fatalf("unexpected provider config diagnostics: %v", diags)
	}

	d := schema.TestResourceDataRaw(t, dataSourcePolicyList().Schema, map[string]interface{}{
		"workspace": "test-workspace",
		"query":     "name:policy",
	})

	diags = dataSourcePolicyListRead(context.Background(), d, pc)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}

	policies := d.Get("policies").([]interface{})
	if got := len(policies); got != totalPages {
		t.Errorf("expected %d policies, got %d", totalPages, got)
	}

	mu.Lock()
	defer mu.Unlock()
	if got := len(pagesVisited); got != totalPages {
		t.Errorf("expected %d page requests, got %d (pages: %v)", totalPages, got, pagesVisited)
	}
	for i, p := range pagesVisited {
		if p != i+1 {
			t.Errorf("page request %d: expected page %d, got %d", i, i+1, p)
		}
	}
}

func testAccPolicyListDataSourceConfigBasic(name string) string {
	return fmt.Sprintf(`
resource "cloudsmith_policy" "seed" {
    workspace = "%s"
    name      = "%s"
    rego      = <<-EOT
        package cloudsmith.policy
        default allow := true
    EOT
}

data "cloudsmith_policy_list" "all" {
    workspace  = "%s"
    query      = "name:\"%s\""
    depends_on = [cloudsmith_policy.seed]
}
`, testAccNamespace(), name, testAccNamespace(), name)
}

func testAccPolicyListDataSourceConfigFilter(name string) string {
	return fmt.Sprintf(`
resource "cloudsmith_policy" "seed" {
    workspace = "%s"
    name      = "%s"
    rego      = <<-EOT
        package cloudsmith.policy
        default allow := true
    EOT
}

data "cloudsmith_policy_list" "filtered" {
    workspace  = "%s"
    query      = "name:\"%s\""
    sort       = "-created_at"
    depends_on = [cloudsmith_policy.seed]
}
`, testAccNamespace(), name, testAccNamespace(), name)
}
