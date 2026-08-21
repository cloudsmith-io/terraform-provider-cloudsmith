package cloudsmith

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type rotatingTestTokenSource struct {
	mu          sync.Mutex
	current     string
	next        string
	invalidated []string
}

func (s *rotatingTestTokenSource) Token(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current, nil
}

func (s *rotatingTestTokenSource) invalidate(used string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalidated = append(s.invalidated, used)
	if s.current == used {
		s.current = s.next
	}
}

func (s *rotatingTestTokenSource) canRetryAuth() bool { return true }

type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}

func TestAPIKeyTransportStripsCredentialOutsideAPIEndpoint(t *testing.T) {
	endpoint, err := credentialEndpointFor("https://api.example.com/v1")
	if err != nil {
		t.Fatalf("credential endpoint: %v", err)
	}

	for _, target := range []string{
		"https://storage.example.com/package",
		"http://api.example.com/v1/packages/",
	} {
		t.Run(target, func(t *testing.T) {
			transport := &apiKeyTransport{
				tokens:      staticToken("secret"),
				apiEndpoint: endpoint,
				rt: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if got := req.Header.Get("X-Api-Key"); got != "" {
						t.Errorf("X-Api-Key = %q, want credential stripped", got)
					}
					return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
				}),
			}
			req, err := http.NewRequest(http.MethodGet, target, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			req.Header.Set("X-Api-Key", "credential-set-by-sdk")

			if _, err := transport.RoundTrip(req); err != nil {
				t.Fatalf("round trip: %v", err)
			}
		})
	}
}

func TestAPIKeyTransportStripsCredentialOnHTTPSDowngradeRedirect(t *testing.T) {
	endpoint, err := credentialEndpointFor("https://api.example.com/v1")
	if err != nil {
		t.Fatalf("credential endpoint: %v", err)
	}
	calls := 0
	client := &http.Client{Transport: &apiKeyTransport{
		tokens:      staticToken("secret"),
		apiEndpoint: endpoint,
		rt: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			switch calls {
			case 1:
				if got := req.Header.Get("X-Api-Key"); got != "secret" {
					t.Errorf("initial X-Api-Key = %q, want secret", got)
				}
				return &http.Response{
					StatusCode: http.StatusFound,
					Header:     http.Header{"Location": []string{"http://api.example.com/v1/packages/"}},
					Body:       http.NoBody,
					Request:    req,
				}, nil
			case 2:
				if got := req.Header.Get("X-Api-Key"); got != "" {
					t.Errorf("redirect X-Api-Key = %q, want credential stripped", got)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: req}, nil
			default:
				t.Fatalf("round trips = %d, want 2", calls)
				return nil, nil
			}
		}),
	}}

	req, err := http.NewRequest(http.MethodGet, "https://api.example.com/v1/packages/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client request: %v", err)
	}
	defer resp.Body.Close()
	if calls != 2 {
		t.Errorf("round trips = %d, want 2", calls)
	}
}

func TestAPIKeyTransportRetriesUnauthorizedRequestWithBody(t *testing.T) {
	tokens := &rotatingTestTokenSource{current: "old-token", next: "new-token"}
	firstBody := &closeTrackingBody{Reader: strings.NewReader("unauthorized")}
	var headers, bodies []string
	transport := &apiKeyTransport{
		tokens:      tokens,
		apiEndpoint: credentialEndpoint{scheme: "https", host: "api.example.com"},
		rt: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			headers = append(headers, req.Header.Get("X-Api-Key"))
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			bodies = append(bodies, string(body))
			if len(headers) == 1 {
				return &http.Response{StatusCode: http.StatusUnauthorized, Body: firstBody}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		}),
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/v1/packages/", strings.NewReader("request-body"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got, want := strings.Join(headers, ","), "old-token,new-token"; got != want {
		t.Errorf("credentials = %q, want %q", got, want)
	}
	if got, want := strings.Join(bodies, ","), "request-body,request-body"; got != want {
		t.Errorf("bodies = %q, want %q", got, want)
	}
	if !firstBody.closed {
		t.Error("first response body was not closed before retry")
	}
	if got, want := strings.Join(tokens.invalidated, ","), "old-token"; got != want {
		t.Errorf("invalidated tokens = %q, want %q", got, want)
	}
}

func TestAPIKeyTransportDoesNotRetryNonReplayableBody(t *testing.T) {
	tokens := &rotatingTestTokenSource{current: "old-token", next: "new-token"}
	calls := 0
	transport := &apiKeyTransport{
		tokens:      tokens,
		apiEndpoint: credentialEndpoint{scheme: "https", host: "api.example.com"},
		rt: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: http.NoBody}, nil
		}),
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/v1/packages/", io.NopCloser(strings.NewReader("request-body")))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if calls != 1 {
		t.Errorf("round trips = %d, want 1", calls)
	}
	if got, want := strings.Join(tokens.invalidated, ","), "old-token"; got != want {
		t.Errorf("invalidated tokens = %q, want %q", got, want)
	}
}
