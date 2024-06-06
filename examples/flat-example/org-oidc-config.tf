resource "cloudsmith_oidc" "devops-oidc" {
    namespace  = data.cloudsmith_organization.org-demo.slug
    name       = "OIDC-DEMO"
    enabled    = true
    provider_url = "https://token.actions.githubusercontent.com"
    service_accounts = [cloudsmith_service.production-service.slug]
    claims = {
        "repository" = "Owner/GitHubRepoName"
    }
}