provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
  slug = "my-organization"
}

data "cloudsmith_service_details" "service" {
  organization = data.cloudsmith_organization.org.slug_perm
  service      = "example-service"
}
