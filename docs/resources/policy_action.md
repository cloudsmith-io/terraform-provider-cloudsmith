# Policy Action Resource

Attaches an effect to a [`cloudsmith_policy`](policy.md). Each action is one of three typed shapes: `set_package_state`, `add_package_tags`, or `remove_package_tags`. Exactly one block must be set. Switching between them forces resource replacement, because the API exposes a separate typed resource per kind.

## Example Usage

```hcl
resource "cloudsmith_policy" "p" {
    workspace = "my-workspace"
    name      = "Example"
    rego      = file("${path.module}/policies/example.rego")
}

resource "cloudsmith_policy_action" "quarantine" {
    workspace        = "my-workspace"
    policy_slug_perm = cloudsmith_policy.p.slug_perm

    set_package_state {
        package_state = "QUARANTINED"
    }
}

resource "cloudsmith_policy_action" "tag" {
    workspace        = "my-workspace"
    policy_slug_perm = cloudsmith_policy.p.slug_perm
    precedence       = 10

    add_package_tags {
        tags = ["needs-review", "imported"]
    }
}
```

## Argument Reference

* `workspace` - (Required) Workspace the policy belongs to.
* `policy_slug_perm` - (Required, ForceNew) The `slug_perm` of the policy this action belongs to.
* `precedence` - (Optional) The order in which this action occurs relative to other actions for the same policy.
* `set_package_state` - (Optional) `package_state` must be one of `AVAILABLE`, `DELETED`, `QUARANTINED`, `HIDDEN`.
* `add_package_tags` - (Optional) `tags` is an unordered set of strings to add.
* `remove_package_tags` - (Optional) `tags` is an unordered set of strings to remove.

## Attribute Reference

* `slug_perm` - The unique permanent slug of the action.
* `created_at`, `updated_at` - RFC 3339 timestamps.

## Import

```shell
terraform import cloudsmith_policy_action.my_action my-workspace.policy-slug-perm.action-slug-perm
```

## See Also

* [`cloudsmith_policy`](policy.md)
