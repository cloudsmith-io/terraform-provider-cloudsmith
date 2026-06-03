# Policy Data Source

Fetch a single policy by its `slug_perm`.

## Example Usage

```hcl
data "cloudsmith_policy" "p" {
    workspace        = "my-workspace"
    policy_slug_perm = "abcdef"
}

output "policy_name" {
    value = data.cloudsmith_policy.p.name
}
```

## Argument Reference

* `workspace` - (Required) The workspace the policy belongs to.
* `policy_slug_perm` - (Required) The unique permanent slug of the policy.

## Attribute Reference

* `name` - The name of the policy.
* `description` - The description of the policy.
* `rego` - The Rego source for the policy logic.
* `enabled` - If true, the policy is enabled.
* `is_terminal` - If true and the policy matches, no further policies are evaluated.
* `precedence` - The order in which this policy is evaluated relative to other policies.
* `version` - The version of the Rego code.
* `read_only` - Whether the policy is read-only.
* `created_at` - The time the policy was created.
* `updated_at` - The time the policy was last updated.

## See Also

* [`cloudsmith_policy`](../resources/policy.md)
* [`data.cloudsmith_policy_list`](policy_list.md)
