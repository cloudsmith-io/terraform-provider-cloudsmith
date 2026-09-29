package cloudsmith

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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
		Steps:        testAccDataSourceTeamMembersSteps(teamName, os.Getenv("CLOUDSMITH_NAMESPACE"), testAccProvider),
	})
}

func testAccDataSourceTeamMembersSteps(teamName, organization string, provider *schema.Provider) []resource.TestStep {
	teamConfig := testAccDataSourceTeamMembersConfig(teamName, organization)
	return []resource.TestStep{
		{
			Config: teamConfig,
			Check: func(s *terraform.State) error {
				team, ok := s.RootModule().Resources["cloudsmith_team.example"]
				if !ok || team.Primary == nil || team.Primary.ID == "" {
					return fmt.Errorf("owned team missing from state")
				}
				org, slug := team.Primary.Attributes["organization"], team.Primary.Attributes["slug"]
				if org != organization || slug != teamName {
					return fmt.Errorf("refusing to empty unexpected team %q/%q", org, slug)
				}
				// The required members block cannot represent an empty team in HCL.
				// Clear only the team created in this step, before adding the data source.
				pc := provider.Meta().(*providerConfig)
				req := pc.APIClient.OrgsApi.OrgsTeamsMembersUpdate(pc.Auth, org, slug).
					Data(cloudsmith.OrganizationTeamMembers{Members: []cloudsmith.OrganizationTeamServiceMember{}})
				if _, _, err := pc.APIClient.OrgsApi.OrgsTeamsMembersUpdateExecute(req); err != nil {
					return fmt.Errorf("emptying owned team %s/%s: %w", org, slug, err)
				}
				return nil
			},
		},
		{
			Config: teamConfig + `
data "cloudsmith_team_members" "test" {
  organization = cloudsmith_team.example.organization
  team_name    = cloudsmith_team.example.slug
}
`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.cloudsmith_team_members.test", "id"),
				resource.TestCheckResourceAttrPair("data.cloudsmith_team_members.test", "organization", "cloudsmith_team.example", "organization"),
				resource.TestCheckResourceAttr("data.cloudsmith_team_members.test", "team_name", teamName),
				resource.TestCheckResourceAttr("data.cloudsmith_team_members.test", "members.#", "0"),
			),
		},
	}
}

func testAccDataSourceTeamMembersConfig(teamName, organization string) string {
	return fmt.Sprintf(`
resource "cloudsmith_team" "example" {
  name         = %q
  slug         = %q
  organization = %q
  description  = "Acceptance test team members"
  visibility   = "Visible"
}

`, teamName, teamName, organization)
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

	for _, creator := range []string{"human", "service"} {
		t.Run(creator, func(t *testing.T) {
			t.Parallel()
			const teamName = "example-team"
			var mu sync.Mutex
			var exists, cleared bool
			var reads, deletions int
			members := []cloudsmith.OrganizationTeamServiceMember{}
			if creator == "human" {
				members = append(members, cloudsmith.OrganizationTeamServiceMember{User: "creator", Role: "Manager"})
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				path := strings.TrimSuffix(r.URL.Path, "/")
				switch {
				case r.Method == http.MethodPost && path == "/orgs/example-org/teams":
					exists = true
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"name":"example-team","slug":"example-team","slug_perm":"team-id","description":"Acceptance test team members","visibility":"Visible"}`)
				case r.Method == http.MethodGet && path == "/orgs/example-org/teams/team-id":
					if !exists {
						http.Error(w, `{"detail":"Not found"}`, http.StatusNotFound)
						return
					}
					fmt.Fprint(w, `{"name":"example-team","slug":"example-team","slug_perm":"team-id","description":"Acceptance test team members","visibility":"Visible"}`)
				case r.Method == http.MethodPut && path == "/orgs/example-org/teams/example-team/members":
					var body cloudsmith.OrganizationTeamMembers
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode membership replacement: %v", err)
						http.Error(w, "invalid body", http.StatusBadRequest)
						return
					}
					if !exists || body.Members == nil || len(body.Members) != 0 {
						t.Errorf("expected explicit empty replacement on an existing team, got %#v (exists=%t)", body.Members, exists)
						http.Error(w, "invalid replacement", http.StatusBadRequest)
						return
					}
					members = body.Members
					cleared = true
					fmt.Fprint(w, `{"members":[]}`)
				case r.Method == http.MethodGet && path == "/orgs/example-org/teams/example-team/members":
					if !exists || !cleared {
						t.Error("data source read before owned team was created and explicitly emptied")
						http.Error(w, "fixture not ready", http.StatusBadRequest)
						return
					}
					reads++
					if err := json.NewEncoder(w).Encode(cloudsmith.OrganizationTeamMembers{Members: members}); err != nil {
						t.Errorf("encode members: %v", err)
					}
				case r.Method == http.MethodDelete && path == "/orgs/example-org/teams/team-id":
					exists = false
					deletions++
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusBadRequest)
				}
			}))
			defer server.Close()
			config := cloudsmith.NewConfiguration()
			config.Servers = cloudsmith.ServerConfigurations{{URL: server.URL}}
			pc := &providerConfig{Auth: context.Background(), APIClient: cloudsmith.NewAPIClient(config)}
			provider := Provider()
			provider.ConfigureContextFunc = func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
				return pc, nil
			}
			resource.UnitTest(t, resource.TestCase{
				Providers: map[string]*schema.Provider{"cloudsmith": provider},
				Steps:     testAccDataSourceTeamMembersSteps(teamName, "example-org", provider),
			})
			mu.Lock()
			defer mu.Unlock()
			if exists || !cleared || reads == 0 || deletions != 1 {
				t.Errorf("incomplete lifecycle: exists=%t cleared=%t reads=%d deletions=%d", exists, cleared, reads, deletions)
			}
		})
	}
}
