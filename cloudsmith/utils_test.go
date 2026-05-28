//nolint:testpackage
package cloudsmith

import (
	"errors"
	"testing"
)

func TestFormatAPIError_Nil(t *testing.T) {
	if got := formatAPIError(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestFormatAPIError_NonAPIError(t *testing.T) {
	orig := errors.New("boom")
	got := formatAPIError(orig)
	if got != orig {
		t.Fatalf("expected original error to pass through, got %v", got)
	}
}

// TestFormatAPIErrorBody covers the raw-body fallback path used by
// formatAPIError when the SDK's typed ErrorDetail model is not populated.
// This path is the one that fixes issue #203: the SDK declares
// ErrorDetail.Fields as map[string][]string and fails to decode real
// responses whose fields contain non-[]string values, leaving callers with
// an opaque "json: cannot unmarshal ..." message. The raw-body parse here
// tolerates any fields shape via json.RawMessage.
//
// The primary SDK-native path (apiErr.Model().(cloudsmith.ErrorDetail)) is
// not unit-testable: GenericOpenAPIError's body/model fields are unexported,
// so a populated instance cannot be constructed from outside the SDK. That
// path is exercised by acceptance tests against the live API.
func TestFormatAPIErrorBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "detail only",
			body: `{"detail":"Authentication credentials were not provided."}`,
			want: "Authentication credentials were not provided.",
		},
		{
			name: "detail with non-[]string fields (issue #203 shape)",
			body: `{"detail":"Invalid input.","fields":{"slug":["Enter a valid slug."]}}`,
			want: `Invalid input. (fields: {"slug":["Enter a valid slug."]})`,
		},
		{
			name: "detail with nested-object fields (SDK decode would fail)",
			body: `{"detail":"Validation failed.","fields":{"privileges":{"0":{"slug":["Invalid."]}}}}`,
			want: `Validation failed. (fields: {"privileges":{"0":{"slug":["Invalid."]}}})`,
		},
		{
			name: "detail with null fields",
			body: `{"detail":"Forbidden.","fields":null}`,
			want: "Forbidden.",
		},
		{
			name: "unparseable body falls back to raw",
			body: `not-json`,
			want: "API error: not-json",
		},
		{
			name: "json without detail falls back to raw",
			body: `{"other":"value"}`,
			want: `API error: {"other":"value"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := formatAPIErrorBody([]byte(tc.body))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, err.Error())
			}
		})
	}
}
