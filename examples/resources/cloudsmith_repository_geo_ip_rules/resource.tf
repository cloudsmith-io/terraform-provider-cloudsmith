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

resource "cloudsmith_repository_geo_ip_rules" "my_rules" {
  namespace  = data.cloudsmith_organization.my_organization.slug_perm
  repository = resource.cloudsmith_repository.my_repository.slug_perm
  cidr_allow = [
    "10.0.0.0/24",
    "6cc2:ab98:2143:7e6e:8827:e81a:1527:9645/128",
    "140.59.25.1/32",
  ]
  cidr_deny = [
    "83.154.136.12/32",
    "203.0.0.0/10",
  ]
  country_code_allow = [
    "ST",
    "CM",
  ]
  country_code_deny = [
    "CA",
    "WF",
  ]
}
