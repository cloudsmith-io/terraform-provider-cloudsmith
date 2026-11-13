# Changelog

## Unreleased

### Added

* **provider:** OIDC authentication for HCP Terraform (Terraform Cloud). Enable it with an `oidc` block or `CLOUDSMITH_USE_OIDC=true`; the provider reads the workspace's workload identity token, exchanges it for a Cloudsmith token, refreshes that token before expiry, and retries once on 401. No `local-exec` or `external` data source needed. Other CI platforms are not supported yet. ([#128](https://github.com/cloudsmith-io/terraform-provider-cloudsmith/issues/128))
* **resource:** Add an `enabled` toggle to the SAML Group Sync resource.
* **resource:** Retention rules can now be managed as a `retention_rule` block on `cloudsmith_repository`. A repository accepts at most one block, applied via the retention partial update (PATCH) endpoint, and removing the block disables retention. Use of the standalone `cloudsmith_repository_retention_rule` resource is discouraged, as multiple instances can target the same repository and clobber each other.

### Fixed

* **resource:** `cloudsmith_api_key_rule` supports `enforce_refresh` again. Cloudsmith restored automatic API key rotation, so the setting is no longer deprecated, `true` is accepted, and it is sent to the API on create and update. Omitting it leaves the rule's current setting unchanged.
* **docs:** Add a description and example usage for `cloudsmith_api_key_rule`.

## v0.0.90

### Breaking changes

* **resource:** A change to `name` on `cloudsmith_repository` now replaces the repository. Cloudsmith does not support a rename after creation. A replacement deletes the repository and all of its packages, so check the plan before you apply it.

### Added

* **resource:** `cloudsmith_policy_action` supports the `block_package` policy action, which fails a matching package at first publish rather than acting on it afterwards. The block takes no arguments; set `precedence` on the resource itself. This also fixes reads of a policy that already carries a BlockPackage action, which previously failed with `unsupported policy action type UNKNOWN`. The action requires a workspace with the policy block action enabled.

### Fixed

* **provider:** The provider sends a JWT credential, such as a SAML token from `cloudsmith auth`, in the `Authorization: Bearer` header. Before, it sent every credential in `X-Api-Key`, and the API rejected SAML tokens with a 401. API keys still go in `X-Api-Key`.
* **resource:** `cloudsmith_repository_upstream` waits while Cloudsmith validates a new upstream. Before, the create failed with `was disabled during activation: Pending validation`.
