package cloudsmith

import (
	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func optionalBool(d *schema.ResourceData, name string) *bool {
	var optionalValue *bool

	if value, ok := d.GetOkExists(name); ok { //nolint:staticcheck
		optionalValue = cloudsmith.PtrBool(value.(bool))
	}

	return optionalValue
}

func optionalInt64(d *schema.ResourceData, name string) *int64 {
	var optionalValue *int64

	if value, ok := d.GetOk(name); ok { //nolint:staticcheck
		optionalValue = cloudsmith.PtrInt64(int64(value.(int)))
	}

	return optionalValue
}

func optionalString(d *schema.ResourceData, name string) *string {
	var optionalValue *string

	if value, ok := d.GetOk(name); ok {
		optionalValue = cloudsmith.PtrString(value.(string))
	}

	return optionalValue
}

func requiredBool(d *schema.ResourceData, name string) bool {
	return d.Get(name).(bool)
}

func requiredString(d *schema.ResourceData, name string) string {
	return d.Get(name).(string)
}
