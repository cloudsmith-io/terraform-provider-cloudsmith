package cloudsmith

import (
	"context"
	"fmt"

	"github.com/cloudsmith-io/cloudsmith-go-v2/models/apierrors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := requiredString(d, "workspace")
	pc := m.(*providerConfig)
	policySlugPerm := requiredString(d, "policy_slug_perm")
	resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesRetrieve(
		ctx, policySlugPerm, workspace,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return diag.Errorf("policy %q not found in workspace %q", policySlugPerm, workspace)
		}
		return diag.FromErr(fmt.Errorf("retrieving policy %q in workspace %q: %w", policySlugPerm, workspace, err))
	}
	if resp == nil || resp.Policy == nil {
		return diag.Errorf("policy %q not found in workspace %q", policySlugPerm, workspace)
	}
	p := resp.Policy
	d.SetId(p.SlugPerm)
	setPolicyOnSchema(d, p, "policy_slug_perm")
	return nil
}

func dataSourcePolicy() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePolicyRead,
		Description: "Get an existing policy in a Cloudsmith workspace.",

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:         schema.TypeString,
				Description:  "Workspace the policy belongs to.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"policy_slug_perm": {
				Type:         schema.TypeString,
				Description:  "The slug_perm of the policy.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"name":        {Type: schema.TypeString, Description: "The name of the policy.", Computed: true},
			"description": {Type: schema.TypeString, Description: "The description of the policy.", Computed: true},
			"rego":        {Type: schema.TypeString, Description: "The rego source for the policy logic.", Computed: true},
			"enabled":     {Type: schema.TypeBool, Description: "If true, the policy is enabled.", Computed: true},
			"is_terminal": {Type: schema.TypeBool, Description: "If true and the policy matches, no further policies are evaluated.", Computed: true},
			"precedence":  {Type: schema.TypeInt, Description: "The order in which this policy is evaluated relative to other policies.", Computed: true},
			"version":     {Type: schema.TypeInt, Description: "The version of the rego code.", Computed: true},
			"read_only":   {Type: schema.TypeBool, Description: "Whether the policy is read-only (only specific variables can be updated).", Computed: true},
			"created_at":  {Type: schema.TypeString, Description: "The time the policy was created.", Computed: true},
			"updated_at":  {Type: schema.TypeString, Description: "The time the policy was last updated.", Computed: true},
		},
	}
}
