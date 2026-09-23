# Changelog

## Unreleased

### Added

* **provider:** OIDC authentication for HCP Terraform (Terraform Cloud). Enable it with an `oidc` block or `CLOUDSMITH_USE_OIDC=true`; the provider reads the workspace's workload identity token, exchanges it for a Cloudsmith token, refreshes that token before expiry, and retries once on 401. No `local-exec` or `external` data source needed. Other CI platforms are not supported yet. ([#128](https://github.com/cloudsmith-io/terraform-provider-cloudsmith/issues/128))
* **resource:** Add an `enabled` toggle to the SAML Group Sync resource.
* **resource:** Retention rules can now be managed as a `retention_rule` block on `cloudsmith_repository`. A repository accepts at most one block, applied via the retention partial update (PATCH) endpoint, and removing the block disables retention. Use of the standalone `cloudsmith_repository_retention_rule` resource is discouraged, as multiple instances can target the same repository and clobber each other.
