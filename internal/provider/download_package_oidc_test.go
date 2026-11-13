// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:testpackage
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cloudsmithpackage "github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/schema/pkg"
)

func TestDownloadPackageUsesOIDCExchangeToken(t *testing.T) {
	t.Parallel()

	const (
		assertion      = "tfc-workload-identity-token"
		exchangedToken = "cloudsmith-oidc-token"
		packageBody    = "package contents"
	)

	var exchanges atomic.Int32
	authorization := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/openid/test-org/":
			exchanges.Add(1)
			var payload struct {
				OIDCToken   string `json:"oidc_token"`
				ServiceSlug string `json:"service_slug"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if payload.OIDCToken != assertion || payload.ServiceSlug != "terraform-cloud" {
				http.Error(w, "unexpected exchange payload", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"token":%q}`, exchangedToken)
		case r.Method == http.MethodGet && (strings.TrimSuffix(r.URL.Path, "/") == "/user/self" || strings.TrimSuffix(r.URL.Path, "/") == "/v1/user/self"):
			if r.Header.Get("X-Api-Key") != exchangedToken {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"email":"sa@example.com","name":"tfc","slug":"tfc","slug_perm":"tfc"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/package.tar.gz":
			authorization <- r.Header.Get("Authorization")
			fmt.Fprint(w, packageBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	tokens := &oidcTokenSource{
		identity: oidcIdentity{organization: "test-org", serviceSlug: "terraform-cloud"},
		apiHost:  server.URL,
		getenv: func(key string) string {
			if key == "TFC_WORKLOAD_IDENTITY_TOKEN" {
				return assertion
			}
			return ""
		},
		now: time.Now,
	}
	config, diags := newProviderConfig(context.Background(), server.URL, tokens, nil, "test-agent")
	if diags.HasError() {
		t.Fatalf("configure provider: %v", diags)
	}

	downloaded, err := cloudsmithpackage.DownloadPackage(server.URL+"/package.tar.gz", t.TempDir(), config, false)
	if err != nil {
		t.Fatalf("download package: %v", err)
	}
	if got := <-authorization; got != "Token "+exchangedToken {
		t.Fatalf("Authorization = %q, want %q", got, "Token "+exchangedToken)
	}
	if got := exchanges.Load(); got != 1 {
		t.Fatalf("OIDC exchanges = %d, want 1", got)
	}
	contents, err := os.ReadFile(downloaded)
	if err != nil {
		t.Fatalf("read downloaded package: %v", err)
	}
	if got := string(contents); got != packageBody {
		t.Fatalf("downloaded contents = %q, want %q", got, packageBody)
	}
}
