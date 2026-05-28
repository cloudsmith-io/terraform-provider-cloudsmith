# Repository Connected Resource

The repository connected resource allows the management of connections between two Cloudsmith repositories within the same organization. A connection links a source repository to a target repository so that requests to the source can be resolved against the target as well, according to a configurable lookup priority.

By default, each source repository has a soft limit of 5 connected repositories. Contact Cloudsmith support if you need this limit increased.

See [docs.cloudsmith.com](https://docs.cloudsmith.com) for full Connected Repositories documentation.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

resource "cloudsmith_repository" "source" {
    description = "Source repository"
    name        = "source-repo"
    namespace   = data.cloudsmith_organization.my_organization.slug_perm
}

resource "cloudsmith_repository" "target" {
    description = "Target repository"
    name        = "target-repo"
    namespace   = data.cloudsmith_organization.my_organization.slug_perm
}

resource "cloudsmith_repository_connected" "link" {
    namespace         = cloudsmith_repository.source.namespace
    repository        = cloudsmith_repository.source.slug_perm
    target_repository = cloudsmith_repository.target.slug
    is_active         = true
    priority          = 1
}
```

## Argument Reference

The following arguments are supported:

* `namespace` - (Required) Organization to which the source Repository belongs. Changing this forces a new resource to be created.
* `repository` - (Required) Source Repository (slug or slug_perm) from which the connection is established. Changing this forces a new resource to be created.
* `target_repository` - (Required) The slug of the target Repository to connect to. Changing this forces a new resource to be created.
* `is_active` - (Optional) Whether the connection is active. Defaults to `true`.
* `priority` - (Optional) Lookup order priority. Repositories are checked in ascending order (starting at 1). When multiple connections share the same priority the oldest is used first. Must be between 1 and 32767.

## Attribute Reference

* `slug_perm` - The immutable slug identifier of the connection.
* `created_at` - The date and time at which the connection was created (RFC 3339).

## Import

This resource can be imported using the source namespace, source repository slug, and the connection `slug_perm`:

```shell
terraform import cloudsmith_repository_connected.link my-organization.source-repo.abcdef123456
```
