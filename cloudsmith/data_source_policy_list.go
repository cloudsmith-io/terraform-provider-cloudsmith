package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudsmith-io/cloudsmith-go-v2/models/components"
	"github.com/cloudsmith-io/cloudsmith-go-v2/models/operations"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var policyListPageSize = DefaultPageSize

func dataSourcePolicyListRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := requiredString(d, "workspace")
	pc := m.(*providerConfig)
	query := optionalString(d, "query")
	sort := optionalString(d, "sort")

	exec := func(page, pageSize int64) ([]components.Policy, *http.Response, error) {
		req := operations.WorkspacesPoliciesListRequest{
			Workspace: workspace,
			Page:      page,
			PageSize:  &pageSize,
			Query:     query,
			Sort:      sort,
		}
		resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesList(ctx, req)
		if err != nil {
			return nil, nil, err
		}
		var results []components.Policy
		if resp.PaginatedPolicyList != nil {
			results = resp.PaginatedPolicyList.Results
		}
		return results, resp.HTTPMeta.Response, nil
	}

	all, err := PaginateAllHTTP[components.Policy](exec, PaginationOptions{
		PageSize:   policyListPageSize,
		MaxResults: DefaultMaxResults,
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("listing policies in workspace %q: %w", workspace, formatV2APIError(err)))
	}

	if err := d.Set("policies", flattenPolicies(all)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}

func flattenPolicies(in []components.Policy) []interface{} {
	out := make([]interface{}, 0, len(in))
	for i := range in {
		out = append(out, policyToMap(&in[i]))
	}
	return out
}

func dataSourcePolicyList() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePolicyListRead,
		Description: "List policies in a Cloudsmith workspace, filtered by a required query.",

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:         schema.TypeString,
				Description:  "Workspace to list policies for.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"query": {
				Type:         schema.TypeString,
				Description:  "Search query (e.g. 'name:my-policy').",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"sort": {
				Type:        schema.TypeString,
				Description: "Sort field (e.g. 'created_at', '-created_at', 'name', '-name').",
				Optional:    true,
			},
			"policies": {
				Type:        schema.TypeList,
				Description: "The list of policies matching the query.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name":        {Type: schema.TypeString, Description: "The name of the policy.", Computed: true},
						"description": {Type: schema.TypeString, Description: "The description of the policy.", Computed: true},
						"rego":        {Type: schema.TypeString, Description: "The rego source for the policy logic.", Computed: true},
						"enabled":     {Type: schema.TypeBool, Description: "If true, the policy is enabled.", Computed: true},
						"is_terminal": {Type: schema.TypeBool, Description: "If true and the policy matches, no further policies are evaluated.", Computed: true},
						"precedence":  {Type: schema.TypeInt, Description: "The order in which this policy is evaluated relative to other policies.", Computed: true},
						"slug_perm":   {Type: schema.TypeString, Description: "The policy's unique permanent slug.", Computed: true},
						"version":     {Type: schema.TypeInt, Description: "The version of the rego code.", Computed: true},
						"read_only":   {Type: schema.TypeBool, Description: "Whether the policy is read-only.", Computed: true},
						"created_at":  {Type: schema.TypeString, Description: "The time the policy was created.", Computed: true},
						"updated_at":  {Type: schema.TypeString, Description: "The time the policy was last updated.", Computed: true},
					},
				},
			},
		},
	}
}
