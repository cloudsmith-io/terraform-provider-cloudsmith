resource "cloudsmith_license_policy" "apache-python-policy" {
  name                    = "Example License Policy"
  description             = "Apache 2 License Policy for Python packages"
  spdx_identifiers        = ["Apache-2.0"]
  on_violation_quarantine = true
  package_query_string    = "format:python AND downloads:>50"
  organization            = data.cloudsmith_organization.org-demo.slug
}

resource "cloudsmith_license_policy" "mit-npm-policy" {
  name                    = "Example License Policy"
  description             = "MIT License Policy for Python packages"
  spdx_identifiers        = ["MIT"]
  on_violation_quarantine = true
  package_query_string    = "format:npm"
  organization            = data.cloudsmith_organization.org-demo.slug
}