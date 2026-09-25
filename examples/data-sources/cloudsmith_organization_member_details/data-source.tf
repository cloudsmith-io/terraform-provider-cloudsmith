# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

data "cloudsmith_org_member_details" "my_org_member_details" {
  organization = data.cloudsmith_organization.my_organization.slug
  member       = "username-or-email"
}
