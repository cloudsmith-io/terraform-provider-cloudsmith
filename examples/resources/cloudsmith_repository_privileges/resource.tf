# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

resource "cloudsmith_repository" "my_repository" {
  description = "A certifiably-awesome private package repository"
  name        = "My Repository"
  namespace   = data.cloudsmith_organization.my_organization.slug_perm
  slug        = "my-repository"
}

resource "cloudsmith_team" "my_team" {
  organization = data.cloudsmith_organization.my_organization.slug_perm
  name         = "My Team"
}

resource "cloudsmith_team" "my_other_team" {
  organization = data.cloudsmith_organization.my_organization.slug_perm
  name         = "My Other Team"
}

resource "cloudsmith_service" "my_service" {
  name         = "My Service"
  organization = data.cloudsmith_organization.my_organization.slug_perm
}

# This will return a slug for either service or user
data "cloudsmith_user_self" "current" {}

resource "cloudsmith_repository_privileges" "privs" {
  organization = data.cloudsmith_organization.my_organization.slug
  repository   = cloudsmith_repository.my_repository.slug

  ### Always include the authenticated account to avoid lockout (see note above)

  # user {
  #   privilege = "Admin"
  #   slug      = data.cloudsmith_user_self.current.slug
  # }

  # service {
  #    privilege = "Admin"
  #    slug      = data.cloudsmith_user_self.current.slug
  # }

  service {
    privilege = "Write"
    slug      = cloudsmith_service.my_service.slug
  }

  team {
    privilege = "Write"
    slug      = cloudsmith_team.my_team.slug
  }

  team {
    privilege = "Read"
    slug      = cloudsmith_team.my_other_team.slug
  }

  user {
    privilege = "Read"
    slug      = "some-user-slug"
  }
}
