// Copyright 2026 Cloudsmith Ltd

package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	EnforceRefresh string = "enforce_refresh"
	IsEnabled      string = "is_enabled"
	LastAppliedAt  string = "last_applied_at"
	MaxAgeHours    string = "max_age_hours"
	RuleType       string = "rule_type"
	Slug           string = "slug"
)

const ruleTypeAllAccounts = "All Accounts"

// Matches the minimum the API enforces, so a bad value fails at plan time.
const minAPIKeyRuleMaxAgeHours = 24

var apiKeyRuleTypes = []string{"Service Accounts", "User Accounts"}

func apiKeyRuleMaxAgeHours(d *schema.ResourceData) cloudsmith.NullableInt64 {
	var maxAge *int64
	if hours := int64(d.Get(MaxAgeHours).(int)); hours > 0 {
		maxAge = &hours
	}

	return *cloudsmith.NewNullableInt64(maxAge)
}

func importAPIKeyRule(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<rule_slug_perm>, got: %s", d.Id(),
		)
	}

	d.Set(Organization, idParts[0])
	d.SetId(idParts[1])
	return []*schema.ResourceData{d}, nil
}

func resourceAPIKeyRuleCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, Organization)

	req := pc.APIClient.OrgsApi.OrgsApiKeyRulesCreate(pc.Auth, org)
	req = req.Data(cloudsmith.OrganizationApiKeyRuleRequest{
		EnforceRefresh: optionalBool(d, EnforceRefresh),
		IsEnabled:      optionalBool(d, IsEnabled),
		MaxAgeHours:    apiKeyRuleMaxAgeHours(d),
		RuleType:       requiredString(d, RuleType),
	})

	apiKeyRule, _, err := pc.APIClient.OrgsApi.OrgsApiKeyRulesCreateExecute(req)
	if err != nil {
		return formatAPIError(err)
	}

	d.SetId(apiKeyRule.GetSlugPerm())

	if err := waitForCreation(func() (*http.Response, error) {
		req := pc.APIClient.OrgsApi.OrgsApiKeyRulesRead(pc.Auth, org, d.Id())
		_, resp, err := pc.APIClient.OrgsApi.OrgsApiKeyRulesReadExecute(req)
		return resp, err
	}, "api key rule", d.Id()); err != nil {
		return err
	}

	return resourceAPIKeyRuleRead(d, m)
}

func resourceAPIKeyRuleRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, Organization)

	req := pc.APIClient.OrgsApi.OrgsApiKeyRulesRead(pc.Auth, org, d.Id())
	apiKeyRule, resp, err := pc.APIClient.OrgsApi.OrgsApiKeyRulesReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return formatAPIError(err)
	}

	if apiKeyRule.GetRuleType() == ruleTypeAllAccounts {
		return fmt.Errorf(
			"api key rule %s is an %q rule, which this resource does not manage; "+
				"it is mutually exclusive with the per-account-type rules. Remove it and "+
				"declare separate 'User Accounts' and 'Service Accounts' rules instead",
			d.Id(), ruleTypeAllAccounts,
		)
	}

	_ = d.Set(CreatedAt, apiKeyRule.GetCreatedAt().String())
	_ = d.Set(EnforceRefresh, apiKeyRule.GetEnforceRefresh())
	_ = d.Set(IsEnabled, apiKeyRule.GetIsEnabled())
	_ = d.Set(LastAppliedAt, apiKeyRule.GetLastAppliedAt().String())
	_ = d.Set(RuleType, apiKeyRule.GetRuleType())
	_ = d.Set(Slug, apiKeyRule.GetSlug())
	_ = d.Set(SlugPerm, apiKeyRule.GetSlugPerm())
	_ = d.Set(UpdatedAt, apiKeyRule.GetUpdatedAt().String())

	if apiKeyRule.MaxAgeHours.IsSet() && apiKeyRule.MaxAgeHours.Get() != nil {
		_ = d.Set(MaxAgeHours, *apiKeyRule.MaxAgeHours.Get())
	} else {
		_ = d.Set(MaxAgeHours, nil)
	}

	_ = d.Set(Organization, org)

	return nil
}

func resourceAPIKeyRuleUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, Organization)

	req := pc.APIClient.OrgsApi.OrgsApiKeyRulesPartialUpdate(pc.Auth, org, d.Id())
	req = req.Data(cloudsmith.OrganizationApiKeyRuleRequestPatch{
		EnforceRefresh: optionalBool(d, EnforceRefresh),
		IsEnabled:      optionalBool(d, IsEnabled),
		MaxAgeHours:    apiKeyRuleMaxAgeHours(d),
	})

	apiKeyRule, _, err := pc.APIClient.OrgsApi.OrgsApiKeyRulesPartialUpdateExecute(req)
	if err != nil {
		return formatAPIError(err)
	}

	d.SetId(apiKeyRule.GetSlugPerm())

	if err := waitForUpdate("api key rule", d.Id()); err != nil {
		return err
	}

	return resourceAPIKeyRuleRead(d, m)
}

func resourceAPIKeyRuleDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, Organization)

	req := pc.APIClient.OrgsApi.OrgsApiKeyRulesDelete(pc.Auth, org, d.Id())
	if _, err := pc.APIClient.OrgsApi.OrgsApiKeyRulesDeleteExecute(req); err != nil {
		return formatAPIError(err)
	}

	if err := waitForDeletion(func() (*http.Response, error) {
		req := pc.APIClient.OrgsApi.OrgsApiKeyRulesRead(pc.Auth, org, d.Id())
		_, resp, err := pc.APIClient.OrgsApi.OrgsApiKeyRulesReadExecute(req)
		return resp, err
	}, "api key rule", d.Id()); err != nil {
		return err
	}

	return nil
}

//nolint:funlen
func resourceAPIKeyRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAPIKeyRuleCreate,
		Read:   resourceAPIKeyRuleRead,
		Update: resourceAPIKeyRuleUpdate,
		Delete: resourceAPIKeyRuleDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importAPIKeyRule,
		},

		Schema: map[string]*schema.Schema{
			CreatedAt: {
				Type:        schema.TypeString,
				Description: "The time the rule was created at.",
				Computed:    true,
			},
			EnforceRefresh: {
				Type:        schema.TypeBool,
				Description: "When enabled, API keys that violate this rule are replaced automatically.",
				Optional:    true,
				Computed:    true,
			},
			IsEnabled: {
				Type:        schema.TypeBool,
				Description: "Whether this rule is currently active and enforced.",
				Optional:    true,
				Computed:    true,
			},
			LastAppliedAt: {
				Type:        schema.TypeString,
				Description: "The last time this rule was evaluated and applied by the expiry task.",
				Computed:    true,
			},
			MaxAgeHours: {
				Type: schema.TypeInt,
				Description: "The maximum permitted age of an API key, in hours. API keys older than " +
					"this lose access until refreshed. Omit to disable the age limit.",
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(minAPIKeyRuleMaxAgeHours),
			},
			Organization: {
				Type:         schema.TypeString,
				Description:  "Organization to which this rule belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			RuleType: {
				Type: schema.TypeString,
				Description: "The account types this rule applies to. One of: " +
					"'Service Accounts', 'User Accounts'.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(apiKeyRuleTypes, false),
			},
			Slug: {
				Type:        schema.TypeString,
				Description: "Human-readable identifier, generated from the rule type.",
				Computed:    true,
			},
			SlugPerm: {
				Type:        schema.TypeString,
				Description: "An auto-generated id that uniquely identifies the rule.",
				Computed:    true,
			},
			UpdatedAt: {
				Type:        schema.TypeString,
				Description: "The time the rule was last updated at.",
				Computed:    true,
			},
		},
	}
}
