# Entitlement Resource

The entitlement resource allows the creation and management of Entitlement tokens for a given Cloudsmith repository. Entitlement tokens grant read-only access to a repository and can be configured with a number of custom restrictions if necessary.

See [help.cloudsmith.io](https://help.cloudsmith.io/docs/entitlements) for full entitlement documentation.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_namespace" "my_namespace" {
    slug = "my-namespace"
}

resource "cloudsmith_repository" "my_repository" {
    description = "A certifiably-awesome private package repository"
    name        = "My Repository"
    namespace   = "${data.cloudsmith_namespace.my_namespace.slug_perm}"
    slug        = "my-repository"
}

resource "cloudsmith_entitlement" "my_entitlement" {
    name       = "Test Entitlement"
    namespace  = "${cloudsmith_repository.test.namespace}"
    repository = "${cloudsmith_repository.test.slug_perm}"
}
```

## Argument Reference

* `is_active` - (Optional) If enabled, the token will allow downloads based on configured restrictions (if any).
* `limit_date_range_from` - (Optional) The starting date/time the token is allowed to be used from.
* `limit_date_range_to` - (Optional) The ending date/time the token is allowed to be used until.
* `limit_num_clients` - (Optional) The maximum number of unique clients allowed for the token. Please note that since clients are calculated asynchronously (after the download happens), the limit may not be imposed immediately but at a later point.
* `limit_num_downloads` - (Optional) The maximum number of downloads allowed for the token. Please note that since downloads are calculated asynchronously (after the download happens), the limit may not be imposed immediately but at a later point.
* `limit_package_query` - (Optional) The package-based search query to apply to restrict downloads to. This uses the same syntax as the standard search used for repositories, and also supports boolean logic operators such as OR/AND/NOT and parentheses for grouping. This will still allow access to non-package files, such as metadata.
* `limit_path_query` - (Optional) The path-based search query to apply to restrict downloads to. This supports boolean logic operators such as OR/AND/NOT and parentheses for grouping. The path evaluated does not include the domain name, the namespace, the entitlement code used, the package format, etc. and it always starts with a forward slash.
* `name` - (Required) A descriptive name for the entitlement.
* `namespace` - (Required) Namespace to which this entitlement belongs.
* `repository` - (Required) Repository to which this entitlement belongs.
* `token` - (Optional) The literal value of the token to be created.

## Attribute Reference

* `is_active` - If enabled, the token will allow downloads based on configured restrictions (if any).
* `limit_date_range_from` - The starting date/time the token is allowed to be used from.
* `limit_date_range_to` - The ending date/time the token is allowed to be used until.
* `limit_num_clients` - The maximum number of unique clients allowed for the token. Please note that since clients are calculated asynchronously (after the download happens), the limit may not be imposed immediately but at a later point.
* `limit_num_downloads` - The maximum number of downloads allowed for the token. Please note that since downloads are calculated asynchronously (after the download happens), the limit may not be imposed immediately but at a later point.
* `limit_package_query` - The package-based search query to apply to restrict downloads to. This uses the same syntax as the standard search used for repositories, and also supports boolean logic operators such as OR/AND/NOT and parentheses for grouping. This will still allow access to non-package files, such as metadata.
* `limit_path_query` - The path-based search query to apply to restrict downloads to. This supports boolean logic operators such as OR/AND/NOT and parentheses for grouping. The path evaluated does not include the domain name, the namespace, the entitlement code used, the package format, etc. and it always starts with a forward slash.
* `name` - A descriptive name for the entitlement.
* `namespace` - Namespace to which this entitlement belongs.
* `repository` - Repository to which this entitlement belongs.
* `token` - The literal value of the token to be created.
