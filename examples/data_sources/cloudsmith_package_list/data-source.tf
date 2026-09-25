# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_namespace" "my_namespace" {
  slug = "my-namespace"
}

data "cloudsmith_repository" "my_repository" {
  namespace  = data.cloudsmith_namespace.my_namespace.slug_perm
  identifier = "my-repository"
}

data "cloudsmith_package_list" "my_packages" {
  namespace  = data.cloudsmith_repository.my_repository.namespace
  repository = data.cloudsmith_repository.my_repository.slug_perm

  package_group = "my-package"
  filters       = ["format:docker"]
}

output "packages" {
  value = formatlist("%s-%s", data.cloudsmith_package_list.my_packages.packages.*.name, data.cloudsmith_package_List.my_packages.*.version)
}
