// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"context"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	cloudsmithv2 "github.com/cloudsmith-io/cloudsmith-go-v2"
)

// TokenSource supplies the current API credential.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// ProviderConfig is the configured state handed to every resource and data
// source as the provider meta value.
type ProviderConfig struct {
	Auth context.Context

	APIClient *cloudsmith.APIClient

	V2ApiClient *cloudsmithv2.Cloudsmith

	Tokens TokenSource
}

// GetAPIKey returns the current credential. A token-source failure is returned
// rather than swallowed: falling back to the token minted at configure time
// would send a known-expired credential and report the resulting rejection
// instead of the refresh error that caused it.
func (pc *ProviderConfig) GetAPIKey() (string, error) {
	if pc.Tokens != nil {
		return pc.Tokens.Token(context.Background())
	}
	apiKeys, _ := pc.Auth.Value(cloudsmith.ContextAPIKeys).(map[string]cloudsmith.APIKey)
	return apiKeys["apikey"].Key, nil
}
