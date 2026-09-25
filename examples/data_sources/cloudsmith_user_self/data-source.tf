# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

data "cloudsmith_user_self" "current" {}

# Reference user attributes
output "current_user_email" {
  value = data.cloudsmith_user_self.current.email
}

output "current_user_slug" {
  value = data.cloudsmith_user_self.current.slug
}
