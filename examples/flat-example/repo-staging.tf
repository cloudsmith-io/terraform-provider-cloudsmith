resource "cloudsmith_repository" "staging" {
  description                = "Staging repository"
  name                       = "staging"
  namespace                  = data.cloudsmith_organization.org-demo.slug_perm
  slug                       = "staging"
  repository_type            = "Private"
  storage_region             = var.default_storage_region
  proxy_npmjs                = false
  proxy_pypi                 = false
  use_default_cargo_upstream = false
}

resource "cloudsmith_entitlement" "staging-main_entitlement" {
  namespace           = data.cloudsmith_organization.org-demo.slug
  name                = var.main_entitlement_token
  repository          = cloudsmith_repository.staging.slug
  limit_num_downloads = var.main_entitlement_token_limit_num_downloads
}

resource "cloudsmith_repository_privileges" "staging-privs" {
  organization = data.cloudsmith_organization.org-demo.slug
  repository   = cloudsmith_repository.staging.slug

  service {
    privilege = "Write"
    slug      = cloudsmith_service.devops-service.slug
  }

  service {
    privilege = "Write"
    slug      = cloudsmith_service.qa-service.slug
  }

  team {
    privilege = "Read"
    slug      = cloudsmith_team.developers.slug
  }

  # user {
  #   privilege = "Read"
  #   username  = "example-user"
  # }

}
