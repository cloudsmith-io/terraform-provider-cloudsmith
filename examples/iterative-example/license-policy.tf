resource "cloudsmith_license_policy" "agpl-policy" {
  name                    = "Block AGPL"
  description             = "Block AGPL licensed packages"
  spdx_identifiers        = ["AGPL-1.0", "AGPL-1.0-only", "AGPL-1.0-or-later", "AGPL-3.0", "AGPL-3.0-only", "AGPL-3.0-or-later"]
  on_violation_quarantine = true
  organization            = data.cloudsmith_organization.cloudsmith-org.slug
}