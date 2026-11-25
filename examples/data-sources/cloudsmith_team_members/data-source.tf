provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
  slug = "my-organization"
}

data "cloudsmith_team_members" "teams" {
  organization = data.cloudsmith_organization.org.slug_perm
  team_name    = "example"
}
