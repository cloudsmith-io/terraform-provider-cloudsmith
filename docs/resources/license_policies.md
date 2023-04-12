# License Policies Resource

The repository geo/ip rules resource allows the management of geo/ip rules for a given Cloudsmith repository. Using this resource it is possible to allow and/or deny access to a repository using CIDR notation, two-character ISO 3166-1 country codes or a combination thereof.

The license policies resource allows the management of license policies for a given cloudsmith organization. This resource allows creation and management of license policies within a Cloudsmith organization

See [help.cloudsmith.io](https://help.cloudsmith.io/docs/license-policies) for the full license policies documentation.

## Example Usage

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

resource "cloudsmith_license_policies" "my_license_policies" {
    name                    = "TF Test Policy Updated"
    description             = "TF Test Policy Description Updated"
    spdx_identifiers        = ["Apache-2.0"]
    on_violation_quarantine = true
    organization            = "%s"
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Required) Organization to which the policy belongs.
* `name` - (Required) The name of the license policy.
* `description` - (Required) The description of the license policy.
* `spdx_identifiers` - (Required) The licenses to deny.
* `on_violation_quarantine` - (Optional) On violation of the license policy, quarantine violating packages.
* `allow_unknown_licenses` - (Optional) Allow unknown licenses within the policy.

## Import

This resource can be imported using the organization slug.

```shell
terraform import cloudsmith_license_policies.my_policies my-organization
```
