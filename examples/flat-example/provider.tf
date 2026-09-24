# Copyright Cloudsmith 2026
# SPDX-License-Identifier: MPL-2.0

# tflint-ignore-file: terraform_required_version, terraform_required_providers

terraform {
  required_providers {
    cloudsmith = {
      source = "cloudsmith-io/cloudsmith"
    }
  }
}

provider "cloudsmith" {
  api_key = var.api_key
}
