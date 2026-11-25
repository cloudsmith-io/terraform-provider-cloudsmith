# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

data "cloudsmith_organization" "my_organization" {
  slug = "my-organization"
}

data "cloudsmith_repository" "my_repository" {
  namespace  = data.cloudsmith_organization.my_organization.slug_perm
  identifier = "my-repository"
}

resource "cloudsmith_repository_upstream" "gradle_distributions" {
  name            = "Gradle Distributions"
  namespace       = data.cloudsmith_organization.my_organization.slug_perm
  repository      = resource.cloudsmith_repository.my_repository.id
  upstream_type   = "generic"
  upstream_url    = "https://services.gradle.org"
  upstream_prefix = "distributions"
}

# Strict mode applies only to Cache Only upstreams. The first download of each
# package waits until it has been cached and evaluated against your policies.
resource "cloudsmith_repository_upstream" "npmjs" {
  name          = "npmjs"
  namespace     = data.cloudsmith_organization.my_organization.slug_perm
  repository    = data.cloudsmith_repository.my_repository.slug_perm
  upstream_type = "npm"
  upstream_url  = "https://registry.npmjs.org"
  mode          = "Cache Only"
}

# GitHub skills upstreams require a GitHub access token and Cache Only mode.
variable "github_token" {
  type      = string
  sensitive = true
}

resource "cloudsmith_repository_upstream" "github_skills" {
  name          = "Official Claude plugins"
  namespace     = data.cloudsmith_organization.my_organization.slug_perm
  repository    = data.cloudsmith_repository.my_repository.slug_perm
  upstream_type = "skills"
  upstream_url  = "https://github.com/anthropics/claude-plugins-official"
  mode          = "Cache Only"
  auth_mode     = "Token"
  auth_secret   = var.github_token
}
