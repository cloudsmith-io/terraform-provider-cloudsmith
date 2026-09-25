# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

resource "cloudsmith_repository" "test" {
  name      = "terraform-acc-test-package"
  namespace = "<your-namespace>"
}

data "cloudsmith_package_list" "test" {
  repository = cloudsmith_repository.test.name
  namespace  = cloudsmith_repository.test.namespace
  filters = [
    "name:dummy-package",
    "version:1.0.48",
  ]
}

data "cloudsmith_package" "test" {
  repository   = cloudsmith_repository.test.name
  namespace    = cloudsmith_repository.test.namespace
  identifier   = data.cloudsmith_package_list.test.packages[0].slug_perm
  download     = true
  download_dir = "/path/to/your/directory"
}
