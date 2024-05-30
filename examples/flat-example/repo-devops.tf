resource "cloudsmith_repository" "devops" {
  description                = "DevOps repository"
  name                       = "devops"
  namespace                  = data.cloudsmith_organization.org-demo.slug_perm
  slug                       = "devops"
  repository_type            = "Private"
  storage_region             = var.default_storage_region
  proxy_npmjs                = false
  proxy_pypi                 = false
  use_default_cargo_upstream = false
}


resource "cloudsmith_repository_privileges" "devops-privs" {
  organization = data.cloudsmith_organization.org-demo.slug
  repository   = cloudsmith_repository.devops.slug

  service {
    privilege = "Write"
    slug      = cloudsmith_service.qa-service.slug
  }

  service {
    privilege = "Read"
    slug      = cloudsmith_service.developer-service.slug
  }
}

resource "cloudsmith_repository_geo_ip_rules" "devops-geoip" {
  repository = cloudsmith_repository.devops.slug
  namespace = data.cloudsmith_organization.org-demo.slug
  country_code_allow = var.geopip_allow_countries
}