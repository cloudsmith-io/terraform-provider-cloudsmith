# Package Deny Policy Data Source

The `package_deny_policy` data source allows fetching of a package deny policy within a Cloudsmith organization.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

data "cloudsmith_package_deny_policy" "my_package_deny_policy" {
    namespace = data.cloudsmith_organization.my_organization.slug_perm
    slug_perm = "my-policy-slug-perm"
}
```

## Argument Reference

* `namespace` - (Required) Namespace to which this package deny policy belongs.
* `slug_perm` - (Required) Identifier of the package deny policy.

## Attribute Reference

* `name` - A descriptive name for the package deny policy.
* `description` - Description of the package deny policy.
* `package_query` - The query to match the packages to be blocked.
* `enabled` - Is the package deny policy enabled?
