provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "my_org" {
  slug = "my-organization"
}

resource "cloudsmith_team" "my_team" {
  organization = data.cloudsmith_organization.my_org.slug_perm
  name         = "My Team"
}
