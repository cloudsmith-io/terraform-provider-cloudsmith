resource "cloudsmith_service" "devops-service" {
  name         = "devops-service"
  organization = data.cloudsmith_organization.org-demo.slug
}

resource "cloudsmith_service" "production-service" {
  name         = "production-service"
  organization = data.cloudsmith_organization.org-demo.slug
}

resource "cloudsmith_service" "qa-service" {
  name         = "qa-service"
  organization = data.cloudsmith_organization.org-demo.slug
}

resource "cloudsmith_service" "developer-service" {
  name         = "developer-service"
  organization = data.cloudsmith_organization.org-demo.slug
}
