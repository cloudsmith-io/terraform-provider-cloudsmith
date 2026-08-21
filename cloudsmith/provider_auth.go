package cloudsmith

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	tokenRefreshSkew    = time.Minute
	opaqueTokenLifetime = 15 * time.Minute
	oidcHTTPTimeout     = 30 * time.Second
)

var (
	errMixedCredentials   = errors.New("api_key and oidc cannot be set together")
	errMissingCredentials = errors.New("set api_key, an oidc block, or CLOUDSMITH_USE_OIDC")
	errMissingOIDCToken   = errors.New(
		"oidc requires an HCP Terraform workload identity token in TFC_WORKLOAD_IDENTITY_TOKEN_CLOUDSMITH " +
			"or TFC_WORKLOAD_IDENTITY_TOKEN; set TFC_WORKLOAD_IDENTITY_AUDIENCE (or " +
			"TFC_WORKLOAD_IDENTITY_AUDIENCE_CLOUDSMITH) as an environment variable on the workspace so HCP " +
			"Terraform injects one",
	)
	errIncompleteOIDC     = errors.New("oidc requires organization and service_slug (set the attributes or CLOUDSMITH_ORG and CLOUDSMITH_SERVICE_SLUG)")
	errEmptyExchangeToken = errors.New("OIDC token exchange returned an empty token")
	errInvalidCredential  = errors.New("invalid credential")
	errInvalidAPIHost     = errors.New("api_host must be an absolute URL")
)

// tokenSource hands the HTTP transport a Cloudsmith credential. The static
// api_key path and the OIDC path both go through it, so the transport does not
// need to know which one is in use.
type tokenSource interface {
	Token(ctx context.Context) (string, error)
	invalidate(used string)
	canRetryAuth() bool
}

type staticToken string

func (s staticToken) Token(context.Context) (string, error) { return string(s), nil }
func (s staticToken) invalidate(string)                     {}
func (s staticToken) canRetryAuth() bool                    { return false }

type oidcTokenSource struct {
	mu        sync.Mutex
	cached    string
	expiry    time.Time
	identity  oidcIdentity
	apiHost   string
	headers   map[string]interface{}
	userAgent string
	getenv    func(string) string
	now       func() time.Time
}

func (s *oidcTokenSource) canRetryAuth() bool { return true }

// invalidate drops the cached token only when it is still the one the caller
// used. Without that check, N requests failing a 401 in parallel would each
// discard the token a sibling had already refreshed, forcing N exchanges.
func (s *oidcTokenSource) invalidate(used string) {
	s.mu.Lock()
	if used == "" || s.cached == used {
		s.cached = ""
		s.expiry = time.Time{}
	}
	s.mu.Unlock()
}

func (s *oidcTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if s.cached != "" && now.Add(tokenRefreshSkew).Before(s.expiry) {
		return s.cached, nil
	}
	assertion, err := loadAssertion(s.getenv)
	if err != nil {
		return "", err
	}
	token, err := exchangeOIDC(ctx, s.apiHost, s.headers, s.userAgent, s.identity, assertion)
	if err != nil {
		return "", err
	}
	s.cached = token
	s.expiry = jwtExpiry(token)
	if s.expiry.IsZero() {
		// The exchange returned something we cannot read an exp from. Refresh on
		// a timer rather than caching forever: a token with no readable expiry
		// would otherwise only ever be replaced by a 401.
		s.expiry = now.Add(opaqueTokenLifetime)
	}
	return token, nil
}

type credential struct {
	static *string
	oidc   *oidcIdentity
}

type oidcIdentity struct {
	organization string
	serviceSlug  string
}

type authSpec struct {
	apiKey string
	oidc   *oidcBlockSpec
}

type oidcBlockSpec struct {
	organization string
	serviceSlug  string
}

func authSpecFromResourceData(d *schema.ResourceData) authSpec {
	spec := authSpec{apiKey: strings.TrimSpace(requiredString(d, "api_key"))}
	raw, ok := d.Get("oidc").([]interface{})
	if !ok || len(raw) == 0 {
		return spec
	}
	block := oidcBlockSpec{}
	if m, ok := raw[0].(map[string]interface{}); ok && m != nil {
		if v, ok := m["organization"].(string); ok {
			block.organization = strings.TrimSpace(v)
		}
		if v, ok := m["service_slug"].(string); ok {
			block.serviceSlug = strings.TrimSpace(v)
		}
	}
	spec.oidc = &block
	return spec
}

