# Repository Retention Rules Resource

The repository retention rules resource allows the management of retention rules for a given Cloudsmith repository. Using this resource, it is possible to define rules that control the retention of packages based on various criteria such as count, days, size, and grouping.

Note that while retention rules can be managed in this manner, changes made outside of Terraform may not be reflected in the Terraform state.

**Note: Retention rule settings are only applied once retention is enabled for the repository.**

See [docs.cloudsmith.com](https://docs.cloudsmith.com/artifact-management/retention-rules) for full retention rules documentation.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

resource "cloudsmith_repository" "my_repository" {
    name        = "retention-rules"
    namespace   = data.cloudsmith_organization.my_organization.slug_perm
}

resource "cloudsmith_repository_retention_rule" "retention_rule" {
    namespace                  = data.cloudsmith_organization.my_organization.slug
    repository                 = cloudsmith_repository.my_repository.slug
    retention_enabled          = true
    retention_count_limit      = 100
    retention_days_limit       = 28
    retention_group_by_name    = false
    retention_group_by_format  = false
    retention_group_by_package_type = false
    retention_size_limit       = 200000
}
```

## Argument Reference

The following arguments are supported:

* `namespace` - (Required) The namespace of the repository.
* `repository` - (Required) If true, the retention rules will be activated for the repository and settings will be updated.
* `retention_enabled` - (Required) If true, the retention rules will be activated for the repository and settings will be updated.
* `retention_count_limit` - (Optional) The maximum number of packages to retain. Must be between `0` and `10000`. Default set to 100 packages as part of repository creation.
* `retention_days_limit` - (Optional) The number of days of packages to retain. Must be between `0` and `180`. Default set to 28 days as part of repository creation.
* `retention_group_by_name` - (Optional) If true, retention will apply to groups of packages by name rather than all packages.
* `retention_group_by_format` - (Optional) If true, retention will apply to packages by package formats rather than across all package formats.
* `retention_group_by_package_type` - (Optional) If true, retention will apply to packages by package type rather than across all package types for one or more formats.
* `retention_size_limit` - (Optional) The maximum total size (in bytes) of packages to retain. Must be between `0` and `20000000000` up to the maximum size of 20 GB (20,000,000,000 bytes).
* `retention_package_query_string` - (Optional) A package search expression which, if provided, filters the packages to be deleted. For example, `name:foo` will only delete packages called 'foo'.

## Import

This resource can be imported using the namespace and repository slug:

```shell
terraform import cloudsmith_repository_retention_rule.retention_rule my-namespace-slug.my-repository
```
