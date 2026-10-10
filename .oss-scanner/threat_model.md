# Threat model

## What this project does and where untrusted input enters

The Cloudsmith Terraform provider manages Cloudsmith resources (repositories, upstreams, entitlements, services, teams, webhooks, policies, SAML and OIDC settings) through the Cloudsmith REST API. Terraform starts it as a plugin on the operator's machine or on a CI runner. All provider code is in `cloudsmith/`.

Untrusted:

- Every API response body, status code and header, including error bodies and paginated lists (`cloudsmith/pagination.go`).
- Package download responses and redirects. `cloudsmith_package` with `download = true` fetches the CDN URL that the API returns and writes the file to `download_dir` (`cloudsmith/data_source_package.go`).
- OIDC token exchange responses from the API (`cloudsmith/provider_auth.go`).

Trusted: the Terraform configuration, provider attributes (`api_key`, `api_host`, `headers`, `oidc`), environment variables (`CLOUDSMITH_API_KEY`, `CLOUDSMITH_API_HOST`, `CLOUDSMITH_USE_OIDC`, `TFC_WORKLOAD_IDENTITY_TOKEN*`), and the operator's file system.

## Components that matter most / least

- Credential handling: `cloudsmith/provider.go`, `cloudsmith/provider_auth.go` and `cloudsmith/provider_config.go`. `apiKeyTransport` sends `X-Api-Key` only to the scheme and host of `api_host`. It strips the header on redirects to other hosts. Any path that sends the API key, the OIDC assertion or an exchanged token to a different host is a real bug.
- Package download: `downloadPackage` adds an `Authorization` header and builds the output path from the URL. Path traversal, writes outside `download_dir`, and credential leaks through redirects are in scope.
- Secrets in plan output, logs and state. These attributes carry secrets and must stay `Sensitive`: provider `api_key`, `cloudsmith_service.key`, `cloudsmith_entitlement.token`, `cloudsmith_webhook.secret_value` and `signature_key`, and `cloudsmith_repository_upstream.auth_secret`, `auth_certificate` and `auth_certificate_key`. Data sources that return keys or tokens (`data_source_service_details.go`, `data_source_service_list.go`, `data_entitlement.go`) must also mark them `Sensitive`. A secret in an error message, a `log.Printf` or `tflog` line, or a non-sensitive attribute is in scope.
- State correctness: a read that silently drops or changes a security setting (repository privileges, entitlement restrictions, geo/IP rules, policies, SAML enforcement) so that `terraform plan` shows no drift is in scope.
- Least important: documentation (`docs/`, `templates/`, `examples/`), tooling in `tools/`, and test fixtures.

## How to exercise it

- The image installs the provider at `/go/bin/terraform-provider-cloudsmith` and Terraform at `/usr/local/bin/terraform`. `/root/.terraformrc` has a `dev_overrides` entry, so `terraform plan` uses the local build with no registry access. Do not run `terraform init`.
- No network is available. Start a local HTTP server as a fake or malicious Cloudsmith API, and point the provider at it with `api_host` or `CLOUDSMITH_API_HOST`.
- Unit tests run with `go test -short ./...`. Acceptance tests need `TF_ACC=1` and a live Cloudsmith account, so they skip offline.

## How you rate severity

- Critical: code execution on the Terraform host triggered by an API response.
- High: the API key, OIDC assertion or exchanged token goes to a host other than `api_host`; a secret appears in plan output, logs or a non-sensitive state attribute; a file write outside `download_dir`.
- Medium: an API response causes a crash, a hang or unbounded memory or disk use; a read hides drift in a security setting.
- Low: misleading error messages; information leaks that expose no credential.

## Anything to leave alone

- Secrets in the Terraform state file. Terraform stores `Sensitive` values in state by design.
- An operator who sets `api_host` to a host they do not trust, or who adds secrets to `headers`.
- The `Authorization` header on the package download request to the CDN. The CDN requires it.
- Vulnerabilities in dependencies with no reachable path in the provider.