func envEnabled(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseCredential(spec authSpec, getenv func(string) string) (credential, error) {
	oidcOn := spec.oidc != nil || envEnabled(getenv("CLOUDSMITH_USE_OIDC"))
	// spec.apiKey is the HCL argument only: the api_key schema entry has no
	// DefaultFunc, so a leftover CLOUDSMITH_API_KEY workspace variable does not
	// conflict with oidc and an existing workspace can migrate as-is.
	if oidcOn && spec.apiKey != "" {
		return credential{}, errMixedCredentials
	}
	if oidcOn {
		org := ""
		slug := ""
		if spec.oidc != nil {
			org = spec.oidc.organization
			slug = spec.oidc.serviceSlug
		}
		if org == "" {
			org = strings.TrimSpace(getenv("CLOUDSMITH_ORG"))
		}
		if slug == "" {
			slug = strings.TrimSpace(getenv("CLOUDSMITH_SERVICE_SLUG"))
		}
		if org == "" || slug == "" {
			return credential{}, errIncompleteOIDC
		}
		return credential{oidc: &oidcIdentity{organization: org, serviceSlug: slug}}, nil
	}
	if spec.apiKey != "" {
		key := spec.apiKey
		return credential{static: &key}, nil
	}
	if key := strings.TrimSpace(getenv("CLOUDSMITH_API_KEY")); key != "" {
		return credential{static: &key}, nil
	}
	return credential{}, errMissingCredentials
}

func tokenSourceFromCredential(
	cred credential,
	apiHost string,
	headers map[string]interface{},
	userAgent string,
	getenv func(string) string,
	now func() time.Time,
) (tokenSource, error) {
	switch {
	case cred.static != nil:
		return staticToken(*cred.static), nil
	case cred.oidc != nil:
		if now == nil {
			now = time.Now
		}
		return &oidcTokenSource{
			identity:  *cred.oidc,
			apiHost:   apiHost,
			headers:   headers,
			userAgent: userAgent,
			getenv:    getenv,
			now:       now,
		}, nil
	default:
		return nil, errInvalidCredential
	}
}

// loadAssertion reads the identity token HCP Terraform injects into the run.
// The tagged variable wins, so a workspace can keep a Cloudsmith-specific
// audience alongside a generic one.
func loadAssertion(getenv func(string) string) (string, error) {
	for _, key := range []string{"TFC_WORKLOAD_IDENTITY_TOKEN_CLOUDSMITH", "TFC_WORKLOAD_IDENTITY_TOKEN"} {
		if v := strings.TrimSpace(getenv(key)); v != "" {
			return v, nil
		}
	}
	return "", errMissingOIDCToken
}

func exchangeOIDC(ctx context.Context, apiHost string, headers map[string]interface{}, userAgent string, id oidcIdentity, assertion string) (string, error) {
	openIDBase, err := openIDServerURL(apiHost)
	if err != nil {
		return "", err
	}

	cfg := cloudsmith.NewConfiguration()
	cfg.Debug = false
	cfg.HTTPClient = &http.Client{
		Timeout: oidcHTTPTimeout,
		Transport: &headerTransport{
			headers: headers,
			rt:      http.DefaultTransport,
		},
	}
	cfg.Servers = cloudsmith.ServerConfigurations{{URL: openIDBase}}
	cfg.UserAgent = userAgent

	client := cloudsmith.NewAPIClient(cfg)
	out, _, err := client.OpenidApi.OpenidCreate(ctx, id.organization).
		Data(cloudsmith.OidcRequest{
			OidcToken:   assertion,
			ServiceSlug: id.serviceSlug,
		}).
		Execute()
	if err != nil {
		return "", fmt.Errorf("OIDC token exchange failed: %w", err)
	}
	token := ""
	if out != nil {
		token = out.GetToken()
	}
	if token == "" {
		return "", errEmptyExchangeToken
	}
	return token, nil
}

// openIDServerURL drops a trailing /v1 from api_host: the exchange endpoint
// lives at the host root, not under the versioned API path.
func openIDServerURL(apiHost string) (string, error) {
	if apiHost == "" {
		return "", errInvalidAPIHost
	}
	u, err := url.Parse(apiHost)
	if err != nil {
		return "", fmt.Errorf("api_host: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", errInvalidAPIHost
	}

	path := strings.TrimSuffix(u.Path, "/")
	segments := strings.Split(path, "/")
	if n := len(segments); n > 0 && segments[n-1] == "v1" {
		segments = segments[:n-1]
	}
	u.Path = strings.Join(segments, "/")
	u.RawQuery = ""
	u.Fragment = ""
	u.RawPath = ""

	out := u.String()
	if u.Path == "" || u.Path == "/" {
		out = strings.TrimSuffix(out, "/")
	}
	return out, nil
}

func jwtExpiry(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		padded, padErr := base64.URLEncoding.DecodeString(parts[1])
		if padErr != nil {
			return time.Time{}
		}
		payload = padded
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}
