# Repository Resource

The repository resource allows creation and management of package repositories within a Cloudsmith namespace. Repositories store packages and are the main entities with which Cloudsmith users interact.

See [help.cloudsmith.io](https://help.cloudsmith.io/docs/manage-a-repository) for full repository documentation.

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
```

## Argument Reference

* `description` - (Optional) A description of the repository's purpose/contents.
* `index_files` - (Optional) If checked, files contained in packages will be indexed, which increase the synchronisation time required for packages. Note that it is recommended you keep this enabled unless the synchronisation time is significantly impacted.
* `name` - (Required) A descriptive name for the repository.
* `namespace` - (Required) Namespace to which this repository belongs.
* `repository_type` - (Optional) The repository type changes how it is accessed and billed. Private repositories can only be used on paid plans, but are visible only to you or authorised delegates. Public repositories are free to use on all plans and visible to all Cloudsmith users.
* `slug` - (Optional) The slug identifies the repository in URIs.
* `storage_region` - (Optional) The Cloudsmith region in which package files are stored.
* `wait_for_deletion` - (Optional) If true, terraform will wait for a repository to be permanently deleted before finishing.

## Attribute Reference

* `cdn_url` - Base URL from which packages and other artifacts are downloaded.
* `created_at` - ISO 8601 timestamp at which the repository was created.
* `deleted_at` - ISO 8601 timestamp at which the repository was deleted (repositories are soft deleted temporarily to allow cancelling).
* `description` - A description of the repository's purpose/contents.
* `index_files` - If checked, files contained in packages will be indexed, which increase the synchronisation time required for packages. Note that it is recommended you keep this enabled unless the synchronisation time is significantly impacted.
* `name` - A descriptive name for the repository.
* `namespace` - Namespace to which this repository belongs.
* `namespace_url` - API endpoint where data about this namespace can be retrieved.
* `repository_type` - The repository type changes how it is accessed and billed. Private repositories can only be used on paid plans, but are visible only to you or authorised delegates. Public repositories are free to use on all plans and visible to all Cloudsmith users.
* `self_html_url` - Website URL for this repository.
* `self_url` - API endpoint where data about this repository can be retrieved.
* `slug` - The slug identifies the repository in URIs.
* `slug_perm` - The slug_perm immutably identifies the repository. It will never change once a repository has been created.
* `storage_region` - The Cloudsmith region in which package files are stored.
* `wait_for_deletion` - If true, terraform will wait for a repository to be permanently deleted before finishing.
