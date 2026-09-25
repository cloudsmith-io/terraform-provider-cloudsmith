// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package cloudsmith

import (
	"fmt"
	"strings"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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

func resourceRepoRetentionRuleCreate(d *schema.ResourceData, meta interface{}) error {
	pc := meta.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repo := requiredString(d, "repository")

	// check if rule is already enabled
	rule, _, err := pc.APIClient.ReposApi.RepoRetentionRead(pc.Auth, namespace, repo).Execute()
	if err != nil {
		return fmt.Errorf("cannot create resource: %w", err)
	}
	if rule.RetentionEnabled != nil && *rule.RetentionEnabled {
		return fmt.Errorf("%s", "cannot create resource as retention rule already enabled on repository.")
	}

	req := pc.APIClient.ReposApi.RepoRetentionPartialUpdate(pc.Auth, namespace, repo)

	// For integer fields with defaults, we need to always send the value to handle
	// the case where users explicitly set them to 0 (which would otherwise be
	// indistinguishable from "not set" using GetOk)
	retentionCountLimit := int64(d.Get("retention_count_limit").(int))
	retentionDaysLimit := int64(d.Get("retention_days_limit").(int))

	updateData := cloudsmith.RepositoryRetentionRulesRequestPatch{
		RetentionEnabled:            optionalBool(d, "retention_enabled"),
		RetentionGroupByName:        optionalBool(d, "retention_group_by_name"),
		RetentionGroupByFormat:      optionalBool(d, "retention_group_by_format"),
		RetentionGroupByPackageType: optionalBool(d, "retention_group_by_package_type"),
		RetentionPackageQueryString: nullableString(d, "retention_package_query_string"),
		RetentionCountLimit:         &retentionCountLimit,
		RetentionDaysLimit:          &retentionDaysLimit,
	}

	// For retention_size_limit, we need to always send the value to handle
	// the case where users explicitly set it to 0 (which would otherwise be
	// indistinguishable from "not set" using GetOk)
	retentionSizeLimit := int64(d.Get("retention_size_limit").(int))
	updateData.RetentionSizeLimit = &retentionSizeLimit

	req = req.Data(updateData)

	// Execute the request
	_, _, err = req.Execute()
	if err != nil {
		return fmt.Errorf("error updating repository retention rule: %w", formatAPIError(err))
	}

	d.SetId(fmt.Sprintf("%s.%s", namespace, repo))

	// Wait for the API to reflect the updated retention rule values. The
	// Cloudsmith API is eventually consistent, so a read immediately after a
	// write may return stale values. We poll until the count limit and query
	// string match what was submitted.
	checkerFunc := func() error {
		resp, httpResp, err := pc.APIClient.ReposApi.RepoRetentionRead(pc.Auth, namespace, repo).Execute()
		if err != nil {
			// Only treat expected eventual-consistency cases (e.g., 404) as transient.
			if httpResp != nil && httpResp.StatusCode == 404 {
				return errKeepWaiting
			}
			// For all other errors, return the original error so the caller sees the real cause.
			return err
		}
		if resp.GetRetentionCountLimit() != retentionCountLimit {
			return errKeepWaiting
		}
		wantQuery := d.Get("retention_package_query_string").(string)
		gotQuery := ""
		if resp.RetentionPackageQueryString.IsSet() && resp.RetentionPackageQueryString.Get() != nil {
			gotQuery = *resp.RetentionPackageQueryString.Get()
		}
		if gotQuery != wantQuery {
			return errKeepWaiting
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for repository retention rule %s/%s to be updated: %w", namespace, repo, err)
	}

	return resourceRepoRetentionRuleRead(d, meta)
}

func resourceRepoRetentionRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	pc := meta.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repo := requiredString(d, "repository")

	req := pc.APIClient.ReposApi.RepoRetentionPartialUpdate(pc.Auth, namespace, repo)

	// For integer fields with defaults, we need to always send the value to handle
	// the case where users explicitly set them to 0 (which would otherwise be
	// indistinguishable from "not set" using GetOk)
	retentionCountLimit := int64(d.Get("retention_count_limit").(int))
	retentionDaysLimit := int64(d.Get("retention_days_limit").(int))

	updateData := cloudsmith.RepositoryRetentionRulesRequestPatch{
		RetentionEnabled:            optionalBool(d, "retention_enabled"),
		RetentionGroupByName:        optionalBool(d, "retention_group_by_name"),
		RetentionGroupByFormat:      optionalBool(d, "retention_group_by_format"),
		RetentionGroupByPackageType: optionalBool(d, "retention_group_by_package_type"),
		RetentionPackageQueryString: nullableString(d, "retention_package_query_string"),
		RetentionCountLimit:         &retentionCountLimit,
		RetentionDaysLimit:          &retentionDaysLimit,
	}

	// For retention_size_limit, we need to always send the value to handle
	// the case where users explicitly set it to 0 (which would otherwise be
	// indistinguishable from "not set" using GetOk)
	retentionSizeLimit := int64(d.Get("retention_size_limit").(int))
	updateData.RetentionSizeLimit = &retentionSizeLimit

	req = req.Data(updateData)

	// Execute the request
	_, _, err := req.Execute()
	if err != nil {
		return fmt.Errorf("error updating repository retention rule: %w", formatAPIError(err))
	}

	d.SetId(fmt.Sprintf("%s.%s", namespace, repo))

	// Wait for the API to reflect the updated retention rule values. The
	// Cloudsmith API is eventually consistent, so a read immediately after a
	// write may return stale values. We poll until the count limit and query
	// string match what was submitted.
	checkerFunc := func() error {
		resp, httpResp, err := pc.APIClient.ReposApi.RepoRetentionRead(pc.Auth, namespace, repo).Execute()
		if err != nil {
			// Only treat expected eventual-consistency cases (e.g., 404) as transient.
			if httpResp != nil && httpResp.StatusCode == 404 {
				return errKeepWaiting
			}
			// For all other errors, return the original error so the caller sees the real cause.
			return err
		}
		if resp.GetRetentionCountLimit() != retentionCountLimit {
			return errKeepWaiting
		}
		wantQuery := d.Get("retention_package_query_string").(string)
		gotQuery := ""
		if resp.RetentionPackageQueryString.IsSet() && resp.RetentionPackageQueryString.Get() != nil {
			gotQuery = *resp.RetentionPackageQueryString.Get()
		}
		if gotQuery != wantQuery {
			return errKeepWaiting
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for repository retention rule %s/%s to be updated: %w", namespace, repo, err)
	}

	return resourceRepoRetentionRuleRead(d, meta)
}

func resourceRepoRetentionRuleRead(d *schema.ResourceData, meta interface{}) error {
	pc := meta.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repo := requiredString(d, "repository")

	// Execute the request
	resp, _, err := pc.APIClient.ReposApi.RepoRetentionRead(pc.Auth, namespace, repo).Execute()
	if err != nil {
		return fmt.Errorf("error reading repository retention rule: %w", formatAPIError(err))
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
		if httpResp != nil && httpResp.StatusCode == 404 {
			return nil
		}
		return fmt.Errorf("error disabling repository retention rule: %w", formatAPIError(err))
	}

	d.SetId("")
	return nil
}

const retentionMaxBytes int64 = 21474836480

func Int64AtMost(maximum int64) schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		v, ok := i.(int)
		if !ok {
			return diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       "Invalid value type",
				Detail:        fmt.Sprintf("expected an integer, got %T", i),
				AttributePath: path,
			}}
		}
		if int64(v) > maximum {
			return diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       "Value too large",
				Detail:        fmt.Sprintf("expected a value of at most %d, got %d", maximum, v),
				AttributePath: path,
			}}
		}
		return nil
	}
}

