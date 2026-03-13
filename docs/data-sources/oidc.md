# OIDC Data Source

The `oidc` data source allows fetching of metadata about a given Cloudsmith OIDC (OpenID Connect) provider configuration. This can be used to retrieve information about existing OIDC configurations for use in other Terraform resources or to verify configuration details.

## Example Usage

### Static Provider

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_oidc" "my_oidc" {
    namespace = "my-organization"
    slug_perm = "my-oidc-provider-slug-perm"
}
```

### Dynamic Provider

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_oidc" "my_dynamic_oidc" {
    namespace = "my-organization"
    slug_perm = "my-dynamic-oidc-provider-slug-perm"
}

output "mapping_claim" {
    value = data.cloudsmith_oidc.my_dynamic_oidc.mapping_claim
}

output "dynamic_mappings" {
    value = data.cloudsmith_oidc.my_dynamic_oidc.dynamic_mappings
}
```

## Argument Reference

* `namespace` - (Required) Namespace (or organization) to which this OIDC config belongs.
* `slug_perm` - (Required) The slug_perm identifies the OIDC.

## Attribute Reference

* `claims` - The claims associated with these provider settings.
* `enabled` - Whether the provider settings should be used for incoming OIDC requests.
* `name` - The name of the provider settings are being configured for.
* `provider_url` - The URL from the provider that serves as the base for the OpenID configuration.
* `service_accounts` - The service accounts associated with these provider settings (static providers only).
* `mapping_claim` - The claim key whose values dynamically map to service accounts (dynamic providers only).
* `dynamic_mappings` - Set of dynamic claim value -> service account mappings. Each mapping contains:
    * `claim_value` - The value of the mapping claim.
    * `service_account` - Service account slug mapped to the claim value.
* `slug` - The slug identifies the OIDC.
* `slug_perm` - The slug_perm identifies the OIDC.