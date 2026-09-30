# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

data "cloudsmith_repository" "my_repository" {
  namespace  = data.cloudsmith_organization.my_organization.slug_perm
  identifier = "my-repository"
}
