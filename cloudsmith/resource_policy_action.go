package cloudsmith

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudsmith-io/cloudsmith-go-v2/models/apierrors"
	"github.com/cloudsmith-io/cloudsmith-go-v2/models/components"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func firstBlock(d *schema.ResourceData, key string) (map[string]interface{}, bool) {
	raw, ok := d.GetOk(key)
	if !ok {
		return nil, false
	}
	list := raw.([]interface{})
	if len(list) == 0 || list[0] == nil {
		return nil, false
	}
	return list[0].(map[string]interface{}), true
}

func blockIsSet(v interface{}) bool {
	list, _ := v.([]interface{})
	return len(list) > 0
}

func blockStringSet(v interface{}) []string {
	raw := v.(*schema.Set).List()
	out := make([]string, len(raw))
	for i, item := range raw {
		out[i] = item.(string)
	}
	return out
}

const (
	actionSetPackageState   = "set_package_state"
	actionAddPackageTags    = "add_package_tags"
	actionRemovePackageTags = "remove_package_tags"
)

var actionTypeBlocks = []string{actionSetPackageState, actionAddPackageTags, actionRemovePackageTags}

func importPolicyAction(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), ".", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <workspace>.<policy_slug_perm>.<action_slug_perm>, got: %s", d.Id(),
		)
	}
	_ = d.Set("workspace", parts[0])
	_ = d.Set("policy_slug_perm", parts[1])
	d.SetId(parts[2])
	return []*schema.ResourceData{d}, nil
}

func resourcePolicyActionCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := requiredString(d, "workspace")
	body, err := buildPolicyActionInput(d)
	if err != nil {
		return diag.FromErr(err)
	}
	pc := m.(*providerConfig)
	policySlug := requiredString(d, "policy_slug_perm")
	resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesActionsCreate(
		ctx, policySlug, workspace, &body,
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("creating action on policy %q in workspace %q: %w", policySlug, workspace, formatV2APIError(err)))
	}
	if resp == nil || resp.PolicyAction == nil {
		return diag.Errorf("policy action create returned no body")
	}
	actionSlugPerm, err := policyActionSlugPerm(resp.PolicyAction)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(actionSlugPerm)
	return resourcePolicyActionRead(ctx, d, m)
}

func resourcePolicyActionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := requiredString(d, "workspace")
	pc := m.(*providerConfig)
	policySlug := requiredString(d, "policy_slug_perm")
	resp, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesActionsRetrieve(
		ctx, d.Id(), policySlug, workspace,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("retrieving action %q on policy %q in workspace %q: %w", d.Id(), policySlug, workspace, formatV2APIError(err)))
	}
	if resp == nil || resp.PolicyAction == nil {
		d.SetId("")
		return nil
	}
	return diag.FromErr(setPolicyActionState(d, resp.PolicyAction))
}

func resourcePolicyActionUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := requiredString(d, "workspace")
	body, err := buildPolicyActionInput(d)
	if err != nil {
		return diag.FromErr(err)
	}
	pc := m.(*providerConfig)
	policySlug := requiredString(d, "policy_slug_perm")
	_, err = pc.V2ApiClient.Workspaces.WorkspacesPoliciesActionsUpdate(
		ctx, d.Id(), policySlug, workspace, &body,
	)
	if err != nil {
		return diag.FromErr(fmt.Errorf("updating action %q on policy %q in workspace %q: %w", d.Id(), policySlug, workspace, formatV2APIError(err)))
	}
	return resourcePolicyActionRead(ctx, d, m)
}

func resourcePolicyActionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := requiredString(d, "workspace")
	pc := m.(*providerConfig)
	policySlug := requiredString(d, "policy_slug_perm")
	_, err := pc.V2ApiClient.Workspaces.WorkspacesPoliciesActionsDestroy(
		ctx, d.Id(), policySlug, workspace,
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return diag.FromErr(fmt.Errorf("deleting action %q on policy %q in workspace %q: %w", d.Id(), policySlug, workspace, formatV2APIError(err)))
	}
	return nil
}

func buildPolicyActionInput(d *schema.ResourceData) (components.PolicyActionRequest, error) {
	precedence := optionalInt64(d, "precedence")
	if block, ok := firstBlock(d, actionSetPackageState); ok {
		return components.CreatePolicyActionRequestSetPackageState(
			components.SetPackageStatePolicyActionTypedRequest{
				Precedence:   precedence,
				PackageState: components.PackageStateEnum(block["package_state"].(string)),
			},
		), nil
	}
	if block, ok := firstBlock(d, actionAddPackageTags); ok {
		return components.CreatePolicyActionRequestAddPackageTags(
			components.AddPackageTagsPolicyActionTypedRequest{
				Precedence: precedence,
				Tags:       blockStringSet(block["tags"]),
			},
		), nil
	}
	if block, ok := firstBlock(d, actionRemovePackageTags); ok {
		return components.CreatePolicyActionRequestRemovePackageTags(
			components.RemovePackageTagsPolicyActionTypedRequest{
				Precedence: precedence,
				Tags:       blockStringSet(block["tags"]),
			},
		), nil
	}
	return components.PolicyActionRequest{}, fmt.Errorf("no action type block set; expected one of %v", actionTypeBlocks)
}

type actionMeta struct {
	slugPerm   string
	precedence *int64
	createdAt  time.Time
	updatedAt  time.Time
}

