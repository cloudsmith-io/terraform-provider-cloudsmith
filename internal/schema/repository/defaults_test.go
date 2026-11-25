// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package repository

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	cloudsmith "github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/util"
)

// sdkDefaults returns the defaults the OpenAPI spec (as compiled into the Go SDK) declares
// for repository create and update requests, keyed by request type.
func sdkDefaults(t *testing.T) map[string]map[string]interface{} {
	t.Helper()

	out := map[string]map[string]interface{}{}

	for name, v := range map[string]interface{}{
		"create": cloudsmith.NewRepositoryCreateWithDefaults(),
		"patch":  cloudsmith.NewRepositoryRequestPatchWithDefaults(),
	} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}

		m := map[string]interface{}{}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}

		out[name] = m
	}

	return out
}

// Values sourced from the SDK must match the OpenAPI spec exactly for both create and
// update requests, and the spec must not declare a default we have not documented: guards
// against typos and against API changes arriving with an SDK bump.
func TestServerDefaultsMatchSDK(t *testing.T) {
	spec := sdkDefaults(t)
	attrs := resourceRepository().Schema

	for name, d := range serverDefaults {
		for req, defaults := range spec {
			got, inSpec := defaults[name]

			switch d.Source {
			case sourceSDK:
				// A request type can omit a field, but a value that is present must agree.
				if inSpec && fmt.Sprint(got) != d.Value {
					t.Errorf("%s (%s): documented default %q, OpenAPI spec says %q", name, req, d.Value, got)
				}
			case sourceModel:
				if inSpec {
					t.Errorf("%s (%s): the OpenAPI spec now declares a default (%v); switch the source to %q", name, req, got, sourceSDK)
				}
			default:
				t.Errorf("%s: unknown source %q", name, d.Source)
			}
		}

		if d.Source == sourceSDK {
			_, inCreate := spec["create"][name]
			_, inPatch := spec["patch"][name]

			if !inCreate && !inPatch {
				t.Errorf("%s: marked %q but the OpenAPI spec declares no default", name, sourceSDK)
			}
		}
	}

	for req, defaults := range spec {
		for name, got := range defaults {
			attr, ok := attrs[name]
			if !ok || !attr.Optional || attr.Default != nil {
				continue
			}

			_, documented := serverDefaults[name]
			_, exempt := attributesWithoutDefault[name]

			if !documented && !exempt {
				t.Errorf("%s (%s): OpenAPI spec declares default %v but it is not in serverDefaults", name, req, got)
			}
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
		case !documented && !exempt && attr.Computed:
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
		attr, ok := attrs[name]
		if !ok {
			continue
		}

		want := util.FormatDefaultSentence(d.Value)
		if !strings.Contains(attr.Description, want) {
			t.Errorf("%s: description does not state %q: %q", name, want, attr.Description)
		}
	}
}
