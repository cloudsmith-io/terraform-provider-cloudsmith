# Namespace Data Source

!> **WARNING:** This data source is deprecated and will be removed in future. Use `cloudsmith_organization` instead.

The `namespace` data source allows fetching of metadata about a given Cloudsmith namespace. The fetched data can be used to resolve permanent identifiers from a namespace's user-facing name. These identifiers can then be passed to other resources to allow more consistent identification as user-facing names can change.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_namespace" "my_namespace" {
    slug = "my-namespace"
}
```

## Argument Reference

* `slug` - (Required) The slug identifies the namespace in URIs.

## Attribute Reference

* `name` - A descriptive name for the namespace.
* `slug` - The slug identifies the namespace in URIs.
* `slug_perm` - The slug_perm immutably identifies the namespace. It will never change once a namespace has been created.
* `type_name` - Is this a user or an organization namespace?.