func describePolicyAction(pa *components.PolicyAction) (actionMeta, string, []interface{}, error) {
	switch pa.Type {
	case components.PolicyActionTypeSetPackageState:
		t := pa.SetPackageStatePolicyActionTyped
		if t == nil {
			return actionMeta{}, "", nil, fmt.Errorf("set_package_state action is nil")
		}
		return actionMeta{slugPerm: t.SlugPerm, precedence: t.Precedence, createdAt: t.CreatedAt, updatedAt: t.UpdatedAt},
			actionSetPackageState,
			[]interface{}{map[string]interface{}{"package_state": string(t.PackageState)}},
			nil
	case components.PolicyActionTypeAddPackageTags:
		t := pa.AddPackageTagsPolicyActionTyped
		if t == nil {
			return actionMeta{}, "", nil, fmt.Errorf("add_package_tags action is nil")
		}
		return actionMeta{slugPerm: t.SlugPerm, precedence: t.Precedence, createdAt: t.CreatedAt, updatedAt: t.UpdatedAt},
			actionAddPackageTags,
			[]interface{}{map[string]interface{}{"tags": flattenStrings(t.Tags)}},
			nil
	case components.PolicyActionTypeRemovePackageTags:
		t := pa.RemovePackageTagsPolicyActionTyped
		if t == nil {
			return actionMeta{}, "", nil, fmt.Errorf("remove_package_tags action is nil")
		}
		return actionMeta{slugPerm: t.SlugPerm, precedence: t.Precedence, createdAt: t.CreatedAt, updatedAt: t.UpdatedAt},
			actionRemovePackageTags,
			[]interface{}{map[string]interface{}{"tags": flattenStrings(t.Tags)}},
			nil
	default:
		return actionMeta{}, "", nil, fmt.Errorf("unsupported policy action type %s; expected one of %v", pa.Type, actionTypeBlocks)
	}
}

func setPolicyActionState(d *schema.ResourceData, pa *components.PolicyAction) error {
	am, key, block, err := describePolicyAction(pa)
	if err != nil {
		return err
	}

	_ = d.Set("slug_perm", am.slugPerm)
	_ = d.Set("created_at", timeToString(am.createdAt))
	_ = d.Set("updated_at", timeToString(am.updatedAt))
	_ = d.Set("precedence", int64OrZero(am.precedence))

	for _, k := range actionTypeBlocks {
		if k == key {
			_ = d.Set(k, block)
		} else {
			_ = d.Set(k, []interface{}{})
		}
	}
	return nil
}

func policyActionSlugPerm(pa *components.PolicyAction) (string, error) {
	am, _, _, err := describePolicyAction(pa)
	if err != nil {
		return "", err
	}
	return am.slugPerm, nil
}

func customizeDiffPolicyActionType(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if d.Id() == "" {
		return nil
	}
	for _, block := range actionTypeBlocks {
		oldVal, newVal := d.GetChange(block)
		if blockIsSet(oldVal) != blockIsSet(newVal) {
			if err := d.ForceNew(block); err != nil {
				return err
			}
		}
	}
	return nil
}

func resourcePolicyAction() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePolicyActionCreate,
		ReadContext:   resourcePolicyActionRead,
		UpdateContext: resourcePolicyActionUpdate,
		DeleteContext: resourcePolicyActionDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importPolicyAction,
		},

		CustomizeDiff: customizeDiffPolicyActionType,

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:         schema.TypeString,
				Description:  "Workspace the policy belongs to.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"policy_slug_perm": {
				Type:         schema.TypeString,
				Description:  "The slug_perm of the policy this action belongs to.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"precedence": {
				Type:        schema.TypeInt,
				Description: "The order in which this action occurs relative to other actions for the same policy.",
				Optional:    true,
				Computed:    true,
			},
			actionSetPackageState: {
				Type:         schema.TypeList,
				Description:  "Set the state of matching packages.",
				MaxItems:     1,
				Optional:     true,
				ExactlyOneOf: actionTypeBlocks,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"package_state": {
							Type:         schema.TypeString,
							Description:  "The state to set on matching packages.",
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"AVAILABLE", "DELETED", "QUARANTINED", "HIDDEN"}, false),
						},
					},
				},
			},
			actionAddPackageTags: {
				Type:         schema.TypeList,
				Description:  "Add tags to matching packages.",
				MaxItems:     1,
				Optional:     true,
				ExactlyOneOf: actionTypeBlocks,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"tags": {
							Type:        schema.TypeSet,
							Description: "The tags to add to matching packages.",
							Required:    true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringIsNotEmpty,
							},
						},
					},
				},
			},
			actionRemovePackageTags: {
				Type:         schema.TypeList,
				Description:  "Remove tags from matching packages.",
				MaxItems:     1,
				Optional:     true,
				ExactlyOneOf: actionTypeBlocks,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"tags": {
							Type:        schema.TypeSet,
							Description: "The tags to remove from matching packages.",
							Required:    true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringIsNotEmpty,
							},
						},
					},
				},
			},
			"slug_perm": {
				Type:        schema.TypeString,
				Description: "The action's unique permanent slug.",
				Computed:    true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "The time the action was created.",
				Computed:    true,
			},
			"updated_at": {
				Type:        schema.TypeString,
				Description: "The time the action was last updated.",
				Computed:    true,
			},
		},
	}
}
