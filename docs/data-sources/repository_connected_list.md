# Repository Connected List Data Source

The `cloudsmith_repository_connected_list` data source returns all connected repositories configured for a given source repository.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_repository_connected_list" "connections" {
    namespace  = "my-organization"
    repository = "source-repo"
}

output "connected_repositories" {
    value = data.cloudsmith_repository_connected_list.connections.connected_repositories
}
```

## Argument Reference

* `namespace` - (Required) Organization to which the source Repository belongs.
* `repository` - (Required) Source Repository (slug or slug_perm) whose connected repositories will be listed.

## Attribute Reference

* `connected_repositories` - A list of objects describing each connection:
  * `target_repository` - The slug of the connected target repository.
  * `is_active` - Whether the connection is active.
  * `priority` - The lookup order priority of the connection.
  * `slug_perm` - The immutable slug identifier of the connection.
  * `created_at` - The date and time at which the connection was created (RFC 3339).
