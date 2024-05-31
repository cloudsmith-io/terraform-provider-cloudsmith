# Chainguard public repositories, no authentication, affix /chainguard/ to URL when pulling.
resource "cloudsmith_repository_upstream" "cgr-public" {
  for_each      = var.repositories
  name          = "cgr-public"
  namespace     = data.cloudsmith_organization.cloudsmith-org.slug_perm
  repository    = cloudsmith_repository.repositories[each.key].slug_perm
  is_active     = true
  upstream_type = "docker"
  upstream_url  = "https://cgr.dev"
  mode          = "Cache and Proxy"
  priority      = 2
}

# Chainguard private repositories, uses authentication, affix /<your-chainguard-id>/ to URL when pulling.
resource "cloudsmith_repository_upstream" "cgr-private" {
  for_each      = var.repositories
  name          = "cgr-private"
  namespace     = data.cloudsmith_organization.cloudsmith-org.slug_perm
  repository    = cloudsmith_repository.repositories[each.key].slug_perm
  is_active     = true
  upstream_type = "docker"
  upstream_url  = "https://cgr.dev"
  mode          = "Cache and Proxy"
  auth_mode     = "Username and Password"
  auth_username = var.chainguard_api_user
  auth_secret   = var.chainguard_api_secret
  priority      = 2
}

resource "cloudsmith_repository_upstream" "dockerhub" {
  for_each      = var.repositories
  name          = "dockerhub"
  namespace     = data.cloudsmith_organization.cloudsmith-org.slug_perm
  repository    = cloudsmith_repository.repositories[each.key].slug_perm
  upstream_type = "docker"
  upstream_url  = "https://index.docker.io"
  mode          = "Cache and Proxy"
  priority      = 1
}