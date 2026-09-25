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

resource "cloudsmith_entitlement" "my_entitlement" {
  name       = "Test Entitlement"
  namespace  = cloudsmith_repository.test.namespace
  repository = cloudsmith_repository.test.slug_perm
}
