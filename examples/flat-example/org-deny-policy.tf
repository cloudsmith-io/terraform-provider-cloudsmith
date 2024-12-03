resource "cloudsmith_package_deny_policy" "left_pad_policy" {
  name          = "Deny left-pad"
  description   = "Deny left-pad versions greater than 1.1.2"
  package_query = "format:npm AND name:left-pad AND version:>1.1.2"
  namespace     = data.cloudsmith_organization.org-demo.slug
}
