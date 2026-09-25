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

data "cloudsmith_entitlement_list" "my_tokens" {
  namespace  = resource.cloudsmith_repository.my_repository.namespace
  repository = resource.cloudsmith_repository.my_repository.slug_perm

  query = ["name:Default"]
}

resource "cloudsmith_entitlement_control" "my_entitlement_control" {
  namespace  = resource.cloudsmith_repository.my_repository.namespace
  repository = resource.cloudsmith_repository.my_repository.slug_perm
  identifier = data.cloudsmith_entitlement_list.my_tokens.entitlement_tokens[0].slug_perm
  enabled    = false
}
