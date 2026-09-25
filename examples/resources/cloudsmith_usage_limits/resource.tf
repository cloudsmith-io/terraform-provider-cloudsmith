# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

resource "cloudsmith_usage_limits" "example" {
  organization              = "my-organization"
  bandwidth_overage_limit   = 100
  storage_overage_limit     = 50
  allow_open_source_overage = true
}
