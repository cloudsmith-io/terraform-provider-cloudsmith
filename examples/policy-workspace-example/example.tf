# tflint-ignore-file: terraform_required_version, terraform_required_providers

terraform {
  required_providers {
    cloudsmith = {
      source = "cloudsmith-io/cloudsmith"
    }
  }
}

provider "cloudsmith" {
  api_key = var.api_key
}

variable "api_key" {
  type      = string
  sensitive = true
}

variable "organization_slug" {
  type        = string
  description = "Cloudsmith organization slug used for the workspace-scoped policy APIs."
}

locals {
  policy_prefix = "terraform-example-policy"
}

data "cloudsmith_organization" "workspace" {
  slug = var.organization_slug
}

resource "cloudsmith_policy" "quarantine_review" {
  workspace   = data.cloudsmith_organization.workspace.slug
  name        = "${local.policy_prefix}-quarantine-review"
  description = "Disabled by default; quarantine packages that arrive without the expected reviewed tag."
  enabled     = false
  is_terminal = false
  precedence  = 10
  rego        = <<-REGO
    package cloudsmith.policy

    default allow := false

    allow if {
      not reviewed
    }

    reviewed if {
      some tag in input.package.tags
      tag == "reviewed"
    }
  REGO
}

resource "cloudsmith_policy_action" "quarantine_review_state" {
  workspace        = data.cloudsmith_organization.workspace.slug
  policy_slug_perm = cloudsmith_policy.quarantine_review.slug_perm
  precedence       = 10

  set_package_state {
    package_state = "QUARANTINED"
  }
}

resource "cloudsmith_policy_action" "quarantine_review_tags" {
  workspace        = data.cloudsmith_organization.workspace.slug
  policy_slug_perm = cloudsmith_policy.quarantine_review.slug_perm
  precedence       = 20

  add_package_tags {
    tags = ["terraform-managed", "needs-review"]
  }
}

resource "cloudsmith_policy" "cleanup_tags" {
  workspace   = data.cloudsmith_organization.workspace.slug
  name        = "${local.policy_prefix}-cleanup-tags"
  description = "Disabled by default; remove transient ingestion tags after a package clears policy checks."
  enabled     = false
  is_terminal = true
  precedence  = 20
  rego        = <<-REGO
    package cloudsmith.policy

    default allow := false

    allow if {
      some tag in input.package.tags
      tag == "temporary-import"
    }
  REGO
}

resource "cloudsmith_policy_action" "cleanup_tags_remove" {
  workspace        = data.cloudsmith_organization.workspace.slug
  policy_slug_perm = cloudsmith_policy.cleanup_tags.slug_perm
  precedence       = 10

  remove_package_tags {
    tags = ["temporary-import"]
  }
}

resource "cloudsmith_policy_action" "cleanup_tags_remove_staging" {
  workspace        = data.cloudsmith_organization.workspace.slug
  policy_slug_perm = cloudsmith_policy.cleanup_tags.slug_perm
  precedence       = 20

  remove_package_tags {
    tags = ["staging-only", "pre-release"]
  }
}

resource "cloudsmith_policy" "cooldown" {
  workspace   = data.cloudsmith_organization.workspace.slug
  name        = "${local.policy_prefix}-cooldown"
  description = "Delay newly published packages from being available for a set period, giving the security community time to identify vulnerabilities"
  enabled     = false
  is_terminal = true
  precedence  = 0
  rego        = <<-REGO
    package cloudsmith

    import rego.v1

    default match := false

    within_past_days := 3

    supported_formats := {}

    included_repositories := {}

    excluded_repositories := {}

    include_local_packages := true

    match if count(reason) != 0

    _should_evaluate if {
        not input.v0.package.is_local
    }

    _should_evaluate if {
        include_local_packages == true
    }

    _repo_allowed if {
        count(included_repositories) == 0
        not input.v0.repository.slug in excluded_repositories
    }

    _repo_allowed if {
        input.v0.repository.slug in included_repositories
        not input.v0.repository.slug in excluded_repositories
    }

    reason contains msg if {
        _should_evaluate
        _repo_allowed
        pkg := input.v0.package
        within_past_days_date := time.add_date(time.now_ns(), 0, 0, 0 - within_past_days)
        publish_date := time.parse_rfc3339_ns(pkg.upstream_metadata.published_at)
        publish_date >= within_past_days_date
        pkg.format in supported_formats
        msg := sprintf(
            "Package %v/%v (%v) is within the %v-day cooldown period: published %v",
            [pkg.name, pkg.version, pkg.format, within_past_days, pkg.upstream_metadata.published_at],
        )
    }
  REGO
}

resource "cloudsmith_policy_action" "cooldown_hide" {
  workspace        = data.cloudsmith_organization.workspace.slug
  policy_slug_perm = cloudsmith_policy.cooldown.slug_perm
  precedence       = 0

  set_package_state {
    package_state = "HIDDEN"
  }
}

data "cloudsmith_policy" "quarantine_review" {
  workspace        = data.cloudsmith_organization.workspace.slug
  policy_slug_perm = cloudsmith_policy.quarantine_review.slug_perm
}

data "cloudsmith_policy_list" "terraform_examples" {
  workspace = data.cloudsmith_organization.workspace.slug
  query     = "name:${local.policy_prefix}*"
  sort      = "-created_at"
}

output "workspace" {
  value = data.cloudsmith_organization.workspace.slug
}

output "quarantine_review_policy" {
  value = {
    name      = data.cloudsmith_policy.quarantine_review.name
    slug_perm = data.cloudsmith_policy.quarantine_review.policy_slug_perm
    version   = data.cloudsmith_policy.quarantine_review.version
  }
}

output "terraform_example_policies" {
  value = [
    for policy in data.cloudsmith_policy_list.terraform_examples.policies : {
      name       = policy.name
      slug_perm  = policy.slug_perm
      precedence = policy.precedence
      enabled    = policy.enabled
    }
  ]
}
