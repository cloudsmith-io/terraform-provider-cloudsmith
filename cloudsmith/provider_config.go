package cloudsmith

import (
	"context"
	"errors"
	"net/http"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/helper/logging"
)

var errMissingCredentials = errors.New("credentials required for Cloudsmith provider")

type providerConfig struct {
	// authentication credentials for the configured user
	Auth context.Context

	// initialised Cloudsmith API client
	APIClient *cloudsmith.APIClient
}

func newProviderConfig(apiHost, apiKey, userAgent string) (*providerConfig, error) {
	if apiKey == "" {
		return nil, errMissingCredentials
	}

	httpClient := http.DefaultClient
	httpClient.Transport = logging.NewTransport("Cloudsmith", http.DefaultTransport)

	config := cloudsmith.NewConfiguration()
	config.BasePath = apiHost
	config.Debug = logging.IsDebugOrHigher()
	config.HTTPClient = httpClient
	config.UserAgent = userAgent

	apiClient := cloudsmith.NewAPIClient(config)
	auth := context.WithValue(
		context.Background(),
		cloudsmith.ContextAPIKey,
		cloudsmith.APIKey{Key: apiKey},
	)

	return &providerConfig{Auth: auth, APIClient: apiClient}, nil
}
