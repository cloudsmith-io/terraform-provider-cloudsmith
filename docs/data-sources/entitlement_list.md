# Entitlement Tokens Data Source
The `entitlement_tokens` data source allows for retrieval of a list of entitlement tokens within a given repository.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

data "cloudsmith_repository" "my_repository" {
    namespace  = data.cloudsmith_organization.my_organization.slug_perm
    identifier = "my-repository"
}

data "cloudsmith_entitlement_list" "my_tokens" {
    namespace  = data.cloudsmith_repository.my_repository.namespace
    repository = data.cloudsmith_repository.my_repository.slug_perm

    query        = ["name:Default"]
    show_token   = true
    active_token = true
}

output "tokens" {
    value = data.cloudsmith_entitlement_list.my_tokens.entitlement_tokens.*.token
}
```

## Argument Reference

* `namespace` - (Required) Namespace to which the entitlement tokens belong.
* `repository` - (Required) Repository `slug_perm` to which the entitlement tokens belong.
* `query` - (Optional) A search term for querying names of entitlements.
* `show_token` - (Optional) Show entitlement token strings in results. Default is `false`.
* `active_token` - (Optional) If true, only include active tokens. Default is `false`.

## Attribute Reference

All of the argument attributes are also exported as result attributes.

The following attribute is additionally exported:

* `entitlement_tokens` - A list of `entitlement_token` entries as discovered by the data source. Each `entitlement_token` has the following attributes:
  * `clients` - Number of clients associated with the entitlement token.
  * `created_at` - The date/time the token was created at.
  * `created_by` - The user who created the entitlement token.
  * `default` - If selected this is the default token for this repository.
  * `downloads` - Number of downloads associated with the entitlement token.
  * `disable_url` - URL to disable the entitlement token.
  * `enable_url` - URL to enable the entitlement token.
  * `eula_required` - If checked, a EULA acceptance is required for this token.
  * `has_limits` - Indicates if there are limits set for the token.
  * `identifier` - A unique identifier for the entitlement token.
  * `is_active` - If enabled, the token will allow downloads based on configured restrictions (if any).
  * `is_limited` - Indicates if the token is limited.
  * `limit_bandwidth` - The maximum download bandwidth allowed for the token.
  * `limit_bandwidth_unit` - Unit of bandwidth for the maximum download bandwidth.
  * `limit_date_range_from` - The starting date/time the token is allowed to be used from.
  * `limit_date_range_to` - The ending date/time the token is allowed to be used until.
  * `limit_num_clients` - The maximum number of unique clients allowed for the token.
  * `limit_num_downloads` - The maximum number of downloads allowed for the token.
  * `limit_package_query` - The package-based search query to apply to restrict downloads.
  * `limit_path_query` - The path-based search query to apply to restrict downloads.
  * `metadata` - Additional metadata associated with the entitlement token.
  * `name` - The name of the entitlement token.
  * `refresh_url` - URL to refresh the entitlement token.
  * `reset_url` - URL to reset the entitlement token.
  * `scheduled_reset_at` - The time at which the scheduled reset period has elapsed and the token limits were automatically reset to zero.
  * `scheduled_reset_period` - The period after which the token limits are automatically reset to zero.
  * `self_url` - URL for the entitlement token itself.
  * `slug_perm` - Slug permission associated with the entitlement token.
  * `token` - The entitlement token string.
  * `updated_at` - The date/time the token was updated at.
  * `updated_by` - The user who updated the entitlement token.
  * `updated_by_url` - URL for the user who updated the entitlement token.
  * `usage` - The usage associated with the token.
  * `user` - The user associated with the token.
  * `user_url` - URL for the user associated with the token.