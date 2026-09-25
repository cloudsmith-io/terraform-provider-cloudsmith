# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

resource "cloudsmith_repository" "test" {
  name      = "terraform-acc-test-privileges"
  namespace = "<your-namespace>"
}

data "cloudsmith_repository_privileges" "test_data" {
  organization = cloudsmith_repository.test.namespace
  repository   = cloudsmith_repository.test.slug
}
