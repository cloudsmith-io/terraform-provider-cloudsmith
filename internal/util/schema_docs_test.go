// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package util

import "testing"

func TestAppendDefaultSentence(t *testing.T) {
	cases := map[string]string{
		"Enables the feature.":    "Enables the feature. Defaults to `false`.",
		"Enables the feature":     "Enables the feature. Defaults to `false`.",
		"  Enables the feature  ": "Enables the feature. Defaults to `false`.",
		"":                        "Defaults to `false`.",
	}

	for description, want := range cases {
		if got := AppendDefaultSentence(description, false); got != want {
			t.Errorf("AppendDefaultSentence(%q) = %q, want %q", description, got, want)
		}
	}
}
