resource "cloudsmith_repository" "upstream" {
  description       = "Global upstream proxy repository for docker, nuget, python, npm and maven"
  name              = "upstreams-demo"
  namespace         = data.cloudsmith_organization.org-demo.slug_perm
  slug              = "upstreams-demo"
  repository_type   = "Private"
  storage_region    = var.default_storage_region
  default_privilege = "Read"
}

resource "cloudsmith_repository_upstream" "pypi" {
  name          = "pypi"
  namespace     = data.cloudsmith_organization.org-demo.slug_perm
  repository    = cloudsmith_repository.upstream.slug_perm
  upstream_type = "python"
  upstream_url  = "https://pypi.org"
  mode          = "Cache and Proxy"
}

resource "cloudsmith_repository_upstream" "npm" {
  name          = "npm"
  namespace     = data.cloudsmith_organization.org-demo.slug_perm
  repository    = cloudsmith_repository.upstream.slug_perm
  upstream_type = "npm"
  upstream_url  = "https://registry.npmjs.org"
  mode          = "Cache and Proxy"
}

resource "cloudsmith_repository_upstream" "nuget" {
  name          = "nuget.org"
  namespace     = data.cloudsmith_organization.org-demo.slug_perm
  repository    = cloudsmith_repository.upstream.slug_perm
  upstream_type = "nuget"
  upstream_url  = "https://api.nuget.org/v3/index.json"
  mode          = "Cache and Proxy"
}

resource "cloudsmith_repository_upstream" "dockerhub" {
  name          = "dockerhub"
  namespace     = data.cloudsmith_organization.org-demo.slug_perm
  repository    = cloudsmith_repository.upstream.slug_perm
  upstream_type = "docker"
  upstream_url  = "https://index.docker.io"
  mode          = "Cache and Proxy"
}

resource "cloudsmith_repository_upstream" "mcr-microsoft" {
  name          = "mcr.microsoft.com"
  namespace     = data.cloudsmith_organization.org-demo.slug_perm
  repository    = cloudsmith_repository.upstream.slug_perm
  upstream_type = "docker"
  upstream_url  = "https://mcr.microsoft.com"
  mode          = "Cache and Proxy"
}

resource "cloudsmith_repository_upstream" "maven" {
  name          = "Maven"
  namespace     = data.cloudsmith_organization.org-demo.slug_perm
  repository    = cloudsmith_repository.upstream.slug_perm
  upstream_type = "maven"
  upstream_url  = "https://repo1.maven.org/maven2"
  mode          = "Cache and Proxy"
}
