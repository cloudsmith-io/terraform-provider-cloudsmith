package main

import (
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/cloudsmith"
	"github.com/hashicorp/terraform-plugin-sdk/plugin"

	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider { return cloudsmith.Provider() },
	})
}
