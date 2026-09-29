package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccDataSourceTeamMembers_basic explicitly empties the team so the result
// does not depend on whether the authenticated creator is auto-added.
func TestAccDataSourceTeamMembers_basic(t *testing.T) {
	t.Parallel()
	teamName := acctest.RandomWithPrefix("tfacc-team-members")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTeamCheckDestroy("cloudsmith_team.example"),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceTeamMembersConfig(teamName),
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("cloudsmith_team.example"),
					resource.TestCheckResourceAttrSet("data.cloudsmith_team_members.test", "id"),
					resource.TestCheckResourceAttrPair("data.cloudsmith_team_members.test", "organization", "cloudsmith_team.example", "organization"),
					resource.TestCheckResourceAttr("data.cloudsmith_team_members.test", "team_name", teamName),
					resource.TestCheckResourceAttr("cloudsmith_manage_team.example", "members.#", "0"),
					resource.TestCheckResourceAttr("data.cloudsmith_team_members.test", "members.#", "0"),
				),
			},
		},
	})
}

func testAccDataSourceTeamMembersConfig(teamName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_team" "example" {
  name         = %q
  slug         = %q
  organization = "%s"
  description  = "Acceptance test team members"
  visibility   = "Visible"
}

resource "cloudsmith_manage_team" "example" {
  organization = cloudsmith_team.example.organization
  team_name    = cloudsmith_team.example.slug
  members      = []
}

data "cloudsmith_team_members" "test" {
  organization = cloudsmith_team.example.organization
  team_name    = cloudsmith_team.example.slug
  depends_on   = [cloudsmith_manage_team.example]
}
`, teamName, teamName, os.Getenv("CLOUDSMITH_NAMESPACE"))
}

func TestDataSourceTeamMembersRead(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		response string
		want     []interface{}
	}{
		{
			name:     "populated",
			response: `{"members":[{"user":"alice","role":"Manager"},{"user":"bob","role":"Member"}]}`,
			want: []interface{}{
				map[string]interface{}{"user": "alice", "role": "Manager"},
				map[string]interface{}{"user": "bob", "role": "Member"},
			},
		},
		{
			name:     "empty",
			response: `{"members":[]}`,
			want:     []interface{}{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/orgs/example-org/teams/example-team/members" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tc.response)
			}))
			defer server.Close()

			config := cloudsmith.NewConfiguration()
			config.Servers = cloudsmith.ServerConfigurations{{URL: server.URL}}
			pc := &providerConfig{
				Auth:      context.Background(),
				APIClient: cloudsmith.NewAPIClient(config),
			}
			d := schema.TestResourceDataRaw(t, dataSourceTeamMembers().Schema, map[string]interface{}{
				"organization": "example-org",
				"team_name":    "example-team",
			})
			// An empty response must clear previously populated state, not leave stale members.
			if err := d.Set("members", []interface{}{
				map[string]interface{}{"user": "previous-member", "role": "Manager"},
			}); err != nil {
				t.Fatal(err)
			}

			if err := dataSourceTeamMembersRead(d, pc); err != nil {
				t.Fatal(err)
			}
			if d.Id() == "" {
				t.Error("expected a data source ID, including for an empty team")
			}
			if got := d.Get("members"); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("members = %#v, want %#v", got, tc.want)
			}
			if got := d.Get("organization"); got != "example-org" {
				t.Errorf("organization = %q, want example-org", got)
			}
			if got := d.Get("team_name"); got != "example-team" {
				t.Errorf("team_name = %q, want example-team", got)
			}
		})
	}
}

func TestDataSourceTeamMembersEmptyFixture(t *testing.T) {
	t.Parallel()

	diagnostics := Provider().ValidateResource("cloudsmith_manage_team", terraform.NewResourceConfigRaw(map[string]interface{}{
		"organization": "example-org",
		"team_name":    "example-team",
		"members":      []interface{}{},
	}))
	if diagnostics.HasError() {
		t.Fatalf("empty membership fixture is invalid: %v", diagnostics)
	}
}
