resource "cloudsmith_oidc" "org-oidc" {
  for_each         = var.repositories
  namespace        = data.cloudsmith_organization.cloudsmith-org.slug
  name             = "Github OIDC - ${each.key}"
  enabled          = true
  provider_url     = "https://token.actions.githubusercontent.com"
  service_accounts = [cloudsmith_service.ci-service[each.key].slug]
  claims           = (each.value.oidc_claims != null) ? each.value.oidc_claims : var.oidc_claims
}
