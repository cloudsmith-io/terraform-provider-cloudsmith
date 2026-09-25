# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
  slug = "my-organization"
}

resource "cloudsmith_service" "oidc_static" {
  name         = "oidc-static"
  organization = data.cloudsmith_organization.org.slug
}

resource "cloudsmith_oidc" "static" {
  namespace    = data.cloudsmith_organization.org.slug_perm
  name         = "My OIDC (static)"
  enabled      = true
  provider_url = "https://token.actions.githubusercontent.com/"
  service_accounts = [
    cloudsmith_service.oidc_static.slug,
  ]

  claims = {
    aud = "cloudsmith"
  }
}
