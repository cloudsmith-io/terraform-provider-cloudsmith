// Copyright Cloudsmith Ltd 2026
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/samber/lo"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	v2apierrors "github.com/cloudsmith-io/cloudsmith-go-v2/models/apierrors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	ErrKeepWaiting = errors.New("keep waiting")
	ErrTimedOut    = errors.New("timed out")

	DefaultCreationTimeout  = time.Minute * 1
	DefaultCreationInterval = time.Second * 2
	DefaultDeletionTimeout  = time.Minute * 20
	DefaultDeletionInterval = time.Second * 10
	DefaultUpdateTimeout    = time.Minute * 1
	DefaultUpdateInterval   = time.Second * 2
)

// Contains returns true if value equals any element in the slice.
func Contains[T comparable](slice []T, value T) bool {
	for _, elem := range slice {
		if elem == value {
			return true
		}
	}
	return false
}

// ExpandStrings retrieves a *schema.Set from TF state and converts it to
// a slice of strings which we can use with the API bindings.
func ExpandStrings(d *schema.ResourceData, key string) []string {
	set := d.Get(key).(*schema.Set)
	return lo.Map(set.List(), func(item interface{}, _ int) string {
		return item.(string)
	})
}

// FlattenStrings converts a slice of strings such as might be returned by the
// API bindings to a *schema.Set which can be stored in TF state.
func FlattenStrings(strings []string) *schema.Set {
	set := schema.NewSet(schema.HashString, []interface{}{})
	for _, s := range strings {
		set.Add(s)
	}
	return set
}

func Is200(resp *http.Response) bool {
	if resp == nil {
		return false
	}

	return resp.StatusCode == http.StatusOK
}

func Is404(resp *http.Response) bool {
	if resp == nil {
		return false
	}

	return resp.StatusCode == http.StatusNotFound
}

// apiErrorBody is a permissive parse of the JSON error body returned by the
// Cloudsmith API. We use json.RawMessage for Fields because the SDK's typed
// ErrorDetail uses map[string][]string, which fails to decode some real API
// responses (e.g. nested objects). Falling back to a raw message lets us
// surface the body verbatim instead of leaking the SDK unmarshal error.
type apiErrorBody struct {
	Detail string          `json:"detail"`
	Fields json.RawMessage `json:"fields,omitempty"`
}

// FormatAPIError surfaces API-provided detail/fields from a generated-SDK
// error. Order of preference:
//  1. SDK's typed ErrorDetail model (populated for status codes the SDK
//     decodes per-endpoint, e.g. 400/404/422 with a compatible body).
//  2. Raw response body parsed permissively. Required because the SDK's
//     ErrorDetail.Fields is map[string][]string and some real API responses
//     do not match that schema, causing the SDK to return the unmarshal
//     error verbatim (the original bug from issue #203).
//  3. Original error, unchanged.
func FormatAPIError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *cloudsmith.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		return err
	}

	if detail, ok := apiErr.Model().(cloudsmith.ErrorDetail); ok {
		if detail.HasFields() {
			return fmt.Errorf("%s (fields: %v)", detail.GetDetail(), detail.GetFields())
		}
		return fmt.Errorf("%s", detail.GetDetail())
	}

	body := apiErr.Body()
	if len(body) == 0 {
		return err
	}
	return FormatAPIErrorBody(body)
}

// FormatAPIErrorBody parses a raw API error body and returns a formatted error.
// Split out from FormatAPIError so the parsing logic is unit-testable without
// needing to construct a generated-SDK error (its body field is unexported).
func FormatAPIErrorBody(body []byte) error {
	var parsed apiErrorBody
	if jsonErr := json.Unmarshal(body, &parsed); jsonErr == nil && parsed.Detail != "" {
		if len(parsed.Fields) > 0 && string(parsed.Fields) != "null" {
			return fmt.Errorf("%s (fields: %s)", parsed.Detail, string(parsed.Fields))
		}
		return fmt.Errorf("%s", parsed.Detail)
	}

	return fmt.Errorf("API error: %s", string(body))
}

// FormatV2APIError surfaces API-provided detail/fields from a v2-SDK error.
// Order of preference:
//  1. SDK's typed apierrors.ErrorDetail (populated when the response body
//     decodes into the typed model).
//  2. apierrors.APIError raw body parsed permissively via FormatAPIErrorBody.
//     The v2 SDK's ErrorDetail.Fields is map[string][]string, so responses
//     with non-matching field shapes (e.g. nested objects) don't decode
//     into the typed model and surface as APIError; this branch keeps that
//     case readable instead of dumping "Status N\n<json blob>".
//  3. Original error, unchanged.
func FormatV2APIError(err error) error {
	if err == nil {
		return nil
	}

	var detail *v2apierrors.ErrorDetail
	if errors.As(err, &detail) {
		if len(detail.Fields) > 0 {
			return fmt.Errorf("%s (fields: %v)", detail.Detail, detail.Fields)
		}
		return fmt.Errorf("%s", detail.Detail)
	}

	var apiErr *v2apierrors.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Body == "" {
			return err
		}
		return FormatAPIErrorBody([]byte(apiErr.Body))
	}

	return err
}

