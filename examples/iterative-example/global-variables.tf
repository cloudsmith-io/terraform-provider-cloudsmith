data "cloudsmith_organization" "cloudsmith-org" {
  slug = "YOUR-ORG-NAME"
}

variable "api_key" {
  type    = string
  default = "YOUR-API-KEY"
}

variable "default_storage_region" {
  type    = string
  default = "us-ohio"
}

variable "chainguard_api_user" {
  type    = string
  default = "YOUR-CHAINGUARD-API-USER"
}

variable "chainguard_api_secret" {
  type    = string
  default = "YOUR-CHAINGUARD-API-SECRET"
}

variable "repositories" {
  type = map(object({
    add_developers = optional(bool)
    oidc_claims    = optional(map(string))
  }))
  description = "A map of repositories with their configurations."
  default = {
    "staging" = {
      add_developers = false
    },
    "production" = {
      add_developers = false
    }
  }
}

variable "oidc_claims" {
  type = map(string)
  default = {
    "repository" = "Owner/GitHubRepoName"
  }
}
