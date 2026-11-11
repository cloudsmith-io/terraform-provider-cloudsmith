// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// BuildQueryString joins the members of a set into a space separated query.
func BuildQueryString(set *schema.Set) string {
	var query strings.Builder
	for _, v := range set.List() {
		query.WriteString(v.(string))
		query.WriteString(" ")
	}
	return query.String()
}

// RetentionMaxBytes is the largest retention size accepted by the API.
const RetentionMaxBytes int64 = 21474836480

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
