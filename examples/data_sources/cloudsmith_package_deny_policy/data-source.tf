# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

resource "cloudsmith_package_deny_policy" "my_package_deny_policy" {
  namespace     = data.cloudsmith_organization.my_organization.slug_perm
  name          = "My Deny Policy"
  description   = "My package deny policy"
  package_query = "name:example"
  enabled       = true
}

data "cloudsmith_package_deny_policy" "existing" {
  namespace = cloudsmith_package_deny_policy.my_package_deny_policy.namespace
  slug_perm = cloudsmith_package_deny_policy.my_package_deny_policy.id
}
