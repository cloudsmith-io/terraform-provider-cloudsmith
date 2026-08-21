# Cloudsmith Provider

This provider allows Cloudsmith users to automate the provisioning of resources using Terraform. Users can create and manage repositories, along with entitlement tokens to grant access to repository contents.

See [docs.cloudsmith.com](https://docs.cloudsmith.com/) for full documentation (including an API reference).

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_namespace" "my_namespace" {
    slug = "my-namespace"
}

resource "cloudsmith_repository" "my_repository" {
    description = "A certifiably-awesome private package repository"
    name        = "My Repository"
    namespace   = "${data.cloudsmith_namespace.my_namespace.slug_perm}"
    slug        = "my-repository"
}

resource "cloudsmith_entitlement" "my_entitlement" {
    name       = "Test Entitlement"
    namespace  = "${cloudsmith_repository.test.namespace}"
    repository = "${cloudsmith_repository.test.slug_perm}"
}
```

## Authenticate with OIDC

Authenticate the provider with a Cloudsmith OIDC service account instead of a static API key. **Only HCP Terraform (Terraform Cloud) workload identity is supported for now** - other CI platforms are planned but not implemented, so use `api_key` there.

Enable it with an `oidc` block, or by setting `CLOUDSMITH_USE_OIDC=true`. Do not set `api_key` in HCL. The identity token is never an argument and never lands in Terraform state.

This is provider *login*. It is not the [`cloudsmith_oidc`](resources/oidc.md) resource, which configures which identity providers Cloudsmith trusts and which claims a token must carry (including the `aud` your token is issued for). You still create that trust - with the resource, or in the Cloudsmith UI - before Terraform can exchange a token.

An `oidc` block (or `CLOUDSMITH_USE_OIDC`) ignores a leftover `CLOUDSMITH_API_KEY` environment variable, so an existing workspace can migrate without removing it first. An `api_key` argument in HCL still conflicts with `oidc`. An empty `provider "cloudsmith" {}` block does not enable OIDC unless `CLOUDSMITH_USE_OIDC` is true.

The provider reads the identity token from `TFC_WORKLOAD_IDENTITY_TOKEN_CLOUDSMITH`, then `TFC_WORKLOAD_IDENTITY_TOKEN`. It POSTs that token to the OpenID endpoint derived from `api_host` (the default production endpoint is `https://api.cloudsmith.io/openid/{organization}/`) and uses the returned Cloudsmith JWT as `X-Api-Key`. The JWT is refreshed a minute before its `exp` claim, or every 15 minutes if it carries no readable expiry. If a request still comes back 401, the provider discards that token, exchanges a fresh one, and replays the request once.

You can omit `organization` and `service_slug` when `CLOUDSMITH_ORG` and `CLOUDSMITH_SERVICE_SLUG` are set.

```hcl
provider "cloudsmith" {
    oidc {
        organization = "acme-org"
        service_slug = "tfc-prod"
    }
}
```

### HCP Terraform / Terraform Cloud setup

This replaces the `local-exec` and `data.external` workarounds previously needed to get a Cloudsmith token into a run.

1. In Cloudsmith, create a service account, then an OIDC provider for `https://app.terraform.io`. Require claims that match the workspace, at least `aud` and `terraform_organization_name`.
2. On the HCP Terraform workspace, set `TFC_WORKLOAD_IDENTITY_AUDIENCE` as an **environment** variable (not a Terraform variable) to the same `aud` value you configured in Cloudsmith, for example `cloudsmith`. HCP Terraform then injects `TFC_WORKLOAD_IDENTITY_TOKEN` into the run. To keep a Cloudsmith-specific audience alongside others, set `TFC_WORKLOAD_IDENTITY_AUDIENCE_CLOUDSMITH` instead and the token arrives as `TFC_WORKLOAD_IDENTITY_TOKEN_CLOUDSMITH`, which the provider prefers.
3. Configure the provider with the `oidc` block shown above, or set `CLOUDSMITH_USE_OIDC=true` together with `CLOUDSMITH_ORG` and `CLOUDSMITH_SERVICE_SLUG`.
4. Do not set `api_key` in HCL. A leftover `CLOUDSMITH_API_KEY` workspace variable is ignored, but delete it once the run works if you no longer want a static key in the workspace.

The identity token `aud` must match the `aud` claim on the Cloudsmith OIDC provider. If they disagree the exchange fails with a 401 and no Cloudsmith token is issued.

See [Manually generate workload identity tokens](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/manual-generation) and [Cloudsmith OpenID Connect](https://docs.cloudsmith.com/authentication/openid-connect).

## Argument Reference

* `api_key` - (Optional) The API key for authenticating with the Cloudsmith API. If you omit it and set neither `oidc` nor `CLOUDSMITH_USE_OIDC`, the provider reads `CLOUDSMITH_API_KEY`. Conflicts with `oidc`.
* `api_host` - (Optional) The API host to connect to (used to connect to a non-production Cloudsmith instance, mostly useful for testing).
* `oidc` - (Optional) Authenticate with a Cloudsmith OIDC service account using an HCP Terraform workload identity token. Conflicts with `api_key`. The identity token is not an argument. Ignores a leftover `CLOUDSMITH_API_KEY`. Equivalent env: `CLOUDSMITH_USE_OIDC=true`.
  * `organization` - Organization slug. Defaults to `CLOUDSMITH_ORG`.
  * `service_slug` - Service account slug. Defaults to `CLOUDSMITH_SERVICE_SLUG`.
