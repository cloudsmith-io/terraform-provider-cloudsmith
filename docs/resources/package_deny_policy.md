# Package Deny Policy Resource

The package deny policy resource allows for creation and management of package deny policies within a Cloudsmith organization.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

resource "cloudsmith_package_deny_policy" "my_package_deny_policy" {
    name          = "My Deny Policy"
    description   = "My package deny policy"
    package_query = "name:example"
    enabled       = true
    namespace     = data.cloudsmith_organization.my_organization.slug_perm
}
```

## Argument Reference

The following arguments are supported:

* `namespace` - (Required) Namespace to which this package deny policy belongs.
* `package_query` - (Required) The query to match the packages to be blocked.
* `name` - (Optional) A descriptive name for the package deny policy.
* `description` - (Optional) Description of the package deny policy.
* `enabled` - (Optional) Is the package deny policy enabled? Defaults to `true`.

## Attribute Reference

The following attributes are exported:

* `id` - The ID of the package deny policy.
* `name` - The name of the package deny policy.
* `description` - The description of the package deny policy.
* `package_query` - The query used to match the packages to be blocked.
* `enabled` - Whether the package deny policy is enabled.
* `namespace` - The namespace where package deny policy is managed.

## Import

This resource can be imported using the namespace slug and the package deny policy slug_perm.

```shell
terraform import cloudsmith_package_deny_policy.my_package_deny_policy my-organization.my-policy-slug-perm
```
