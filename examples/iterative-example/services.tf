resource "cloudsmith_service" "ci-service" {
  for_each     = var.repositories
  name         = "${lower(each.key)}-ci-service"
  organization = data.cloudsmith_organization.cloudsmith-org.slug
}
