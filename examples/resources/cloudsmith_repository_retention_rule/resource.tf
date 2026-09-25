# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

resource "cloudsmith_repository" "my_repository" {
  name      = "retention-rules"
  namespace = data.cloudsmith_organization.my_organization.slug_perm
}

resource "cloudsmith_repository_retention_rule" "retention_rule" {
  namespace                       = data.cloudsmith_organization.my_organization.slug
  repository                      = cloudsmith_repository.my_repository.slug
  retention_enabled               = true
  retention_count_limit           = 100
  retention_days_limit            = 28
  retention_group_by_name         = false
  retention_group_by_format       = false
  retention_group_by_package_type = false
  retention_size_limit            = 200000
}
