package cloudsmith

import (
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/samber/lo"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	errKeepWaiting = errors.New("keep waiting")
	errTimedOut    = errors.New("timed out")

	defaultCreationTimeout  = time.Minute * 1
	defaultCreationInterval = time.Second * 2
	defaultDeletionTimeout  = time.Minute * 20
	defaultDeletionInterval = time.Second * 10
	defaultUpdateTimeout    = time.Minute * 1
	defaultUpdateInterval   = time.Second * 2
)

// contains returns true if value equals any element in the slice.
func contains[T comparable](slice []T, value T) bool {
	for _, elem := range slice {
		if elem == value {
			return true
		}
	}
	return false
}

// expandStrings retrieves a *schema.Set from TF state and converts it to
// a slice of strings which we can use with the API bindings.
func expandStrings(d *schema.ResourceData, key string) []string {
	set := d.Get(key).(*schema.Set)
	return lo.Map(set.List(), func(item interface{}, _ int) string {
		return item.(string)
	})
}

// flattenStrings converts a slice of strings such as might be returned by the
// API bindings to a *schema.Set which can be stored in TF state.
func flattenStrings(strings []string) *schema.Set {
	set := schema.NewSet(schema.HashString, []interface{}{})
	for _, s := range strings {
		set.Add(s)
	}
	return set
}

func is200(resp *http.Response) bool {
	if resp == nil {
		return false
	}

	return resp.StatusCode == http.StatusOK
}

func is404(resp *http.Response) bool {
	if resp == nil {
		return false
	}

	return resp.StatusCode == http.StatusNotFound
}

func nullableInt64(d *schema.ResourceData, name string) cloudsmith.NullableInt64 {
	i := optionalInt64(d, name)
	return *cloudsmith.NewNullableInt64(i)
}

func nullableString(d *schema.ResourceData, name string) cloudsmith.NullableString {
	s := optionalString(d, name)
	return *cloudsmith.NewNullableString(s)
}

func nullableTime(d *schema.ResourceData, name string) cloudsmith.NullableTime {
	s := optionalString(d, name)

	if s == nil {
		return *cloudsmith.NewNullableTime(nil)
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		panic(err)
	}

	return *cloudsmith.NewNullableTime(&t)
}

// optionalBool retrieves an optional/nullable boolean from Terraform state
func optionalBool(d *schema.ResourceData, name string) *bool {
	var optionalValue *bool

	if value, ok := d.GetOkExists(name); ok { //nolint:staticcheck
		optionalValue = cloudsmith.PtrBool(value.(bool))
	}

	return optionalValue
}

// optionalInt64 retrieves an optional/nullable int64 from Terraform state
func optionalInt64(d *schema.ResourceData, name string) *int64 {
	var optionalValue *int64

	if value, ok := d.GetOk(name); ok { //nolint:staticcheck
		optionalValue = cloudsmith.PtrInt64(int64(value.(int)))
	}

	return optionalValue
}

// optionalString retrieves an optional/nullable string from Terraform state
func optionalString(d *schema.ResourceData, name string) *string {
	var optionalValue *string

	if value, ok := d.GetOk(name); ok {
		optionalValue = cloudsmith.PtrString(value.(string))
	}

	return optionalValue
}

// requiredBool retrieves a boolean from Terraform state
func requiredBool(d *schema.ResourceData, name string) bool {
	return d.Get(name).(bool)
}

// requiredString retrieves a string from Terraform state
func requiredString(d *schema.ResourceData, name string) string {
	return d.Get(name).(string)
}

// stringSlicesAreEqual compares two string slices and returns true if they are equal.
func stringSlicesAreEqual(x, y []string, sortSlices bool) bool {
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

// timeToString converts a time.Time object to a string
func timeToString(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.Format(time.RFC3339)
}

// stringToTime converts a string to a time.Time object
func stringToTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// waitFunc should be implemented by callers that want to wait on a particular
// action
type waitFunc func() error

// waiter can be called with a waitFunc to poll for completion of a given
// action. This is mostly useful for actions that change state and may not be
// immediately reflected in the API for any reason.
func waiter(checker waitFunc, timeout, interval time.Duration) error {
	// the initial sleep here helps avoid issues with cross-region database
	// replication. Most endpoints deal with this fine, but there are still a
	// few edge cases that we need to fix in the APIs before we can safely
	// remove this.
	time.Sleep(interval)

	for start := time.Now(); time.Since(start) < timeout; {
		if err := checker(); err != nil {
			if err == errKeepWaiting {
				time.Sleep(interval)
				continue
			}
			return err
		}
		return nil
	}

	return errTimedOut
}
