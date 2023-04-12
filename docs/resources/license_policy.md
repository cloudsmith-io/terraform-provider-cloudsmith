# License Policy Resource

The license policy resource allows for creation and management of license policies within a Cloudsmith organization.

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
    namespace   = "my-organization"
    slug        = "my-repository"
}

resource "cloudsmith_license_policy" "my_license_policy" {
    name                    = "My Policy"
    description             = "My license policy"
    spdx_identifiers        = ["Apache-2.0"]
    on_violation_quarantine = true
    organization            = "my-organization"
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
terraform import cloudsmith_license_policy.my_policy my-organization
```
