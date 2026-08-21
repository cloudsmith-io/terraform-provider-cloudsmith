package cloudsmith

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	cloudsmithv2 "github.com/cloudsmith-io/cloudsmith-go-v2"
	"github.com/cloudsmith-io/cloudsmith-go-v2/models/components"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

type providerConfig struct {
	Auth context.Context

	APIClient *cloudsmith.APIClient

	V2ApiClient *cloudsmithv2.Cloudsmith

	tokens tokenSource
}

func newProviderConfig(ctx context.Context, apiHost string, tokens tokenSource, headers map[string]interface{}, userAgent string) (*providerConfig, diag.Diagnostics) {
	if tokens == nil {
		return nil, diag.FromErr(errMissingCredentials)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	apiKey, err := tokens.Token(ctx)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	if apiKey == "" {
		return nil, diag.FromErr(errMissingCredentials)
	}

	credHost, err := credentialHost(apiHost)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	httpClient := &http.Client{
		Transport: logging.NewSubsystemLoggingHTTPTransport("Cloudsmith", &headerTransport{
			headers: headers,
			rt: &apiKeyTransport{
				tokens:  tokens,
				apiHost: credHost,
				rt:      http.DefaultTransport,
			},
		}),
	}

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
		return nil, diag.FromErr(fmt.Errorf("invalid API credentials: %w", err))
	}

	v2Options := []cloudsmithv2.SDKOption{
		cloudsmithv2.WithClient(httpClient),
		cloudsmithv2.WithSecuritySource(func(ctx context.Context) (components.Security, error) {
			tok, err := tokens.Token(ctx)
			if err != nil {
				return components.Security{}, err
			}
			return components.Security{Apikey: tok}, nil
		}),
	}
	if apiHost != "" {
		v2Options = append(v2Options, cloudsmithv2.WithServerURL(apiHost))
	}

	return &providerConfig{
		Auth:        auth,
		APIClient:   apiClient,
		V2ApiClient: cloudsmithv2.New(v2Options...),
		tokens:      tokens,
	}, nil
}

// GetAPIKey returns the current credential. A token-source failure is returned
// rather than swallowed: falling back to the token minted at configure time
// would send a known-expired credential and report the resulting rejection
// instead of the refresh error that caused it.
func (pc *providerConfig) GetAPIKey() (string, error) {
	if pc.tokens != nil {
		return pc.tokens.Token(context.Background())
	}
	apiKeys, _ := pc.Auth.Value(cloudsmith.ContextAPIKeys).(map[string]cloudsmith.APIKey)
	return apiKeys["apikey"].Key, nil
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

type apiKeyTransport struct {
	tokens  tokenSource
	apiHost string
	rt      http.RoundTripper
}

// credentialHost is the only host apiKeyTransport will send the Cloudsmith
// credential to. An empty api_host yields an empty host, which disables the
// restriction and keeps callers that pass no api_host working. A non-empty
// api_host that is not absolute is rejected here rather than left to fail later
// as an opaque "unsupported protocol scheme" from the round tripper.
func credentialHost(apiHost string) (string, error) {
	if apiHost == "" {
		return "", nil
	}
	parsed, err := url.Parse(apiHost)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errInvalidAPIHost
	}
	return parsed.Host, nil
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// The same client serves package downloads, which go to dl.cloudsmith.io
	// and then redirect to third-party object storage. RoundTrip runs once per
	// redirect hop, so without this check the credential would be re-stamped
	// onto the storage host too and land in its access logs.
	if t.apiHost != "" && req.URL.Host != t.apiHost {
		// The generated SDK sets X-Api-Key above this transport, and Go copies
		// caller-set headers across redirect hops. Only Authorization and a few
		// others are stripped on a cross-host redirect, so drop the credential
		// here as well. Authorization is left alone: package downloads send it
		// to the CDN deliberately.
		req.Header.Del("X-Api-Key")
		return t.rt.RoundTrip(req)
	}
	used, err := t.setAPIKey(req)
	if err != nil {
		return nil, err
	}
	resp, err := t.rt.RoundTrip(req)
	if err != nil || resp == nil || resp.StatusCode != http.StatusUnauthorized || t.tokens == nil || !t.tokens.canRetryAuth() {
		return resp, err
	}
	retry, cloneErr := cloneHTTPRequest(req)
	if cloneErr != nil {
		// Cannot retry this one, but the token is still known-bad: drop it so
		// the next request refreshes instead of reusing a dead token.
		t.tokens.invalidate(used)
		return resp, err
	}
	resp.Body.Close()
	t.tokens.invalidate(used)
	if _, err := t.setAPIKey(retry); err != nil {
		return nil, err
	}
	return t.rt.RoundTrip(retry)
}

// setAPIKey stamps the current token on the request and reports which token was
// used, so a 401 retry only invalidates that token and not a fresher one.
func (t *apiKeyTransport) setAPIKey(req *http.Request) (string, error) {
	if t.tokens == nil {
		return "", nil
	}
	tok, err := t.tokens.Token(req.Context())
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Api-Key", tok)
	return tok, nil
}

func cloneHTTPRequest(req *http.Request) (*http.Request, error) {
	clone := req.Clone(req.Context())
	if req.Body == nil || req.Body == http.NoBody {
		return clone, nil
	}
	if req.GetBody == nil {
		return nil, errCannotReplayRequest
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	clone.Body = body
	return clone, nil
}

var errCannotReplayRequest = errors.New("cannot retry OIDC auth: request body is not replayable")
