//nolint:testpackage
package cloudsmith

import (
	"errors"
	"fmt"
	"testing"

	v2apierrors "github.com/cloudsmith-io/cloudsmith-go-v2/models/apierrors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

func TestFormatV2APIError_Nil(t *testing.T) {
	if got := formatV2APIError(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestFormatV2APIError_NonAPIError(t *testing.T) {
	orig := errors.New("boom")
	if got := formatV2APIError(orig); got != orig {
		t.Fatalf("expected original error to pass through, got %v", got)
	}
}

func TestFormatV2APIError_ErrorDetail(t *testing.T) {
	tests := []struct {
		name   string
		detail *v2apierrors.ErrorDetail
		want   string
	}{
		{
			name:   "detail only",
			detail: &v2apierrors.ErrorDetail{Detail: "Forbidden."},
			want:   "Forbidden.",
		},
		{
			name: "detail with fields",
			detail: &v2apierrors.ErrorDetail{
				Detail: "Invalid input.",
				Fields: map[string][]string{"slug": {"Enter a valid slug."}},
			},
			want: "Invalid input. (fields: map[slug:[Enter a valid slug.]])",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatV2APIError(tc.detail)
			if got == nil {
				t.Fatalf("expected error, got nil")
			}
			if got.Error() != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got.Error())
			}
		})
	}
}

func TestFormatV2APIError_APIErrorBodyFallback(t *testing.T) {
	// Mirrors the issue #203 shape: nested-object fields the typed
	// ErrorDetail can't decode. v2 SDK surfaces these as APIError with the
	// raw body; we want formatAPIErrorBody's permissive parse to apply.
	apiErr := v2apierrors.NewAPIError(
		"unexpected response",
		400,
		`{"detail":"Validation failed.","fields":{"privileges":{"0":{"slug":["Invalid."]}}}}`,
		nil,
	)
	got := formatV2APIError(apiErr)
	want := `Validation failed. (fields: {"privileges":{"0":{"slug":["Invalid."]}}})`
	if got.Error() != want {
		t.Fatalf("expected %q, got %q", want, got.Error())
	}
}

func TestFormatV2APIError_APIErrorEmptyBodyPassthrough(t *testing.T) {
	apiErr := v2apierrors.NewAPIError("boom", 500, "", nil)
	got := formatV2APIError(apiErr)
	// Empty body: the helper should pass through the original APIError so
	// the caller still gets a useful (if generic) message.
	want := apiErr.Error()
	if got.Error() != want {
		t.Fatalf("expected pass-through %q, got %q", want, got.Error())
	}
}

func TestFormatV2APIError_WrappedError(t *testing.T) {
	apiErr := v2apierrors.NewAPIError(
		"unexpected response",
		400,
		`{"detail":"Bad request."}`,
		nil,
	)
	wrapped := fmt.Errorf("listing policies: %w", apiErr)
	got := formatV2APIError(wrapped)
	want := "Bad request."
	if got.Error() != want {
		t.Fatalf("expected %q, got %q", want, got.Error())
	}
}

func TestOptionalInt64IncludesExplicitZero(t *testing.T) {
	t.Parallel()

	resourceSchema := map[string]*schema.Schema{
		"precedence": {
			Type:     schema.TypeInt,
			Optional: true,
			Computed: true,
		},
	}

	d := schema.TestResourceDataRaw(t, resourceSchema, map[string]interface{}{
		"precedence": 0,
	})

	value := optionalInt64(d, "precedence")
	if value == nil {
		t.Fatal("expected explicit zero to be preserved")
		return
	}
	if *value != 0 {
		t.Fatalf("expected explicit zero, got %d", *value)
	}
}

func TestOptionalInt64Unset(t *testing.T) {
	t.Parallel()

	resourceSchema := map[string]*schema.Schema{
		"precedence": {
			Type:     schema.TypeInt,
			Optional: true,
			Computed: true,
		},
	}

	d := schema.TestResourceDataRaw(t, resourceSchema, map[string]interface{}{})

	if value := optionalInt64(d, "precedence"); value != nil {
		t.Fatalf("expected unset value to be nil, got %d", *value)
	}
}
