# Organization Data Source

The `organization` data source allows fetching of metadata about a given Cloudsmith organization. The fetched data can be used to resolve permanent identifiers from an organization's user-facing name. These identifiers can then be passed to other resources to allow more consistent identification as user-facing names can change.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}
```

## Argument Reference

* `slug` - (Required) The slug identifies the organization in URIs.

## Attribute Reference

* `country` - Country in which the organization is based.
* `created_at` - ISO 8601 timestamp at which the organization was created.
* `location` - The city/town/area in which the organization is based.
* `name` - A descriptive name for the organization.
* `slug` - The slug identifies the organization in URIs.
* `slug_perm` - The slug_perm immutably identifies the organization. It will never change once a organization has been created.
* `tagline` - A short public description for the organization.
