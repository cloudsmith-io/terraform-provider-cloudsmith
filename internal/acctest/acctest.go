// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

// Package acctest holds helpers shared by acceptance tests that need the full
// provider. It imports the provider, so only external (_test) test packages of
// the schema packages may import it.
package acctest

import (
	"context"
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/util"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/cloudsmith-io/terraform-provider-cloudsmith/internal/provider"
)

var (
	TestAccProvider  *schema.Provider
	TestAccProviders map[string]*schema.Provider
)

//nolint:gochecknoinits
func init() {
	TestAccProvider = provider.Provider()
	TestAccProviders = map[string]*schema.Provider{
		"cloudsmith": TestAccProvider,
	}
}

func TestAccPreCheck(t *testing.T) {
	if v := os.Getenv("CLOUDSMITH_API_KEY"); v == "" {
		t.Fatal("CLOUDSMITH_API_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDSMITH_NAMESPACE"); v == "" {
		t.Fatal("CLOUDSMITH_NAMESPACE must be set for acceptance tests")
	}
}

// TestAccAPIClient builds a util.ProviderConfig directly from the acceptance test
// environment variables, bypassing acctest.TestAccProvider.Meta(): the config-building
// functions below run while Terraform is generating the test's HCL, before
// resource.Test invokes the provider's own ConfigureContextFunc, so
// acctest.TestAccProvider.Meta() is still nil at that point.
func TestAccAPIClient(t *testing.T) *util.ProviderConfig {
	apiHost := os.Getenv("CLOUDSMITH_API_HOST")
	if apiHost == "" {
		apiHost = "https://api.cloudsmith.io/v1"
	}

	pc, diags := provider.NewStaticTokenProviderConfig(
		context.Background(),
		apiHost,
		os.Getenv("CLOUDSMITH_API_KEY"),
		nil,
		"terraform-provider-cloudsmith-acctest",
	)
	if diags.HasError() {
		t.Fatalf("error building API client for acceptance test setup: %v", diags)
	}

	return pc
}
