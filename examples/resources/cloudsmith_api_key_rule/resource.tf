# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

# User accounts: keys stop working after 90 days until the user rotates them.
resource "cloudsmith_api_key_rule" "users" {
  organization  = data.cloudsmith_organization.my_organization.slug_perm
  rule_type     = "User Accounts"
  max_age_hours = 2160
  is_enabled    = true
}

# Service accounts: keys older than 90 days are rotated automatically.
resource "cloudsmith_api_key_rule" "service_accounts" {
  organization    = data.cloudsmith_organization.my_organization.slug_perm
  rule_type       = "Service Accounts"
  max_age_hours   = 2160
  is_enabled      = true
  enforce_refresh = true
}
