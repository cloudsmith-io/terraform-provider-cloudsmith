resource "cloudsmith_repository" "repositories" {
  for_each                  = var.repositories
  description               = "${title(each.key)} repository"
  name                      = each.key
  namespace                 = data.cloudsmith_organization.cloudsmith-org.slug_perm
  slug                      = each.key
  repository_type           = "Private"
  storage_region            = var.default_storage_region
  user_entitlements_enabled = false

}

resource "cloudsmith_repository_privileges" "repo-privs" {
  for_each     = var.repositories
  organization = data.cloudsmith_organization.cloudsmith-org.slug
  repository   = cloudsmith_repository.repositories[each.key].slug

  # if you're using a service account to provision, be sure to include it as an Admin here!
  service {
    privilege = "Write"
    slug      = cloudsmith_service.ci-service[each.key].slug
  }

  dynamic "team" {
    for_each = each.value.add_developers == true ? [1] : []
    content {
      privilege = "Write"
      slug      = cloudsmith_team.developers.slug
    }
  }
}
