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

	"github.com/cloudsmith-io/cloudsmith-go-v2/models/operations"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPolicyListDataSource_basic(t *testing.T) {
	t.Parallel()

	seedName := testAccUniquePolicyName("TF Acc List Seed")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPolicyCheckDestroy("cloudsmith_policy.seed"),
		Steps:        testAccPolicyListDataSourceSteps(t.Context(), seedName, testAccNamespace(), false, testAccProvider),
	})
}

func TestAccPolicyListDataSource_filter(t *testing.T) {
	t.Parallel()

	seedName := testAccUniquePolicyName("TF Acc List Filter")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPolicyCheckDestroy("cloudsmith_policy.seed"),
		Steps:        testAccPolicyListDataSourceSteps(t.Context(), seedName, testAccNamespace(), true, testAccProvider),
	})
}

func testAccPolicyListDataSourceSteps(ctx context.Context, name, workspace string, filtered bool, provider *schema.Provider) []resource.TestStep {
	address, sort := "data.cloudsmith_policy_list.all", ""
	config := testAccPolicyListDataSourceConfigBasic(name, workspace)
	checkSort := resource.TestCheckNoResourceAttr(address, "sort")
	if filtered {
		address, sort = "data.cloudsmith_policy_list.filtered", "-created_at"
		config = testAccPolicyListDataSourceConfigFilter(name, workspace)
		checkSort = resource.TestCheckResourceAttr(address, "sort", sort)
	}
	return []resource.TestStep{
		{
			Config: testAccPolicyListSeedConfig(name, workspace),
			Check: func(s *terraform.State) error {
				ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				// Check functions receive a fixed snapshot. Wait on the live query
				// before Terraform ever reads the data source in the next step.
				return testAccPolicyListSeedVisible(ctx, provider.Meta().(*providerConfig), s, workspace, name, sort, 500*time.Millisecond)
			},
		},
		{
			Config: config,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet(address, "id"),
				resource.TestCheckResourceAttr(address, "workspace", workspace),
				resource.TestCheckResourceAttr(address, "query", fmt.Sprintf("name:%q", name)),
				checkSort,
				resource.TestCheckResourceAttr(address, "policies.#", "1"),
				resource.TestCheckResourceAttr(address, "policies.0.name", name),
				resource.TestCheckResourceAttrPair(address, "policies.0.slug_perm", "cloudsmith_policy.seed", "slug_perm"),
			),
		},
	}
}

func testAccPolicyListSeedVisible(ctx context.Context, pc *providerConfig, s *terraform.State, workspace, name, sort string, interval time.Duration) error {
	seed, ok := s.RootModule().Resources["cloudsmith_policy.seed"]
	if !ok || seed.Primary == nil || seed.Primary.ID == "" {
		return fmt.Errorf("owned policy seed missing from state")
	}
	if seed.Primary.Attributes["workspace"] != workspace || seed.Primary.Attributes["name"] != name {
		return fmt.Errorf("unexpected owned policy seed: want %q/%q", workspace, name)
	}
	query := fmt.Sprintf("name:%q", name)
	pageSize := int64(DefaultPageSize)
	req := operations.WorkspacesPoliciesListRequest{
		Workspace: workspace, Query: &query, Page: 1, PageSize: &pageSize,
	}
	if sort != "" {
		req.Sort = &sort
	}
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("waiting for owned policy %q in query %q: %w", seed.Primary.ID, query, err)
		}
		resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesList(ctx, req)
		if err != nil {
			return fmt.Errorf("querying owned policy %q visibility: %w", seed.Primary.ID, err)
		}
		if resp == nil || resp.PaginatedPolicyList == nil {
			return fmt.Errorf("querying owned policy %q visibility: missing list response", seed.Primary.ID)
		}
		policies := resp.PaginatedPolicyList.Results
		if len(policies) == 1 && policies[0].SlugPerm == seed.Primary.ID && policies[0].Name == name {
			return nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("waiting for owned policy %q in query %q: %w", seed.Primary.ID, query, ctx.Err())
		case <-timer.C:
		}
	}
}

func TestPolicyListDataSource_ReturnsInitialListError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/self/" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"email": "test@example.com", "name": "Test User", "slug": "test-user", "slug_perm": "test-user"}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"detail": "temporary failure"}`)
	}))
	defer server.Close()

	pc, diags := newProviderConfig(context.Background(), server.URL, staticToken("valid-token"), map[string]interface{}{}, "test-agent")
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/self/" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"email": "test@example.com", "name": "Test User", "slug": "test-user", "slug_perm": "test-user"}`)
			return
		}

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
		w.Header().Set("X-Pagination-Count", strconv.Itoa(totalPages))
		w.Header().Set("X-Pagination-Page", strconv.Itoa(page))
		w.Header().Set("X-Pagination-PageTotal", strconv.Itoa(totalPages))
		w.Header().Set("X-Pagination-PageSize", "1")
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
	defer server.Close()

	pc, diags := newProviderConfig(context.Background(), server.URL, staticToken("valid-token"), map[string]interface{}{}, "test-agent")
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

func testAccPolicyListSeedConfig(name, workspace string) string {
	return fmt.Sprintf(`
resource "cloudsmith_policy" "seed" {
    workspace = %q
    name      = %q
    rego      = <<-EOT
        package cloudsmith.policy
        default allow := true
    EOT
}
`, workspace, name)
}

func testAccPolicyListDataSourceConfigBasic(name, workspace string) string {
	return testAccPolicyListSeedConfig(name, workspace) + fmt.Sprintf(`
data "cloudsmith_policy_list" "all" {
    workspace  = %q
    query      = %q
    depends_on = [cloudsmith_policy.seed]
}
`, workspace, fmt.Sprintf("name:%q", name))
}

func testAccPolicyListDataSourceConfigFilter(name, workspace string) string {
	return testAccPolicyListSeedConfig(name, workspace) + fmt.Sprintf(`
data "cloudsmith_policy_list" "filtered" {
    workspace  = %q
    query      = %q
    sort       = "-created_at"
    depends_on = [cloudsmith_policy.seed]
}
`, workspace, fmt.Sprintf("name:%q", name))
}
