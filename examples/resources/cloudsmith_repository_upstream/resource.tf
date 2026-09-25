# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

resource "cloudsmith_repository_upstream" "gradle_distributions" {
  name            = "Gradle Distributions"
  namespace       = data.cloudsmith_organization.my_organization.slug_perm
  repository      = resource.cloudsmith_repository.my_repository.slug_perm
  upstream_type   = "generic"
  upstream_url    = "https://services.gradle.org"
  upstream_prefix = "distributions"
}
