# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_repository_connected_list" "connections" {
  namespace  = "my-organization"
  repository = "source-repo"
}

output "connected_repositories" {
  value = data.cloudsmith_repository_connected_list.connections.connected_repositories
}
