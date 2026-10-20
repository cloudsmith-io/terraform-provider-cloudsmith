// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:testpackage
package cloudsmith

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const ConnectedResourceName string = "cloudsmith_repository_connected.test"

// TestAccRepositoryConnected_basic creates a source repository and a target
// repository, connects them, verifies the connection exists, updates the
// priority/is_active fields, imports the resource and finally verifies the
// connection is cleaned up.
func TestAccRepositoryConnected_basic(t *testing.T) {
	t.Parallel()

	sourceName := testAccUniqueRepositoryName("tf-acc-connected-src")
	targetName := testAccUniqueRepositoryName("tf-acc-connected-tgt")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccRepositoryConnectedCheckDestroy(ConnectedResourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryConnectedConfigCreate(sourceName, targetName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryConnectedCheckExists(ConnectedResourceName),
					resource.TestCheckResourceAttr(ConnectedResourceName, "is_active", "true"),
					resource.TestCheckResourceAttr(ConnectedResourceName, "priority", "1"),
					resource.TestCheckResourceAttrSet(ConnectedResourceName, "slug_perm"),
					resource.TestCheckResourceAttrSet(ConnectedResourceName, "created_at"),
				),
			},
			{
				Config: testAccRepositoryConnectedConfigUpdate(sourceName, targetName),
				Check: resource.ComposeTestCheckFunc(
					testAccRepositoryConnectedCheckExists(ConnectedResourceName),
					resource.TestCheckResourceAttr(ConnectedResourceName, "is_active", "false"),
					resource.TestCheckResourceAttr(ConnectedResourceName, "priority", "5"),
				),
			},
			{
				ResourceName: ConnectedResourceName,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					resourceState := s.RootModule().Resources[ConnectedResourceName]
					return fmt.Sprintf(
						"%s.%s.%s",
						resourceState.Primary.Attributes["namespace"],
						resourceState.Primary.Attributes["repository"],
						resourceState.Primary.Attributes["slug_perm"],
					), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

type repositoryConnectedFixture struct {
	t       *testing.T
	mu      sync.Mutex
	state   map[string]interface{}
	patches []map[string]interface{}
}

func (f *repositoryConnectedFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	const collection = "/repos/fixture-org/fixture-src/connected/"
	const item = collection + "conn-perm/"

	switch {
	case r.Method == http.MethodPost && r.URL.Path == collection:
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			f.t.Errorf("decode create body: %v", err)
		}
		f.state = map[string]interface{}{
			"slug_perm":         "conn-perm",
			"target_repository": body["target_repository"],
			"is_active":         body["is_active"],
			"priority":          body["priority"],
			"created_at":        "2026-01-01T00:00:00Z",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
	case r.Method == http.MethodGet && r.URL.Path == item:
		w.Header().Set("Content-Type", "application/json")
	case r.Method == http.MethodPatch && r.URL.Path == item:
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			f.t.Errorf("decode patch body: %v", err)
		}
		f.patches = append(f.patches, body)
		for k, v := range body {
			f.state[k] = v
		}
		w.Header().Set("Content-Type", "application/json")
	default:
		f.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(f.state); err != nil {
		f.t.Errorf("encode connected repository: %v", err)
	}
}

// TestRepositoryConnectedUpdateSendsOnlyChangedFields guards against the
// generated PATCH constructor defaulting is_active to false: changing only
// priority must not deactivate the connection, and vice versa.
func TestRepositoryConnectedUpdateSendsOnlyChangedFields(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		isActive  bool
		priority  int
		wantPatch map[string]interface{}
	}{
		{"priority only", true, 2, map[string]interface{}{"priority": float64(2)}},
		{"is_active only", false, 1, map[string]interface{}{"is_active": false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := &repositoryConnectedFixture{t: t}
			server := httptest.NewServer(fixture)
			defer server.Close()
			pc := testAPIKeyRuleProvider(server)
			res := resourceRepositoryConnected()

			var state *terraform.InstanceState
			for step, configMap := range []map[string]interface{}{
				{Namespace: "fixture-org", Repository: "fixture-src", TargetRepository: "fixture-tgt", IsActive: true, Priority: 1},
				{Namespace: "fixture-org", Repository: "fixture-src", TargetRepository: "fixture-tgt", IsActive: tc.isActive, Priority: tc.priority},
			} {
				config := terraform.NewResourceConfigRaw(configMap)
				diff, err := res.Diff(context.Background(), state, config, pc)
				if err != nil {
					t.Fatal(err)
				}
				applied, diagnostics := res.Apply(context.Background(), state, diff, pc)
				if diagnostics.HasError() {
					t.Fatal(diagnostics)
				}
				state, diagnostics = res.RefreshWithoutUpgrade(context.Background(), applied, pc)
				if diagnostics.HasError() {
					t.Fatal(diagnostics)
				}
				if got, want := state.Attributes[IsActive], fmt.Sprint(configMap[IsActive]); got != want {
					t.Errorf("step %d: is_active = %s, want %s", step, got, want)
				}
				if got, want := state.Attributes[Priority], fmt.Sprint(configMap[Priority]); got != want {
					t.Errorf("step %d: priority = %s, want %s", step, got, want)
				}
				diff, err = res.Diff(context.Background(), state, config, pc)
				if err != nil || !diff.Empty() {
					t.Errorf("step %d: nonempty post-refresh diff=%#v err=%v", step, diff, err)
				}
			}

			fixture.mu.Lock()
			defer fixture.mu.Unlock()
			if len(fixture.patches) != 1 {
				t.Fatalf("PATCH requests = %d, want 1", len(fixture.patches))
			}
			if !reflect.DeepEqual(fixture.patches[0], tc.wantPatch) {
				t.Errorf("PATCH body = %v, want %v", fixture.patches[0], tc.wantPatch)
			}
		})
	}
}

func testAccRepositoryConnectedConfigCreate(sourceName, targetName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "source" {
    name      = "%s"
    namespace = "%s"
}

resource "cloudsmith_repository" "target" {
    name      = "%s"
    namespace = "%s"
}

resource "cloudsmith_repository_connected" "test" {
    namespace         = resource.cloudsmith_repository.source.namespace
    repository        = resource.cloudsmith_repository.source.slug_perm
    target_repository = resource.cloudsmith_repository.target.slug
    is_active         = true
    priority          = 1
}
`, sourceName, testAccNamespace(), targetName, testAccNamespace())
}

func testAccRepositoryConnectedConfigUpdate(sourceName, targetName string) string {
	return fmt.Sprintf(`
resource "cloudsmith_repository" "source" {
    name      = "%s"
    namespace = "%s"
}

resource "cloudsmith_repository" "target" {
    name      = "%s"
    namespace = "%s"
}

resource "cloudsmith_repository_connected" "test" {
    namespace         = resource.cloudsmith_repository.source.namespace
    repository        = resource.cloudsmith_repository.source.slug_perm
    target_repository = resource.cloudsmith_repository.target.slug
    is_active         = false
    priority          = 5
}
`, sourceName, testAccNamespace(), targetName, testAccNamespace())
}

//nolint:err113
func testAccRepositoryConnectedCheckDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		namespace := resourceState.Primary.Attributes["namespace"]
		repository := resourceState.Primary.Attributes["repository"]
		slugPerm := resourceState.Primary.Attributes["slug_perm"]

		req := pc.APIClient.ReposApi.ReposConnectedRead(pc.Auth, namespace, repository, slugPerm)
		_, resp, err := pc.APIClient.ReposApi.ReposConnectedReadExecute(req)
		if err != nil && !is404(resp) {
			return fmt.Errorf("unable to verify connected repository deletion: %w", err)
		} else if is200(resp) {
			return fmt.Errorf("unable to verify connected repository deletion: still exists: %s/%s/%s", namespace, repository, slugPerm)
		}
		if resp != nil {
			defer resp.Body.Close()
		}

		return nil
	}
}

//nolint:err113
func testAccRepositoryConnectedCheckExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		pc := testAccProvider.Meta().(*providerConfig)

		namespace := resourceState.Primary.Attributes["namespace"]
		repository := resourceState.Primary.Attributes["repository"]
		slugPerm := resourceState.Primary.ID

		req := pc.APIClient.ReposApi.ReposConnectedRead(pc.Auth, namespace, repository, slugPerm)
		_, resp, err := pc.APIClient.ReposApi.ReposConnectedReadExecute(req)
		if err != nil {
			return fmt.Errorf("unable to verify connected repository existence: %w", err)
		}
		defer resp.Body.Close()

		return nil
	}
}
