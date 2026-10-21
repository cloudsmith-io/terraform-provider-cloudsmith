// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:testpackage
package cloudsmith

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func privilegeWithTeam(privilege, team string) cloudsmith.RepositoryPrivilegeDict {
	p := cloudsmith.RepositoryPrivilegeDict{}
	p.SetPrivilege(privilege)
	p.SetTeam(team)
	return p
}

func TestContainsAdminTeam(t *testing.T) {
	service := cloudsmith.RepositoryPrivilegeDict{}
	service.SetPrivilege("Admin")
	service.SetService("svc")

	cases := map[string]struct {
		privs []cloudsmith.RepositoryPrivilegeDict
		want  bool
	}{
		"no privileges":            {nil, false},
		"read team only":           {[]cloudsmith.RepositoryPrivilegeDict{privilegeWithTeam("Read", "t")}, false},
		"admin team":               {[]cloudsmith.RepositoryPrivilegeDict{privilegeWithTeam("Admin", "t")}, true},
		"admin service, no team":   {[]cloudsmith.RepositoryPrivilegeDict{service}, false},
		"read team plus admin one": {[]cloudsmith.RepositoryPrivilegeDict{privilegeWithTeam("Read", "a"), privilegeWithTeam("Admin", "b")}, true},
	}
	for name, tc := range cases {
		if got := containsAdminTeam(tc.privs); got != tc.want {
			t.Errorf("%s: got %v, want %v", name, got, tc.want)
		}
	}
}

func TestSetContainsAdminTeam(t *testing.T) {
	teamSchema := resourceRepositoryPrivileges().Schema["team"].Elem.(*schema.Resource)
	newSet := func(privs ...string) *schema.Set {
		set := schema.NewSet(schema.HashResource(teamSchema), []interface{}{})
		for i, p := range privs {
			set.Add(map[string]interface{}{"privilege": p, "slug": string(rune('a' + i))})
		}
		return set
	}

	if setContainsAdminTeam(nil) {
		t.Error("nil set should not contain an admin team")
	}
	if setContainsAdminTeam(newSet("Read", "Write")) {
		t.Error("non-admin teams should not satisfy the check")
	}
	if !setContainsAdminTeam(newSet("Read", "Admin")) {
		t.Error("admin team should satisfy the check")
	}
}

// TestRepositoryPrivileges_LockoutPlanValidation drives the resource diff (and
// so CustomizeDiff) with a team-only configuration that does not name the
// authenticated account, including a team slug that is unknown at plan time.
func TestRepositoryPrivileges_LockoutPlanValidation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"slug":"caller"}`))
	}))
	defer server.Close()
	pc := testPrivilegesProviderConfig(server)

	const unknownSlug = "74D93920-ED26-11E3-AC10-0800200C9A66"
	for name, slug := range map[string]string{"known slug": "my-team", "unknown slug": unknownSlug} {
		for privilege, wantErr := range map[string]bool{"Read": true, "Write": true, "Admin": false} {
			t.Run(name+"/"+privilege, func(t *testing.T) {
				cfg := terraform.NewResourceConfigRaw(map[string]interface{}{
					"organization": "o",
					"repository":   "r",
					"service":      []interface{}{map[string]interface{}{"slug": "other", "privilege": "Read"}},
					"team":         []interface{}{map[string]interface{}{"slug": slug, "privilege": privilege}},
				})
				_, err := resourceRepositoryPrivileges().Diff(context.Background(), nil, cfg, pc)
				if wantErr {
					if err == nil || !strings.Contains(err.Error(), "with Admin privilege") {
						t.Fatalf("error = %v, want lockout error mentioning Admin privilege", err)
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		}
	}
}
