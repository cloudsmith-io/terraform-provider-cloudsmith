resource "cloudsmith_saml" "owners_mapping" {
  organization = data.cloudsmith_organization.org-demo.slug
  idp_key      = "administrators"
  idp_value    = "administrators"
  role         = "Manager"
  team         = "owners"
}

resource "cloudsmith_saml" "developers_mapping" {
  organization = data.cloudsmith_organization.org-demo.slug
  idp_key      = "interns"
  idp_value    = "interns"
  role         = "Member"
  team         = resource.cloudsmith_team.interns.slug
}
