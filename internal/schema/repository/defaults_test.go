// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package repository

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	cloudsmith "github.com/cloudsmith-io/cloudsmith-api-go"
)

// sdkDefaults returns the defaults the OpenAPI spec (as compiled into the Go SDK) declares
// for repository create and update requests.
func sdkDefaults(t *testing.T) map[string]interface{} {
	t.Helper()

	out := map[string]interface{}{}

	for _, v := range []interface{}{
		cloudsmith.NewRepositoryCreateWithDefaults(),
		cloudsmith.NewRepositoryRequestPatchWithDefaults(),
	} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}

		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}

		for k, val := range m {
			out[k] = val
		}
	}

	return out
}

// Values sourced from the SDK must match the OpenAPI spec exactly, and the spec must not
// declare a default we have not documented: guards against both typos and API changes
// arriving with an SDK bump.
func TestServerDefaultsMatchSDK(t *testing.T) {
	spec := sdkDefaults(t)
	attrs := resourceRepository().Schema

	for name, d := range serverDefaults {
		got, inSpec := spec[name]

		switch d.Source {
		case sourceSDK:
			if !inSpec {
				t.Errorf("%s: marked %q but the OpenAPI spec declares no default", name, sourceSDK)
			} else if fmt.Sprint(got) != d.Value {
				t.Errorf("%s: documented default %q, OpenAPI spec says %q", name, d.Value, got)
			}
		case sourceModel:
			if inSpec {
				t.Errorf("%s: the OpenAPI spec now declares a default (%v); switch the source to %q", name, got, sourceSDK)
			}
		default:
			t.Errorf("%s: unknown source %q", name, d.Source)
		}
	}

	for name, got := range spec {
		attr, ok := attrs[name]
		if !ok || !attr.Optional || attr.Default != nil {
			continue
		}

		_, documented := serverDefaults[name]
		_, exempt := attributesWithoutDefault[name]

		if !documented && !exempt {
			t.Errorf("%s: OpenAPI spec declares default %v but it is not in serverDefaults", name, got)
		}
	}
}

// New optional attributes must come with a deliberate decision about their default.
func TestEveryOptionalAttributeHasDefaultDecision(t *testing.T) {
	for name, attr := range resourceRepository().Schema {
		if !attr.Optional || attr.Default != nil {
			continue
		}

		_, documented := serverDefaults[name]
		_, exempt := attributesWithoutDefault[name]

		switch {
		case documented && exempt:
			t.Errorf("%s: listed in both serverDefaults and attributesWithoutDefault", name)
		case !documented && !exempt && attr.Computed && !strings.Contains(name, "retention"):
			t.Errorf("%s: optional+computed attribute has no entry in serverDefaults or attributesWithoutDefault", name)
		}
	}

	for name := range serverDefaults {
		if _, ok := resourceRepository().Schema[name]; !ok {
			t.Errorf("%s: in serverDefaults but not a repository attribute", name)
		}
	}
}

// The rendered description (what ends up in the registry docs) must state the default.
func TestRepositoryDescriptionsStateDefaults(t *testing.T) {
	attrs := ResourceRepository().Schema

	for name, d := range serverDefaults {
		want := fmt.Sprintf("`%s`", d.Value)
		if desc := attrs[name].Description; !strings.Contains(desc, want) {
			t.Errorf("%s: description does not mention default %s: %q", name, want, desc)
		}
	}
}
