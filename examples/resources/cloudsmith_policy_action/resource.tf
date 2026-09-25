# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

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
