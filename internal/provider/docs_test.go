// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:testpackage
package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// apiImporters lists resources whose importer reads from the API, so it can't run offline.
var apiImporters = map[string]bool{"cloudsmith_saml_auth": true}

func walkSchema(prefix string, s map[string]*schema.Schema, fn func(path string, attr *schema.Schema)) {
	for name, attr := range s {
		fn(prefix+name, attr)

		if nested, ok := attr.Elem.(*schema.Resource); ok {
			walkSchema(prefix+name+".", nested.Schema, fn)
		}
	}
}

// Every schema-level Default must be visible in the rendered docs.
func TestSchemaDefaultsDocumented(t *testing.T) {
	for resName, res := range Provider().ResourcesMap {
		walkSchema(resName+".", res.Schema, func(path string, attr *schema.Schema) {
			if !attr.Optional || attr.Default == nil || attr.Default == "" {
				return
			}

			want := fmt.Sprintf("%v", attr.Default)
			statesDefault := regexp.MustCompile("(?i)(defaults to|default is) `?" + regexp.QuoteMeta(want) + `\b`)
			if !statesDefault.MatchString(attr.Description) {
				t.Errorf("%s: description must state the default %q explicitly: %q", path, want, attr.Description)
			}
		})
	}
}

var importCmd = regexp.MustCompile(`(?m)^terraform import (cloudsmith_\w+)\.(\w+) (\S+)$`)

// Every importable resource must ship an import.sh whose ID the provider's own importer
// accepts, and which it rejects when a segment is removed (so the documented segment
// count is exact). Importers that need the API are only checked for shape.
func TestImportExamples(t *testing.T) {
	for resName, res := range Provider().ResourcesMap {
		if res.Importer == nil {
			continue
		}

		t.Run(resName, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "resources", resName, "import.sh"))
			if err != nil {
				t.Fatalf("missing import example: %v", err)
			}

			m := importCmd.FindStringSubmatch(string(raw))
			if m == nil {
				t.Fatalf("import.sh has no `terraform import <type>.<name> <id>` line:\n%s", raw)
			}

			if m[1] != resName {
				t.Fatalf("import.sh imports %s, want %s", m[1], resName)
			}

			id := m[3]

			tryImport := func(id string) error {
				d := res.Data(nil)
				d.SetId(id)

				var err error
				if res.Importer.StateContext != nil {
					_, err = res.Importer.StateContext(context.Background(), d, nil)
				} else {
					_, err = res.Importer.State(d, nil) //nolint:staticcheck
				}

				return err
			}

			if apiImporters[resName] {
				if strings.Contains(id, ".") {
					t.Errorf("%s needs the API to import, so its ID should be a single slug, got %q", resName, id)
				}

				return
			}

			if err := tryImport(id); err != nil {
				t.Fatalf("importer rejected documented ID %q: %v", id, err)
			}

			if idx := strings.LastIndex(id, "."); idx >= 0 {
				if err := tryImport(id[:idx]); err == nil {
					t.Errorf("importer accepted %q, so documented ID %q has too many segments", id[:idx], id)
				}
			}
		})
	}
}
