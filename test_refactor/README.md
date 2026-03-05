# Test Refactor Summary

## Scope
This document summarizes the acceptance-test refactor work completed to improve reliability and enable higher parallelism in CI.

## What Was Broken

### Flaky resource behavior and state drift
- `cloudsmith_repository_retention_rule`: update checks intermittently read stale values (for example `retention_count_limit` reported `100` when config expected `0`).
- `cloudsmith_repository_upstream` (for example Conda/Huggingface): `is_active` sometimes read as `false` immediately after create when tests expected `true`.
- `cloudsmith_saml_auth`: `saml_metadata_inline` intermittently read back as empty string during update checks.
- `cloudsmith_manage_team`: create path could fail from duplicate auto-membership behavior.

### Name collisions and nondeterministic org-level tests
- OIDC tests reused fixed names and could fail with uniqueness errors (`name` must be unique).
- Multiple acceptance resources were using shared/static naming patterns in one org, increasing collision risk.

### Parallel execution instability
- At higher `go test -parallel` values, tests failed early with provider reattach startup timeouts (`timeout waiting on reattach config`).
- The test suite used shared provider instances (`Providers: testAccProviders`), which was not robust under concurrency.

### Over-strict data source assertion
- `TestAccOrganization_data` expected optional org profile fields (for example `location`) to always be non-empty.

## How It Was Fixed

### Provider/resource fixes
- Added post-update read polling for OIDC to wait for updated fields before state assertions.
- Added retention-rule update polling to ensure zero/non-zero transitions settle before read.
- Updated upstream create behavior to wait for expected default activation for non-Docker types.
- Updated SAML auth read behavior to preserve configured metadata state when API transiently omits metadata fields.
- Made manage-team create idempotent by using replace-members behavior.

### Acceptance test hardening
- Added unique test naming helper and adopted it in collision-prone OIDC and retention tests.
- Updated brittle attribute checks to compare service slugs via attribute pairing where needed.
- Replaced all acceptance test cases from shared `Providers` to `ProviderFactories` to create isolated provider instances.
- Added a dedicated check-time provider-config helper so destroy/existence checks no longer depend on shared `testAccProvider.Meta()`.
- Relaxed `TestAccOrganization_data` to assert only reliably populated attributes.

### CI workflow changes
- Acceptance workflow trigger scope updated to avoid duplicate org contention (`pull_request` plus `push` on `main`).
- Acceptance workflow test parallelism increased from `-parallel=6` to `-parallel=8`.

## Validation Evidence
- Targeted reruns for known flaky cases were repeated and passed.
- Resource acceptance matrix passed repeatedly at `-parallel=6`.
- Resource acceptance matrix passed repeatedly at `-parallel=8`.
- Exact workflow-style command succeeded:
  - `TF_ACC=1 go test -v ./... -parallel=8 -timeout=30m`

## Current Outcome
- Acceptance stability is significantly improved.
- Parallel execution is now validated at a higher level than before.
- Branch is cleaned of generated test artifacts and temporary local scripts.
