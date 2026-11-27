# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

data "cloudsmith_list_org_members" "members" {
  namespace = "my-organization"
}
