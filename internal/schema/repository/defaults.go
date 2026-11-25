// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package repository

import (
	"strings"

	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/util"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// serverDefault describes the value the Cloudsmith API applies when an optional
// (and computed) attribute is left unset.
type serverDefault struct {
	// Value is the documented default.
	Value string
	// Source records where the value was taken from, so it can be re-checked.
	// "sdk" values are verified automatically against the generated API client
	// (see TestServerDefaultsMatchSDK). "model" values are booleans that the OpenAPI
	// spec does not carry; they come from the Django model's field default in
	// cloudsmith-io/cloudsmith (cloudsmith/package/models/repository.py).
	Source string
}

const (
	sourceSDK   = "sdk"
	sourceModel = "model"
)

// serverDefaults maps repository attributes to the default applied by the API.
// Attributes with a schema-level Default are handled by util.DocumentDefaults instead.
var serverDefaults = map[string]serverDefault{
	// Enums/strings: present in the OpenAPI spec (and so the Go SDK).
	"copy_packages":                 {"Read", sourceSDK},
	"default_privilege":             {"None", sourceSDK},
	"delete_packages":               {"Admin", sourceSDK},
	"manage_entitlements_privilege": {"Admin", sourceSDK},
	"move_packages":                 {"Admin", sourceSDK},
	"replace_packages":              {"Write", sourceSDK},
	"resync_packages":               {"Admin", sourceSDK},
	"scan_packages":                 {"Read", sourceSDK},
	"use_entitlements_privilege":    {"Read", sourceSDK},
	"view_statistics":               {"Read", sourceSDK},

	// Booleans: from the Django model field defaults.
	"contextual_auth_realm":                {"true", sourceModel},
	"copy_own":                             {"true", sourceModel},
	"cosign_signing_enabled":               {"false", sourceModel},
	"delete_own":                           {"true", sourceModel},
	"docker_refresh_tokens_enabled":        {"false", sourceModel},
	"index_files":                          {"true", sourceModel},
	"move_own":                             {"true", sourceModel},
	"npm_upstream_tags_take_precedence":    {"false", sourceModel},
	"nuget_native_signing_enabled":         {"false", sourceModel},
	"proxy_npmjs":                          {"false", sourceModel},
	"proxy_pypi":                           {"false", sourceModel},
	"raw_package_index_enabled":            {"false", sourceModel},
	"raw_package_index_signatures_enabled": {"false", sourceModel},
	"replace_packages_by_default":          {"false", sourceModel}, // model field: republish_default
	"resync_own":                           {"true", sourceModel},
	"scan_own":                             {"true", sourceModel},
	"show_setup_all":                       {"false", sourceModel},
	"strict_npm_validation":                {"false", sourceModel},
	"use_debian_labels":                    {"false", sourceModel},
	"use_default_cargo_upstream":           {"true", sourceModel},
	"use_noarch_packages":                  {"true", sourceModel},
	"use_source_packages":                  {"true", sourceModel},
	"use_vulnerability_scanning":           {"true", sourceModel},
	"user_entitlements_enabled":            {"true", sourceModel},
}

// attributesWithoutDefault lists optional+computed attributes that intentionally have
// no fixed default (server generated or dependent on other values), so adding a new
// attribute without deciding on its default fails TestEveryOptionalAttributeHasDefaultDecision.
var attributesWithoutDefault = map[string]string{
	"slug":              "derived from name",
	"skills_versioning": "documented in its description",
	"storage_region":    "documented in its description",
}

func documentServerDefaults(s map[string]*schema.Schema) {
	for name, d := range serverDefaults {
		attr, ok := s[name]
		if !ok || util.MentionsDefault(attr.Description) {
			continue
		}

		attr.Description = strings.TrimSpace(attr.Description) + " " + util.FormatDefaultSentence(d.Value)
	}
}
