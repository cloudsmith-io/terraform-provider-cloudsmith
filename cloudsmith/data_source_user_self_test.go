// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package cloudsmith

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccUserSelf_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccUserSelfConfig,
				Check:  testAccUserSelfCheck("data.cloudsmith_user_self.test", testAccProvider),
			},
		},
	})
}

func testAccUserSelfCheck(name string, p *schema.Provider) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttrSet(name, "name"),
		resource.TestCheckResourceAttrSet(name, "slug"),
		resource.TestCheckResourceAttrSet(name, "slug_perm"),
		func(s *terraform.State) error {
			pc, ok := p.Meta().(*providerConfig)
			if !ok || pc == nil {
				return fmt.Errorf("provider is not configured")
			}
			userSelf, _, err := pc.APIClient.UserApi.UserSelf(pc.Auth).Execute()
			if err != nil {
				return fmt.Errorf("retrieving authenticated account: %w", err)
			}

			// Service accounts may have no email; state must still match the API.
			return resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(name, "email", userSelf.GetEmail()),
				resource.TestCheckResourceAttr(name, "name", userSelf.GetName()),
				resource.TestCheckResourceAttr(name, "slug", userSelf.GetSlug()),
				resource.TestCheckResourceAttr(name, "slug_perm", userSelf.GetSlugPerm()),
				resource.TestCheckResourceAttr(name, "id", userSelf.GetSlugPerm()),
			)(s)
		},
	)
}

func TestUserSelfDataSource_IdentityTypes(t *testing.T) {
	t.Setenv("CLOUDSMITH_USE_OIDC", "false")

	tests := []struct {
		name         string
		response     string
		wantEmail    string
		wantName     string
		wantSlug     string
		wantSlugPerm string
	}{
		{
			name:         "human",
			response:     `{"authenticated":true,"email":"person@example.com","name":"Test Person","slug":"test-person","slug_perm":"human123"}`,
			wantEmail:    "person@example.com",
			wantName:     "Test Person",
			wantSlug:     "test-person",
			wantSlugPerm: "human123",
		},
		{
			name:         "service account with empty email",
			response:     `{"authenticated":true,"email":"","name":"Test Service","slug":"test-service","slug_perm":"service123"}`,
			wantName:     "Test Service",
			wantSlug:     "test-service",
			wantSlugPerm: "service123",
		},
		{
			name:         "service account with null email",
			response:     `{"authenticated":true,"email":null,"name":"Test Service","slug":"test-service","slug_perm":"service123"}`,
			wantName:     "Test Service",
			wantSlug:     "test-service",
			wantSlugPerm: "service123",
		},
		{
			name:         "service account with omitted email",
			response:     `{"authenticated":true,"name":"Test Service","slug":"test-service","slug_perm":"service123"}`,
			wantName:     "Test Service",
			wantSlug:     "test-service",
			wantSlugPerm: "service123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/user/self/" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusNotFound)
					return
				}
				if r.Header.Get("X-Api-Key") != "test-user-self-token" {
					t.Error("request did not use the test API key")
					http.Error(w, "invalid API key", http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := fmt.Fprintln(w, tc.response); err != nil {
					t.Errorf("writing user/self response: %v", err)
				}
			}))
			t.Cleanup(server.Close)

			p := Provider()
			const name = "data.cloudsmith_user_self.test"
			resource.UnitTest(t, resource.TestCase{
				ProviderFactories: map[string]func() (*schema.Provider, error){
					"cloudsmith": func() (*schema.Provider, error) { return p, nil },
				},
				Steps: []resource.TestStep{
					{
						Config: fmt.Sprintf(`
provider "cloudsmith" {
  api_host = %q
  api_key  = "test-user-self-token"
}
%s
`, server.URL, testAccUserSelfConfig),
						Check: resource.ComposeTestCheckFunc(
							testAccUserSelfCheck(name, p),
							resource.TestCheckResourceAttr(name, "email", tc.wantEmail),
							resource.TestCheckResourceAttr(name, "name", tc.wantName),
							resource.TestCheckResourceAttr(name, "slug", tc.wantSlug),
							resource.TestCheckResourceAttr(name, "slug_perm", tc.wantSlugPerm),
							resource.TestCheckResourceAttr(name, "id", tc.wantSlugPerm),
						),
					},
				},
			})
		})
	}
}

const testAccUserSelfConfig = `
data "cloudsmith_user_self" "test" {}
`
