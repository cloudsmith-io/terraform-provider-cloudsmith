package cloudsmith

import (
	"fmt"
	"strings"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func importRepoRetentionRule(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf("expected id of format <namespace>.<repo>")
	}

	d.Set("namespace", idParts[0])
	d.Set("repository", idParts[1])
	return []*schema.ResourceData{d}, nil
}

func resourceRepoRetentionRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	pc := meta.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repo := requiredString(d, "repository")

	req := pc.APIClient.ReposApi.RepoRetentionPartialUpdate(pc.Auth, namespace, repo)
	updateData := cloudsmith.RepositoryRetentionRulesRequestPatch{
		RetentionEnabled:            optionalBool(d, "retention_enabled"),
		RetentionGroupByName:        optionalBool(d, "retention_group_by_name"),
		RetentionGroupByFormat:      optionalBool(d, "retention_group_by_format"),
		RetentionGroupByPackageType: optionalBool(d, "retention_group_by_package_type"),
		RetentionPackageQueryString: nullableString(d, "retention_package_query_string"),
		RetentionCountLimit:         optionalInt64(d, "retention_count_limit"),
		RetentionDaysLimit:          optionalInt64(d, "retention_days_limit"),
		RetentionSizeLimit:          optionalInt64(d, "retention_size_limit"),
	}

	req = req.Data(updateData)

	// Execute the request
	_, httpResp, err := req.Execute()
	if err != nil {
		switch httpResp.StatusCode {
		case 400:
			return fmt.Errorf("request could not be processed: %s", err)
		case 404:
			return fmt.Errorf("namespace or repository not found: %s", err)
		case 422:
			return fmt.Errorf("missing or invalid parameters: %s", err)
		default:
			return fmt.Errorf("error updating repository retention rule: %s", err)
		}
	}

	// Handle the response
	d.SetId(fmt.Sprintf("%s.%s", namespace, repo))
	return resourceRepoRetentionRuleRead(d, meta)
}

func resourceRepoRetentionRuleRead(d *schema.ResourceData, meta interface{}) error {
	pc := meta.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repo := requiredString(d, "repository")

	// Execute the request
	resp, httpResp, err := pc.APIClient.ReposApi.RepoRetentionRead(pc.Auth, namespace, repo).Execute()
	if err != nil {
		switch httpResp.StatusCode {
		case 400:
			return fmt.Errorf("request could not be processed: %s", err)
		case 404:
			return fmt.Errorf("namespace or repository not found: %s", err)
		case 422:
			return fmt.Errorf("missing or invalid parameters: %s", err)
		default:
			return fmt.Errorf("error reading repository retention rule: %s", err)
		}
	}

	d.Set("retention_count_limit", resp.RetentionCountLimit)
	d.Set("retention_days_limit", resp.RetentionDaysLimit)
	d.Set("retention_enabled", resp.RetentionEnabled)
	d.Set("retention_group_by_name", resp.RetentionGroupByName)
	d.Set("retention_group_by_format", resp.RetentionGroupByFormat)
	d.Set("retention_group_by_package_type", resp.RetentionGroupByPackageType)
	d.Set("retention_size_limit", resp.RetentionSizeLimit)
	if resp.RetentionPackageQueryString.IsSet() && resp.RetentionPackageQueryString.Get() != nil {
		d.Set("retention_package_query_string", *resp.RetentionPackageQueryString.Get())
	} else {
		d.Set("retention_package_query_string", "")
	}
	d.SetId(fmt.Sprintf("%s.%s", namespace, repo))

	return nil
}

func resourceRepoRetentionRuleDelete(d *schema.ResourceData, meta interface{}) error {
	pc := meta.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repo := requiredString(d, "repository")

	req := pc.APIClient.ReposApi.RepoRetentionPartialUpdate(pc.Auth, namespace, repo)
	updateData := cloudsmith.RepositoryRetentionRulesRequestPatch{
		RetentionEnabled: cloudsmith.PtrBool(false),
	}
	req = req.Data(updateData)

	_, httpResp, err := req.Execute()
	if err != nil {
		switch httpResp.StatusCode {
		case 400:
			return fmt.Errorf("request could not be processed: %s", err)
		case 404:
			return nil
		case 422:
			return fmt.Errorf("missing or invalid parameters: %s", err)
		default:
			return fmt.Errorf("error disabling repository retention rule: %s", err)
		}
	}

	d.SetId("")
	return nil
}

func resourceRepoRetentionRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepoRetentionRuleUpdate,
		Read:   resourceRepoRetentionRuleRead,
		Update: resourceRepoRetentionRuleUpdate,
		Delete: resourceRepoRetentionRuleDelete,
		Importer: &schema.ResourceImporter{
			State: importRepoRetentionRule,
		},
		Schema: map[string]*schema.Schema{
			"namespace": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The namespace of the repository.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"repository": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The name of the repository.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"retention_count_limit": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      100,
				Description:  "The maximum number of packages to retain. Must be between 0 and 10000.",
				ValidateFunc: validation.IntBetween(0, 10000),
			},
			"retention_days_limit": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      28,
				Description:  "The number of days of packages to retain. Must be between 0 and 180. Defaults to 28 days.",
				ValidateFunc: validation.IntBetween(0, 180),
			},
			"retention_enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "If true, the retention lifecycle rules will be activated for the repository and settings will be updated.",
			},
			"retention_group_by_format": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, retention will apply to packages by package formats rather than across all package formats.",
			},
			"retention_group_by_name": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, retention will apply to groups of packages by name rather than all packages.",
			},
			"retention_group_by_package_type": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, retention will apply to packages by package type rather than across all package types for one or more formats.",
			},
			"retention_size_limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The maximum total size (in bytes) of packages to retain. Must be between 0 and 21474836480 (21.47 GB / 21474.83 MB).",
			},
			"retention_package_query_string": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A package search expression which, if provided, filters the packages to be deleted.",
			},
		},
	}
}
