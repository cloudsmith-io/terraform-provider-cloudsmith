# Changelog

## Unreleased

### Added

* **provider:** OIDC authentication for HCP Terraform (Terraform Cloud). Enable it with an `oidc` block or `CLOUDSMITH_USE_OIDC=true`; the provider reads the workspace's workload identity token, exchanges it for a Cloudsmith token, refreshes that token before expiry, and retries once on 401. No `local-exec` or `external` data source needed. Other CI platforms are not supported yet. ([#128](https://github.com/cloudsmith-io/terraform-provider-cloudsmith/issues/128))
