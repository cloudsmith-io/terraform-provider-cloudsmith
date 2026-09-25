# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

resource "cloudsmith_team" "developers" {
  organization = data.cloudsmith_organization.cloudsmith-org.slug_perm
  name         = "Developers"
}
