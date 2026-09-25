# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

resource "cloudsmith_team" "developers" {
  organization = data.cloudsmith_organization.org-demo.slug_perm
  name         = "Developers"
}

resource "cloudsmith_team" "interns" {
  organization = data.cloudsmith_organization.org-demo.slug_perm
  name         = "Interns"
}
