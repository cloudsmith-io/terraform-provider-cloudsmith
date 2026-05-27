# Policy Resource

Manages a workspace policy in Cloudsmith. The body is authored as [rego](https://www.openpolicyagent.org/docs/latest/policy-language/) and evaluated against package events. Effects are attached via [`cloudsmith_policy_action`](policy_action.md).

## Example Usage

```hcl
resource "cloudsmith_policy" "quarantine_old_packages" {
    workspace   = "my-workspace"
    name        = "Quarantine old packages"
    description = "Quarantines anything older than 12 months."
    enabled     = true
    is_terminal = false
    rego        = file("${path.module}/policies/quarantine_old_packages.rego")
}

resource "cloudsmith_policy_action" "quarantine" {
    workspace        = "my-workspace"
    policy_slug_perm = cloudsmith_policy.quarantine_old_packages.slug_perm

    set_package_state {
        package_state = "QUARANTINED"
    }
}
```

`policies/quarantine_old_packages.rego`:

```rego
package cloudsmith.policy
default allow := true
```

## Argument Reference

* `workspace` - (Required) Workspace the policy belongs to.
* `name` - (Required) The name of the policy.
* `description` - (Optional) The description of the policy.
* `rego` - (Required) The rego source for the policy logic.
* `enabled` - (Optional, default `true`) If `true`, the policy is enabled.
* `is_terminal` - (Optional, default `false`) If `true` and the policy matches, no further policies are evaluated.
* `precedence` - (Optional) The order in which this policy is evaluated relative to other policies.

## Attribute Reference

* `slug_perm` - The unique permanent slug of the policy.
* `version` - The version of the rego code.
* `read_only` - Whether the policy is read-only. Read-only policies cannot be updated through this provider; use `terraform state rm` or `lifecycle { ignore_changes = [...] }`.
* `created_at`, `updated_at` - RFC 3339 timestamps.

## Import

```shell
terraform import cloudsmith_policy.my_policy my-workspace.policy-slug-perm
```

## See Also

* [`cloudsmith_policy_action`](policy_action.md)
* [`data.cloudsmith_policy`](../data-sources/policy.md)
* [`data.cloudsmith_policy_list`](../data-sources/policy_list.md)
