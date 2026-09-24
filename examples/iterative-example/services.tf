# Copyright Cloudsmith 2026
# SPDX-License-Identifier: MPL-2.0

resource "cloudsmith_service" "ci-service" {
  for_each     = var.repositories
  name         = "${lower(each.key)}-ci-service"
  organization = data.cloudsmith_organization.cloudsmith-org.slug
}
