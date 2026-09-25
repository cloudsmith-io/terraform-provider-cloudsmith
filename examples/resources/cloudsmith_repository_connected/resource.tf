# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

resource "cloudsmith_repository" "source" {
  description = "Source repository"
  name        = "source-repo"
  namespace   = data.cloudsmith_organization.my_organization.slug_perm
}

resource "cloudsmith_repository" "target" {
  description = "Target repository"
  name        = "target-repo"
  namespace   = data.cloudsmith_organization.my_organization.slug_perm
}

resource "cloudsmith_repository_connected" "link" {
  namespace         = cloudsmith_repository.source.namespace
  repository        = cloudsmith_repository.source.slug_perm
  target_repository = cloudsmith_repository.target.slug
  is_active         = true
  priority          = 1
}
