# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

resource "cloudsmith_repository" "my_repository" {
  description = "A certifiably-awesome private package repository"
  name        = "My Repository"
  namespace   = data.cloudsmith_organization.my_organization.slug_perm
  slug        = "my-repository"
}

resource "cloudsmith_webhook" "my_webhook" {
  namespace  = cloudsmith_repository.my_repository.namespace
  repository = cloudsmith_repository.my_repository.slug_perm

  events              = ["package.created", "package.deleted"]
  request_body_format = "Handlebars Template"
  target_url          = "https://example.com"

  template {
    event    = "package.created"
    template = "created: {{data.name}}: {{data.version}}"
  }

  template {
    event    = "package.deleted"
    template = "deleted: {{data.name}}: {{data.version}}"
  }
}
