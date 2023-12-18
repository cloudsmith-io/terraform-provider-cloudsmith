package cloudsmith

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func denyRuleImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 3 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<deny_rule_slug>, got: %s", d.Id(),
		)
	}

	d.Set("namespace", idParts[0])
	d.SetId(idParts[1])
	return []*schema.ResourceData{d}, nil
}

func denyRuleCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	req := pc.APIClient.OrgsApi.OrgsDenyPolicyCreate(pc.Auth, namespace)
	req = req.Data(cloudsmith.PackageDenyPolicyRequest{
		Name:               nullableString(d, "name"),
		Enabled:            optionalBool(d, "enabled"),
		Description:        nullableString(d, "description"),
		PackageQueryString: nullableString(d, "package_query"),
	})
	denyRule, _, err := pc.APIClient.OrgsApi.OrgsDenyPolicyCreateExecute(req)
	if err != nil {
		return err
	}
	d.SetId(denyRule.GetSlugPerm())
	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsDenyPolicyRead(pc.Auth, namespace, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsDenyPolicyReadExecute(req); err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return err
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for deny rule (%s) to be created: %w", d.Id(), err)
	}
	return denyRuleRead(d, m)
}

func denyRuleRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	req := pc.APIClient.OrgsApi.OrgsDenyPolicyRead(pc.Auth, namespace, d.Id())
	rule, resp, err := pc.APIClient.OrgsApi.OrgsDenyPolicyReadExecute(req)

	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}
		return err

	}

	d.Set("name", rule.GetName())

	d.Set("description", rule.GetDescription())
	d.Set("package_query", rule.GetPackageQueryString())
	d.Set("enabled", rule.GetEnabled())
	d.Set("slug_perm", rule.GetSlugPerm())

	return nil
}

func denyRuleUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	req := pc.APIClient.OrgsApi.OrgsDenyPolicyPartialUpdate(pc.Auth, namespace, d.Id())
	req = req.Data(cloudsmith.PackageDenyPolicyRequestPatch{
		Name:               nullableString(d, "name"),
		Enabled:            optionalBool(d, "enabled"),
		Description:        nullableString(d, "description"),
		PackageQueryString: nullableString(d, "package_query"),
	})
	denyRule, _, err := pc.APIClient.OrgsApi.OrgsDenyPolicyPartialUpdateExecute(req)
	if err != nil {
		return err
	}
	d.SetId(denyRule.GetSlugPerm())
	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for a
		// deny rule being updated
		time.Sleep(time.Second * 5)
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for deny rule (%s) to be updated: %w", d.Id(), err)
	}
	return denyRuleRead(d, m)
}

func denyRuleDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")

	req := pc.APIClient.OrgsApi.OrgsDenyPolicyDelete(pc.Auth, namespace, d.Id())
	_, err := pc.APIClient.OrgsApi.OrgsDenyPolicyDeleteExecute(req)
	if err != nil {
		return err
	}

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsDenyPolicyRead(pc.Auth, namespace, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsDenyPolicyReadExecute(req); err != nil {
			if is404(resp) {
				return nil
			}
			return err
		}
		return errKeepWaiting
	}

	if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
		return fmt.Errorf("error waiting for deny rule (%s) to be deleted: %w", d.Id(), err)
	}
	return nil
}

//nolint:funlen
func denyRule() *schema.Resource {
	return &schema.Resource{
		Create: denyRuleCreate,
		Read:   denyRuleRead,
		Update: denyRuleUpdate,
		Delete: denyRuleDelete,

		Importer: &schema.ResourceImporter{
			StateContext: denyRuleImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "A descriptive name for the deny rule.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"slug_perm": {
				Type:         schema.TypeString,
				Description:  "The slug of the deny rule.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"description": {
				Type:         schema.TypeString,
				Description:  "Description of the rule.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"package_query": {
				Type:         schema.TypeString,
				Description:  "The query to match the packages to be blocked.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "Is the rule enabled?.",
				Optional:    true,
				Default:     true,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "Namespace to which this rule belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}
