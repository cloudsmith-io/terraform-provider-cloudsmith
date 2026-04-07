# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Terraform provider for Cloudsmith — a package/artifact management platform. Built with Terraform Plugin SDK v2 and the `cloudsmith-api-go` generated API client (v0.0.55).

## Build & Test Commands

```bash
# Build
go build ./...

# Run all unit tests
go test -v ./...

# Run acceptance tests (requires API credentials)
TF_ACC=1 go test -v ./... -parallel=30 -timeout=10m

# Run a single test
go test -v -run=TestAccRepository_basic ./...

# Lint
golangci-lint run
```

### Acceptance Test Environment Variables

- `TF_ACC=1` — required to enable acceptance tests
- `CLOUDSMITH_API_KEY` — API key for Cloudsmith
- `CLOUDSMITH_NAMESPACE` — organization namespace for test resources (CI uses `terraform-provider-testing`)

## Architecture

All provider code lives in `cloudsmith/`. Entry point is `main.go`.

### Key Files

- `provider.go` — registers all resources and data sources in `ResourcesMap`/`DataSourcesMap`
- `provider_config.go` — `providerConfig` struct holds `Auth` (context with API key) and `APIClient`; validates credentials on init via `UserSelf` call
- `utils.go` — shared helpers: `optionalString`, `requiredBool`, `expandStrings`, `flattenStrings`, `waitForCreation`, `waitForDeletion`, `waitForUpdate`, nullable type converters

### Resource Pattern

Every resource follows this structure:

1. **Import function** — parses dot-separated IDs (e.g., `namespace.slug` or `namespace.repo.entitlement_slug`)
2. **CRUD functions** — `resourceXxxCreate`, `resourceXxxRead`, `resourceXxxUpdate`, `resourceXxxDelete`
3. **Schema function** — returns `*schema.Resource` with field definitions
4. **Wait for consistency** — after Create/Delete, poll with `waitForCreation`/`waitForDeletion`; after Update, call `waitForUpdate` (7s fixed delay)

API client access pattern in all CRUD functions:
```go
pc := m.(*providerConfig)
req := pc.APIClient.SomeApi.SomeMethod(pc.Auth, ...)
result, resp, err := pc.APIClient.SomeApi.SomeMethodExecute(req)
```

### Test Pattern

Tests use `//nolint:testpackage` and are in the `cloudsmith` package (not `cloudsmith_test`) to access internal helpers. Key test utilities in `provider_test.go`:
- `testAccPreCheck(t)` — validates env vars
- `testAccUniqueRepositoryName(base)` — generates unique repo names (max 50 chars) to avoid collisions in parallel test runs
- `testAccProviders` / `testAccProvider` — shared provider instances

Tests use multi-step `resource.TestCase` with Config strings, Check functions, ExpectError, and ImportState verification.

### Notable Resources

- `resource_repository_upstream.go` (~1500 lines) — most complex resource; polymorphic across upstream types (Alpine, Cargo, Composer, CRAN, Dart, Docker, Helm, Maven, npm, NuGet, Python, RPM, Ruby, Swift, Terraform) using an `Upstream` interface
- `resource_repository.go` — core repository resource with 40+ configurable fields

## Conventions

- Schema field names use `snake_case` matching Terraform conventions
- Enum fields use `validation.StringInSlice()` for validation
- Resource types: `cloudsmith_<resource_type>`
- Lint config (`.golangci.yml`): `d.Set` errcheck is suppressed; timeout is 5m
- Go version: 1.26
