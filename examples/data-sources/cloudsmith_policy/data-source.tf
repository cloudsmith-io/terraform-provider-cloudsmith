# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

data "cloudsmith_policy" "p" {
  workspace        = "my-workspace"
  policy_slug_perm = "abcdef"
}

output "policy_name" {
  value = data.cloudsmith_policy.p.name
}
