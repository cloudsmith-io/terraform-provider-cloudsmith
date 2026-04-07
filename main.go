package main

import (
	"flag"

	"github.com/cloudsmith-io/terraform-provider-cloudsmith/cloudsmith"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: cloudsmith.Provider,
		ProviderAddr: "registry.terraform.io/cloudsmith-io/cloudsmith",
		Debug:        debugMode,
	})
}
