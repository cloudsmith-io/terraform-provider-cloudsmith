package main

import (
	"github.com/cloudsmith-io/terraform-provider-cloudsmith/cloudsmith"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: cloudsmith.Provider,
	})
}
