package cloudsmith

import (
	"context"
	"errors"
	"fmt"
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

func newProviderConfig(apiHost string, apiKey string, headers map[string]interface{}, userAgent string) (*providerConfig, diag.Diagnostics) {
	if apiKey == "" {
		return nil, diag.FromErr(errMissingCredentials)
	}

	httpClient := http.DefaultClient
	httpClient.Transport = logging.NewSubsystemLoggingHTTPTransport("Cloudsmith", &headerTransport{
		headers: headers,
		rt:      http.DefaultTransport,
	})

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

	req := apiClient.UserApi.UserSelf(auth)
	if _, _, err := apiClient.UserApi.UserSelfExecute(req); err != nil {
		return nil, diag.FromErr(errors.New("invalid API credentials"))
	}

	return &providerConfig{Auth: auth, APIClient: apiClient}, nil
}

func (pc *providerConfig) GetAPIKey() string {
	apiKeys, _ := pc.Auth.Value(cloudsmith.ContextAPIKeys).(map[string]cloudsmith.APIKey)
	return apiKeys["apikey"].Key
}

type headerTransport struct {
	headers map[string]interface{}
	rt      http.RoundTripper
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		req.Header.Add(k, fmt.Sprint(v))
	}
	return t.rt.RoundTrip(req)
}
