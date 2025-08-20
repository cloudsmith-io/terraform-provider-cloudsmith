# Entitlement Control Resource

The entitlement control resource allows enabling and disabling of existing Entitlement tokens for a given Cloudsmith repository. This provides a way to manage the active state of entitlement tokens without modifying their other properties.

> ⚠️ **We highly recommend controlling the entitlement token with the [`cloudsmith_entitlement` resource](../resources/entitlement.md) and the `is_active` flag which controls the same setting. The purpose of this resource is to manage the "Default" entitlement token which is created by default for all repositories.**


See [docs.cloudsmith.com](https://docs.cloudsmith.com/software-distribution/entitlement-tokens) for full entitlement documentation.

## Example Usage 

> **Note:** Ensure `entitlement_tokens` array is returning entitlement tokens (and not an empty array) before using the control resource. By default the example should work, but if there were any changes made to repo settings, the expected behaviour might be different.

Disable repository "Default" entitlement token:

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

resource "cloudsmith_repository" "my_repository" {
    description = "A certifiably-awesome private package repository"
    name        = "My Repository"
    namespace   = "${data.cloudsmith_organization.my_organization.slug_perm}"
    slug        = "my-repository"
}

data "cloudsmith_entitlement_list" "my_tokens" {
    namespace  = resource.cloudsmith_repository.my_repository.namespace
    repository = resource.cloudsmith_repository.my_repository.slug_perm

    query        = ["name:Default"]
}

resource "cloudsmith_entitlement_control" "my_entitlement_control" {
    namespace  = resource.cloudsmith_repository.my_repository.namespace
    repository = resource.cloudsmith_repository.my_repository.slug_perm
    identifier = data.cloudsmith_entitlement_list.my_tokens.entitlement_tokens[0].slug_perm
    enabled    = false
}
```

## Argument Reference

* `namespace` - (Required) Namespace (or organization) to which this entitlement belongs.
* `repository` - (Required) Repository to which this entitlement belongs.
* `identifier` - (Required) The identifier (slug_perm) of the entitlement token to control.
* `enabled` - (Required) Whether the entitlement token should be enabled or disabled.

## Attribute Reference

* `namespace` - Namespace to which this entitlement belongs.
* `repository` - Repository to which this entitlement belongs.
* `identifier` - The identifier (slug_perm) of the entitlement token.
* `enabled` - Whether the entitlement token is enabled or disabled.

## Import

This resource can be imported using the organization slug, the repository slug, and the entitlement slug:

```shell
terraform import cloudsmith_entitlement_control.my_entitlement_control my-organization.my-repository.3nt1lem3nT
```