func resourceRepoRetentionRule() *schema.Resource {
	return &schema.Resource{
		Description: "**Note: use of this resource is discouraged; prefer the `retention_rule` block on the [repository resource](https://registry.terraform.io/providers/cloudsmith-io/cloudsmith/latest/docs/resources/repository).** Because this resource targets a repository by name, nothing prevents several instances of it from pointing at the same repository, and each apply will clobber the others' settings. The nested block is limited to one rule per repository, so that can't happen. Manage a repository's retention rule with one or the other, never both.\n\n" +
			"The repository retention rules resource allows the management of retention rules for a given Cloudsmith repository. Using this resource, it is possible to define rules that control the retention of packages based on various criteria such as count, days, size, and grouping.\n\n" +
			"Note that while retention rules can be managed in this manner, changes made outside of Terraform may not be reflected in the Terraform state.\n\n" +
			"**Note: Retention rule settings are only applied once retention is enabled for the repository.**\n\n" +
			"See [docs.cloudsmith.com](https://docs.cloudsmith.com/artifact-management/retention-rules) for full retention rules documentation.",

		Create: resourceRepoRetentionRuleCreate,
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
				Type:             schema.TypeInt,
				Optional:         true,
				Description:      "The maximum total size (in bytes) of packages to retain. Must be between 0 and 21474836480 (21.47 GB / 21474.83 MB).",
				ValidateDiagFunc: Int64AtMost(retentionMaxBytes),
			},
			"retention_package_query_string": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A package search expression which, if provided, filters the packages to be deleted.",
			},
		},
	}
}
