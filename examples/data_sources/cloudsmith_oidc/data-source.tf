# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_oidc" "my_oidc" {
  namespace = "my-organization"
  slug_perm = "my-oidc-provider-slug-perm"
}

output "mapping_claim" {
  value = data.cloudsmith_oidc.my_dynamic_oidc.mapping_claim
}

output "dynamic_mappings" {
  value = data.cloudsmith_oidc.my_dynamic_oidc.dynamic_mappings
}
