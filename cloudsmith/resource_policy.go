package cloudsmith

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/cloudsmith-io/cloudsmith-go-v2/models/apierrors"
	"github.com/cloudsmith-io/cloudsmith-go-v2/models/components"
	"github.com/cloudsmith-io/cloudsmith-go-v2/optionalnullable"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func importPolicy(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <workspace>.<policy_slug_perm>, got: %s", d.Id(),
		)
	}
	_ = d.Set("workspace", parts[0])
	d.SetId(parts[1])
	return []*schema.ResourceData{d}, nil
}

func resourcePolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := requiredString(d, "workspace")
	pc := m.(*providerConfig)
	body := components.PolicyInput1{
		Name:        requiredString(d, "name"),
		Rego:        requiredString(d, "rego"),
		Description: optionalnullable.From(optionalString(d, "description")),
		Enabled:     optionalBool(d, "enabled"),
		IsTerminal:  optionalBool(d, "is_terminal"),
		Precedence:  optionalInt64(d, "precedence"),
	}
	resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesCreate(
		ctx, workspace, body,
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("creating policy in workspace %q: %w", workspace, err))
	}
	if resp == nil || resp.Policy == nil {
		return diag.Errorf("policy create returned no body")
	}
	d.SetId(resp.Policy.SlugPerm)
	return resourcePolicyRead(ctx, d, m)
}

func resourcePolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := requiredString(d, "workspace")
	pc := m.(*providerConfig)
	resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesRetrieve(
		ctx, d.Id(), workspace,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("retrieving policy %q in workspace %q: %w", d.Id(), workspace, err))
	}
	if resp == nil || resp.Policy == nil {
		d.SetId("")
		return nil
	}
	setPolicyOnSchema(d, resp.Policy, "slug_perm")
	return nil
}

func resourcePolicyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := requiredString(d, "workspace")
	if requiredBool(d, "read_only") {
		return diag.Errorf(
			"policy %q is read-only and cannot be updated through this provider. "+
				"To stop managing it with Terraform, use `terraform state rm`. "+
				"To accept server-side drift, add a `lifecycle { ignore_changes = [...] }` block.",
			d.Id(),
		)
	}
	body := components.PolicyInput1{
		Name:        requiredString(d, "name"),
		Rego:        requiredString(d, "rego"),
		Description: optionalnullable.From(optionalString(d, "description")),
		Enabled:     optionalBool(d, "enabled"),
		IsTerminal:  optionalBool(d, "is_terminal"),
		Precedence:  optionalInt64(d, "precedence"),
	}
	pc := m.(*providerConfig)
	resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesUpdate(
		ctx, d.Id(), workspace, body,
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("updating policy %q in workspace %q: %w", d.Id(), workspace, err))
	}
	if resp != nil && resp.Policy != nil {
		setPolicyOnSchema(d, resp.Policy, "slug_perm")
		return nil
	}
	return resourcePolicyRead(ctx, d, m)
}

func resourcePolicyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := requiredString(d, "workspace")
	pc := m.(*providerConfig)
	_, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesDestroy(
		ctx, d.Id(), workspace,
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return diag.FromErr(fmt.Errorf("deleting policy %q in workspace %q: %w", d.Id(), workspace, err))
	}
	return nil
}

func policyToMap(p *components.Policy) map[string]interface{} {
	entry := map[string]interface{}{
		"name":        p.Name,
		"rego":        p.Rego,
		"slug_perm":   p.SlugPerm,
		"version":     int(p.Version),
		"read_only":   p.ReadOnly,
		"created_at":  timeToString(p.CreatedAt),
		"updated_at":  timeToString(p.UpdatedAt),
		"enabled":     boolOrFalse(p.Enabled),
		"is_terminal": boolOrFalse(p.IsTerminal),
		"precedence":  int64OrZero(p.Precedence),
		"description": "",
	}
	if v, ok := p.Description.Get(); ok && v != nil {
		entry["description"] = *v
	}
	return entry
}

func setPolicyOnSchema(d *schema.ResourceData, p *components.Policy, slugPermKey string) {
	m := policyToMap(p)
	if slugPermKey != "slug_perm" {
		m[slugPermKey] = m["slug_perm"]
		delete(m, "slug_perm")
	}
	for k, v := range m {
		_ = d.Set(k, v)
	}
}

func resourcePolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePolicyCreate,
		ReadContext:   resourcePolicyRead,
		UpdateContext: resourcePolicyUpdate,
		DeleteContext: resourcePolicyDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importPolicy,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:         schema.TypeString,
				Description:  "Workspace the policy belongs to.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"name": {
				Type:         schema.TypeString,
				Description:  "The name of the policy.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "The description of the policy.",
				Optional:    true,
			},
			"rego": {
				Type:         schema.TypeString,
				Description:  "The rego source for the policy logic.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				DiffSuppressFunc: func(_, old, new string, _ *schema.ResourceData) bool {
					return strings.TrimRightFunc(old, unicode.IsSpace) == strings.TrimRightFunc(new, unicode.IsSpace)
				},
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "If true, the policy is enabled.",
				Optional:    true,
				Default:     true,
			},
			"is_terminal": {
				Type:        schema.TypeBool,
				Description: "If true and the policy matches, no further policies are evaluated.",
				Optional:    true,
				Default:     false,
			},
			"precedence": {
				Type:        schema.TypeInt,
				Description: "The order in which this policy is evaluated relative to other policies.",
				Optional:    true,
				Computed:    true,
			},
			"slug_perm": {
				Type:        schema.TypeString,
				Description: "The policy's unique permanent slug.",
				Computed:    true,
			},
			"version": {
				Type:        schema.TypeInt,
				Description: "The version of the rego code.",
				Computed:    true,
			},
			"read_only": {
				Type:        schema.TypeBool,
				Description: "Whether the policy is read-only (only specific variables can be updated).",
				Computed:    true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "The time the policy was created.",
				Computed:    true,
			},
			"updated_at": {
				Type:        schema.TypeString,
				Description: "The time the policy was last updated.",
				Computed:    true,
			},
		},
	}
}
