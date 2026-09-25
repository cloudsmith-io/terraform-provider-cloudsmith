# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

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

// quarantine_old_packages.rego
//
// 
// package cloudsmith.policy
// default allow := true
