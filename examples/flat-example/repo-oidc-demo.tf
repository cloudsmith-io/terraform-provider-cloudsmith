resource "cloudsmith_repository" "oidc_demo" {
  description                = "OIDC repository"
  name                       = "oidc-demo"
  namespace                  = data.cloudsmith_organization.org-demo.slug_perm
  slug                       = "oidc"
  repository_type            = "Private"
  storage_region             = var.default_storage_region
  proxy_npmjs                = false
  proxy_pypi                 = false
  use_default_cargo_upstream = false
}


resource "cloudsmith_repository_privileges" "oidc_demo-privs" {
  organization = data.cloudsmith_organization.org-demo.slug
  repository   = cloudsmith_repository.oidc_demo.slug

  service {
    privilege = "Write"
    slug      = cloudsmith_service.developer-service.slug
  }
}
