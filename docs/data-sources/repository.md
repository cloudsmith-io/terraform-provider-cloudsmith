# Repository Data Source

The `repository` data source allows for access to repository properties.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_namespace" "my_namespace" {
    slug = "my-namespace"
}

data "cloudsmith_repository" "my_repository" {
    namespace  = data.cloudsmith_namespace.my_namespace.slug_perm
    identifier = "my-repository"
}
```

## Argument Reference

* `namespace` - (Required) Namespace to which the repository belongs.
* `identifier` - (Required) An identifier used to resolve this repository. This can be the repository `slug`, or `slug_perm`.

## Attribute Reference

All of the argument attributes are also exported as result attributes.

Additionally, the following attributes are also exported:

* `cdn_url` - Base URL from which packages and other artifacts are downloaded.
* `created_at` - ISO 8601 timestamp at which the repository was created.
* `deleted_at` - ISO 8601 timestamp at which the repository was deleted.
* `description` - A description of the repository's purpose/contents.
* `index_files` - When `true`, package indexing is enabled for this repository.
* `namespace_url` - API endpoint to where data about this namespace can be retrieved.
* `repository_type` - A string describing the type of repository (Private, Public, Open-Source)
* `self_html_url` - The Cloudsmith web URL for this repository.
* `self_url` - The Cloudsmith API endpoint for this repository.
* `slug` - The slug identifies the repository in URIs.
* `slug_perm` - The internal immutable identifier for this repository.
* `storage_region` - The Cloudsmith region in which package files are stored.