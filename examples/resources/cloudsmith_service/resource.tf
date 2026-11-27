# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_org" {
  slug = "my-organization"
}

resource "cloudsmith_team" "my_team" {
  organization = data.cloudsmith_organization.my_org.slug
  name         = "My Team"
}

resource "cloudsmith_service" "my_service" {
  name         = "My Service"
  organization = data.cloudsmith_organization.my_org.slug

  team {
    slug = cloudsmith_team.my_team.slug
  }
}
