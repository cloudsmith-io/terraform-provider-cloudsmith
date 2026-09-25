# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
  slug = "my-organization"
}

data "cloudsmith_service_details" "service" {
  organization = data.cloudsmith_organization.org.slug_perm
  service      = cloudsmith_service.example.slug
}
