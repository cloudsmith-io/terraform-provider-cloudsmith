# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

resource "cloudsmith_saml" "my_saml" {
  organization = "my-organization"
  idp_key      = "role"
  idp_value    = "example"
  role         = "Member"
  enabled      = true
  team         = "owners"
}
