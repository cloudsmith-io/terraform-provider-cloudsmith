# Policy List Data Source

List policies in a workspace, filtered by a required query. Paginates internally up to a hard cap of 10,000 results.

## Example Usage

```hcl
data "cloudsmith_policy_list" "named" {
    workspace = "my-workspace"
    query     = "name:quarantine*"
    sort      = "-created_at"
}
```

## Argument Reference

* `workspace` - (Required) The workspace the policies belong to.
* `query` - (Required) A search string limiting the results (e.g. `name:my-policy`).
* `sort` - (Optional) Comma-separated sort fields. Legal fields: `created_at`, `enabled`, `name`, `precedence`, `version`, `updated_at`. Prefix with `-` for descending. Defaults to `-created_at` on the server side when omitted.

## Attribute Reference

* `policies` - A list of policy objects. Each entry has the same attributes as [`data.cloudsmith_policy`](policy.md).

## See Also

* [`cloudsmith_policy`](../resources/policy.md)
* [`data.cloudsmith_policy`](policy.md)
