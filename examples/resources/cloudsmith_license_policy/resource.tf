# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

resource "cloudsmith_license_policy" "my_license_policy" {
  name                    = "My Policy"
  description             = "My license policy"
  spdx_identifiers        = ["Apache-2.0"]
  on_violation_quarantine = true
  package_query_string    = "format:python AND downloads:>50"
  organization            = "my-organization"
}
