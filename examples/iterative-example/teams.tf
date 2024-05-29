resource "cloudsmith_team" "developers" {
  organization = data.cloudsmith_organization.cloudsmith-org.slug_perm
  name         = "Developers"
}