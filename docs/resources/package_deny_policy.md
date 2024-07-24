# Package Deny Policy Resource

Create a package deny policy resource.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

data "cloudsmith_package_deny_policy" "test" {
    namespace = my_organization.slug_perm
    enabled = true
    name = "test-package-deny-policy-terraform-provider"
    package_query = "name:example"
}
```

## Argument Reference

The following arguments are supported:

- `name` (Optional) - A descriptive name for the package deny policy.
- `description` (Optional) - Description of the package deny policy.
- `package_query` (Required) - The query to match the packages to be blocked.
- `enabled` (Optional) - Is the package deny policy enabled? Defaults to `true`
- `namespace` - The namespace where package deny policy is managed

## Attribute Reference

The following attributes are exported:

- `id` - The ID of the package deny policy.
- `name` - The name of the package deny policy.
- `description` - The description of the package deny policy.
- `package_query` - The query used to match the packages to be blocked.
- `enabled` - Whether the package deny policy is enabled.
- `namespace` - The namespace where package deny policy is managed