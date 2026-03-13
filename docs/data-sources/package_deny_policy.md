# Package Deny Policy Data Source

The `cloudsmith_package_deny_policy` data source allows fetching an existing package deny policy within a Cloudsmith namespace.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

resource "cloudsmith_package_deny_policy" "my_package_deny_policy" {
    namespace     = data.cloudsmith_organization.my_organization.slug_perm
    name          = "My Deny Policy"
    description   = "My package deny policy"
    package_query = "name:example"
    enabled       = true
}

data "cloudsmith_package_deny_policy" "existing" {
    namespace = cloudsmith_package_deny_policy.my_package_deny_policy.namespace
    slug_perm = cloudsmith_package_deny_policy.my_package_deny_policy.id
}
```

## Argument Reference

* `namespace` - (Required) Namespace to which this package deny policy belongs.
* `slug_perm` - (Required) Identifier of the package deny policy.

## Attribute Reference

* `id` - Fully-qualified identifier in the format `<namespace>.<slug_perm>`.
* `name` - A descriptive name for the package deny policy.
* `description` - Description of the package deny policy.
* `package_query` - The query to match the packages to be blocked.
* `enabled` - Whether the package deny policy is enabled.
* `namespace` - Namespace to which this package deny policy belongs.
* `slug_perm` - Identifier of the package deny policy.
