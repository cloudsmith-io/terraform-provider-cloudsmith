package cloudsmith

import (
	"context"
	"errors"
	"net/http"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

var errMissingCredentials = errors.New("credentials required for Cloudsmith provider")

type providerConfig struct {
	// authentication credentials for the configured user
	Auth context.Context

	// initialised Cloudsmith API client
	APIClient *cloudsmith.APIClient
}

func newProviderConfig(apiHost, apiKey, userAgent string) (*providerConfig, diag.Diagnostics) {
	if apiKey == "" {
		return nil, diag.FromErr(errMissingCredentials)
	}

	httpClient := http.DefaultClient
	httpClient.Transport = logging.NewSubsystemLoggingHTTPTransport("Cloudsmith", http.DefaultTransport)

	config := cloudsmith.NewConfiguration()
	config.Debug = logging.IsDebugOrHigher()
	config.HTTPClient = httpClient
	config.Servers = cloudsmith.ServerConfigurations{
		{URL: apiHost},
	}
	config.UserAgent = userAgent

	apiClient := cloudsmith.NewAPIClient(config)

	auth := context.WithValue(
		context.Background(),
		cloudsmith.ContextAPIKeys,
		map[string]cloudsmith.APIKey{
			"apikey": {Key: apiKey},
		},
	)

	return &providerConfig{Auth: auth, APIClient: apiClient}, nil
}
