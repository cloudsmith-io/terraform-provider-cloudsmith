// Package cloudsmith implements a Terraform provider for interacting with Cloudsmith.
package cloudsmith

import (
	"fmt"
	"runtime"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
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
				DefaultFunc: schema.EnvDefaultFunc("CLOUDSMITH_API_HOST", "https://api.cloudsmith.io"),
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"cloudsmith_namespace": dataSourceNamespace(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"cloudsmith_repository": resourceRepository(),
		},
	}

	p.ConfigureFunc = func(d *schema.ResourceData) (interface{}, error) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}

		apiHost := d.Get("api_host").(string)
		apiKey := d.Get("api_key").(string)
		userAgent := fmt.Sprintf("(%s %s) Terraform/%s", runtime.GOOS, runtime.GOARCH, terraformVersion)

		return newProviderConfig(apiHost, apiKey, userAgent)
	}

	return p
}
