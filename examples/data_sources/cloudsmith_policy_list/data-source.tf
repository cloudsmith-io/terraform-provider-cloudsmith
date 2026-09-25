# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

data "cloudsmith_policy_list" "named" {
  workspace = "my-workspace"
  query     = "name:quarantine*"
  sort      = "-created_at"
}
