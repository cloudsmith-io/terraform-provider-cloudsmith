// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// DocumentDefaults appends "Defaults to `X`." to the description of every optional
// attribute (recursively) that declares a schema-level Default and whose description
// does not already mention a default. It keeps the generated registry docs in sync with
// the actual provider behaviour.
func DocumentDefaults(resources map[string]*schema.Resource) {
	for _, r := range resources {
		documentDefaults(r.Schema)
	}
}

func documentDefaults(s map[string]*schema.Schema) {
	for _, attr := range s {
		if nested, ok := attr.Elem.(*schema.Resource); ok {
			documentDefaults(nested.Schema)
		}

		if !attr.Optional || attr.Default == nil || attr.Default == "" {
			continue
		}

		if MentionsDefault(attr.Description) {
			continue
		}

		attr.Description = strings.TrimSpace(attr.Description) + " " + FormatDefaultSentence(attr.Default)
	}
}

// FormatDefaultSentence renders the standard "Defaults to `X`." sentence.
func FormatDefaultSentence(v interface{}) string {
	return fmt.Sprintf("Defaults to `%v`.", v)
}

// MentionsDefault reports whether a description already states a default value.
func MentionsDefault(description string) bool {
	l := strings.ToLower(description)

	return strings.Contains(l, "defaults to") || strings.Contains(l, "default is") || strings.Contains(l, "(default)")
}
