package cloudsmith

import (
	"errors"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

var (
	errKeepWaiting = errors.New("keep waiting")
	errMessage404  = "404 Not Found"
	errTimedOut    = errors.New("timed out")

	defaultCreationTimeout  = time.Minute * 1
	defaultCreationInterval = time.Second * 2
	defaultDeletionTimeout  = time.Minute * 20
	defaultDeletionInterval = time.Second * 10
	defaultUpdateTimeout    = time.Minute * 1
	defaultUpdateInterval   = time.Second * 2
)

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

// waitFunc should be implemented by callers that want to wait on a particular
// action
type waitFunc func() error

// waiter can be called with a waitFunc to poll for completion of a given
// action. This is mostly useful for actions that change state and may not be
// immediately reflected in the API for any reason.
func waiter(checker waitFunc, timeout, interval time.Duration) error {
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
