provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
  slug = "my-organization"
}

data "cloudsmith_service_list" "services" {
  organization = data.cloudsmith_organization.org.slug_perm
}
