# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_usage_limits" "example" {
  organization = "my-organization"
}

output "package_delivery_limit" {
  value = data.cloudsmith_usage_limits.example.bandwidth_overage_limit
}