func NullableInt64(d *schema.ResourceData, name string) cloudsmith.NullableInt64 {
	i := OptionalInt64(d, name)
	return *cloudsmith.NewNullableInt64(i)
}

func NullableString(d *schema.ResourceData, name string) cloudsmith.NullableString {
	s := OptionalString(d, name)
	return *cloudsmith.NewNullableString(s)
}

func NullableTime(d *schema.ResourceData, name string) cloudsmith.NullableTime {
	s := OptionalString(d, name)

	if s == nil {
		return *cloudsmith.NewNullableTime(nil)
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		panic(err)
	}

	return *cloudsmith.NewNullableTime(&t)
}

// OptionalBool retrieves an optional/nullable boolean from Terraform state
func OptionalBool(d *schema.ResourceData, name string) *bool {
	var optionalValue *bool

	if value, ok := d.GetOkExists(name); ok { //nolint:staticcheck
		optionalValue = cloudsmith.PtrBool(value.(bool))
	}

	return optionalValue
}

// OptionalInt64 retrieves an optional/nullable int64 from Terraform state
func OptionalInt64(d *schema.ResourceData, name string) *int64 {
	var optionalValue *int64

	if value, ok := d.GetOkExists(name); ok { //nolint:staticcheck
		optionalValue = cloudsmith.PtrInt64(int64(value.(int)))
	}

	return optionalValue
}

// OptionalString retrieves an optional/nullable string from Terraform state
func OptionalString(d *schema.ResourceData, name string) *string {
	var optionalValue *string

	if value, ok := d.GetOk(name); ok {
		optionalValue = cloudsmith.PtrString(value.(string))
	}

	return optionalValue
}

// RequiredBool retrieves a boolean from Terraform state
func RequiredBool(d *schema.ResourceData, name string) bool {
	return d.Get(name).(bool)
}

// RequiredString retrieves a string from Terraform state
func RequiredString(d *schema.ResourceData, name string) string {
	return d.Get(name).(string)
}

func BoolOrFalse(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

func Int64OrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

// StringSlicesAreEqual compares two string slices and returns true if they are equal.
func StringSlicesAreEqual(x, y []string, sortSlices bool) bool {
	if len(x) != len(y) {
		return false
	}

	if sortSlices {
		sort.Strings(x)
		sort.Strings(y)
	}

	for i, v := range x {
		if v != y[i] {
			return false
		}
	}

	return true
}

// TimeToString converts a time.Time object to a string
func TimeToString(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.Format(time.RFC3339)
}

// StringToTime converts a string to a time.Time object
func StringToTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// WaitFunc should be implemented by callers that want to wait on a particular
// action
type WaitFunc func() error

// Waiter can be called with a WaitFunc to poll for completion of a given
// action. This is mostly useful for actions that change state and may not be
// immediately reflected in the API for any reason.
func Waiter(checker WaitFunc, timeout, interval time.Duration) error {
	// the initial sleep here helps avoid issues with cross-region database
	// replication. Most endpoints deal with this fine, but there are still a
	// few edge cases that we need to fix in the APIs before we can safely
	// remove this.
	time.Sleep(interval)

	for start := time.Now(); time.Since(start) < timeout; {
		if err := checker(); err != nil {
			if err == ErrKeepWaiting {
				time.Sleep(interval)
				continue
			}
			return err
		}
		return nil
	}

	return ErrTimedOut
}

// WaitForCreation polls readFunc until a resource becomes available (not 404).
// readFunc should return the HTTP response and error from a read API call.
func WaitForCreation(readFunc func() (*http.Response, error), resourceType, resourceID string) error {
	checker := func() error {
		resp, err := readFunc()
		if err != nil {
			if Is404(resp) {
				return ErrKeepWaiting
			}
			return err
		}
		return nil
	}
	if err := Waiter(checker, DefaultCreationTimeout, DefaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for %s (%s) to be created: %w", resourceType, resourceID, err)
	}
	return nil
}

// WaitForDeletion polls readFunc until a resource is gone (404).
// readFunc should return the HTTP response and error from a read API call.
func WaitForDeletion(readFunc func() (*http.Response, error), resourceType, resourceID string) error {
	checker := func() error {
		resp, err := readFunc()
		if err != nil {
			if Is404(resp) {
				return nil
			}
			return err
		}
		return ErrKeepWaiting
	}
	if err := Waiter(checker, DefaultDeletionTimeout, DefaultDeletionInterval); err != nil {
		return fmt.Errorf("error waiting for %s (%s) to be deleted: %w", resourceType, resourceID, err)
	}
	return nil
}

// WaitForUpdate pauses briefly to allow eventual consistency to settle after
// an update. There is no concrete condition to poll, so we just sleep for a
// fixed duration.
func WaitForUpdate(resourceType, resourceID string) error {
	time.Sleep(7 * time.Second)
	return nil
}
