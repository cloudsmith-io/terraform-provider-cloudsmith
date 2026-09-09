# API Key Rule Resource

The API key rule resource allows for creation and management of API key rules within a Cloudsmith organization. A rule sets the maximum permitted age of API keys for a given account type, after which those keys lose access until they are refreshed.

An organization may hold at most one rule per account type. Changing `rule_type` therefore creates a new resource.

## Example Usage

```hcl
provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

resource "cloudsmith_api_key_rule" "user_keys" {
  organization    = data.cloudsmith_organization.my_organization.slug
  rule_type       = "User Accounts"
  max_age_hours   = 2160
  is_enabled      = true
  enforce_refresh = false
}

resource "cloudsmith_api_key_rule" "service_keys" {
  organization    = data.cloudsmith_organization.my_organization.slug
  rule_type       = "Service Accounts"
  max_age_hours   = 8760
  is_enabled      = true
  enforce_refresh = true
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Required) Organization to which the rule belongs. Changing this value creates a new resource.
* `rule_type` - (Required) The account types this rule applies to. One of: `Service Accounts`, `User Accounts`. Changing this value creates a new resource.
* `max_age_hours` - (Optional) The maximum permitted age of an API key, in hours. API keys older than this lose access until refreshed. Omit to disable the age limit.
* `is_enabled` - (Optional) Whether this rule is currently active and enforced.
* `enforce_refresh` - (Optional) When enabled, API keys that violate this rule are replaced automatically.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `slug` - Human-readable identifier, generated from the rule type.
* `slug_perm` - An auto-generated id that uniquely identifies the rule.
* `created_at` - The time the rule was created at.
* `updated_at` - The time the rule was last updated at.
* `last_applied_at` - The last time this rule was evaluated and applied by the expiry task.

## Import

This resource can be imported using the organization slug and the rule's `slug_perm`, separated by a period.

```shell
terraform import cloudsmith_api_key_rule.user_keys my-organization.utp-abc123xyz456
```

Rules created outside of Terraform, such as in the Cloudsmith UI, must be imported before Terraform can manage them. Without this, an apply fails because a rule already exists for that organization and account type.
