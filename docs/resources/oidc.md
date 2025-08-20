# OIDC Resource

The OIDC resource allows the creation and management of OpenID Connect (OIDC) configurations for a given Cloudsmith organization. It supports either static service accounts or dynamic mappings from a claim value to service accounts.

Note: Dynamic mappings (`mapping_claim` and `dynamic_mappings`) are in early access; breaking changes are possible.

## Example Usage

### Static (service_accounts)

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
    slug = "my-organization"
}

resource "cloudsmith_service" "oidc_static" {
    name         = "oidc-static"
    organization = data.cloudsmith_organization.org.slug
}

resource "cloudsmith_oidc" "static" {
    namespace        = data.cloudsmith_organization.org.slug_perm
    name             = "My OIDC (static)"
    enabled          = true
    provider_url     = "https://token.actions.githubusercontent.com/"
    service_accounts = [
        cloudsmith_service.oidc_static.slug,
    ]

    claims = {
        aud = "cloudsmith"
    }
}
```

### Dynamic (mapping_claim + dynamic_mappings)

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
    slug = "my-organization"
}

resource "cloudsmith_service" "oidc_a" {
    name         = "oidc-account-a"
    organization = data.cloudsmith_organization.org.slug
}

resource "cloudsmith_service" "oidc_b" {
    name         = "oidc-account-b"
    organization = data.cloudsmith_organization.org.slug
}

resource "cloudsmith_oidc" "dynamic" {
    namespace     = data.cloudsmith_organization.org.slug_perm
    name          = "My OIDC (dynamic)"
    enabled       = true
    provider_url  = "https://token.actions.githubusercontent.com/"
    mapping_claim = "groups" # the claim key to inspect in the OIDC token

    dynamic_mappings = [
        {
            claim_value     = "team-a"
            service_account = cloudsmith_service.oidc_a.slug
        },
        {
            claim_value     = "team-b"
            service_account = cloudsmith_service.oidc_b.slug
        },
    ]

    claims = {
        aud = "cloudsmith"
    }
}
```

## Argument Reference

* `claims` - (Required) The set of claims that any received tokens from the provider must contain to authenticate as the configured service account.
* `enabled` - (Required) Whether the provider settings should be used for incoming OIDC requests.
* `name` - (Required) The name of the provider settings are being configured for.
* `namespace` - (Required) Namespace (or organization) to which this OIDC config belongs.
* `provider_url` - (Required) The URL from the provider that serves as the base for the OpenID configuration. For example, if the OpenID configuration is available at `https://token.actions.githubusercontent.com/.well-known/openid-configuration`, the provider URL would be `https://token.actions.githubusercontent.com/`.
* `service_accounts` - (Optional) Static provider: list of service account slugs. Cannot be provided if `mapping_claim` or `dynamic_mappings` are specified.
* `mapping_claim` - (Optional) Dynamic provider: the OIDC claim to use for mapping to service accounts in `dynamic_mappings`. Cannot be provided if `service_accounts` is also set.
* `dynamic_mappings` - (Optional) Dynamic provider: set of mappings from `mapping_claim` values to service accounts. Cannot be provided if `service_accounts` is also set.
    * `claim_value` (String, Required) - Non-empty value of the `mapping_claim` to match.
    * `service_account` (String, Required) - Non-empty service account slug to authenticate as.
* `slug` - (Computed) The slug identifies the OIDC.
* `slug_perm` - (Computed) The slug_perm identifies the OIDC.

## Attribute Reference

* `slug` - The slug identifies the OIDC.
* `slug_perm` - The slug_perm identifies the OIDC.

## Import

This resource can be imported using the organization slug and the OIDC slug_perm:

```shell
terraform import cloudsmith_oidc.my_oidc my-organization.my-oidc-slug-perm
```
