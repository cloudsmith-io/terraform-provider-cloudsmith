# OIDC Resource

The OIDC resource allows the creation and management of OpenID Connect (OIDC) configurations for a given Cloudsmith organization. OIDC configurations allow integration with external systems by providing OpenID Connect authentication.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

resource "cloudsmith_oidc" "my_oidc" {
    namespace  = data.cloudsmith_organization.my_organization.slug_perm
    name       = "My OIDC"
    enabled    = true
    provider_url = "https://example.com"
    service_accounts = ["account1", "account2"]
    claims = {
        "claim1" = "value1"
        "claim2" = "value2"
    }
}
```

## Argument Reference

* `claims` - (Required) The claims associated with these provider settings.
* `enabled` - (Required) Whether the provider settings should be used for incoming OIDC requests. Default is `true`.
* `name` - (Required) The name of the provider settings are being configured for.
* `namespace` - (Required) Namespace (or organization) to which this OIDC config belongs.
* `provider_url` - (Required) The URL from the provider that serves as the base for the OpenID configuration.
* `service_accounts` - (Required) The service accounts associated with these provider settings.
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
