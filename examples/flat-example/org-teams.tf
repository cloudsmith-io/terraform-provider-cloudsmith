resource "cloudsmith_team" "developers" {
  organization = data.cloudsmith_organization.org-demo.slug_perm
  name         = "Developers"
}

resource "cloudsmith_team" "interns" {
  organization = data.cloudsmith_organization.org-demo.slug_perm
  name         = "Interns"
}
