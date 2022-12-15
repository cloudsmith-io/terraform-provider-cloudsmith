// Package cloudsmith implements a Terraform provider for interacting with Cloudsmith.
package cloudsmith

import (
	"context"
	"fmt"
	"runtime"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider returns a terraform.ResourceProvider.
func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Description: "The API key for authenticating with the Cloudsmith API.",
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSMITH_API_KEY", nil),
				Sensitive:   true,
			},
			"api_host": {
				Type:        schema.TypeString,
				Description: "The API host to connect to (mostly useful for testing).",
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSMITH_API_HOST", "https://api.cloudsmith.io/v1"),
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"cloudsmith_namespace":    dataSourceNamespace(),
			"cloudsmith_package_list": dataSourcePackageList(),
			"cloudsmith_repository":   dataSourceRepository(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"cloudsmith_entitlement": resourceEntitlement(),
			"cloudsmith_repository":  resourceRepository(),
			"cloudsmith_webhook":     resourceWebhook(),
		},
	}

	p.ConfigureContextFunc = func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}

		apiHost := requiredString(d, "api_host")
		apiKey := requiredString(d, "api_key")
		userAgent := fmt.Sprintf("(%s %s) Terraform/%s", runtime.GOOS, runtime.GOARCH, terraformVersion)

		return newProviderConfig(apiHost, apiKey, userAgent)
	}

	return p
}
