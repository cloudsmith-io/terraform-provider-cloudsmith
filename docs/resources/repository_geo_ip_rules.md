# Respository Geo/IP Rules Resource

The repository geo/ip rules resource allows the management of geo/ip rules for a given Cloudsmith repository. Using this resource it is possible to allow and/or deny access to a repository using CIDR notation, two-character ISO 3166-1 country codes or a combination thereof.

See [help.cloudsmith.io](https://help.cloudsmith.io/docs/geoip-restriction) for full geo/ip rules documentation.

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

resource "cloudsmith_repository_geo_ip_rules" "my_rules" {
    namespace          = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository         = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    cidr_allow         = [
      "10.0.0.0/24",
      "6cc2:ab98:2143:7e6e:8827:e81a:1527:9645/128",
      "140.59.25.1/32",
    ]
    cidr_deny          = [
      "83.154.136.12/32",
      "203.0.0.0/10",
    ]
    country_code_allow = [
      "ST",
      "CM",
    ]
    country_code_deny  = [
      "CA",
      "WF",
    ]
}
```

## Argument Reference

The following arguments are supported:

* `namespace` - (Required) Organization to which the Repository belongs.
* `repository` - (Required) Repository to which these Geo/IP rules apply.
* `cidr_allow` - (Optional) The list of IP Addresses for which to allow access to the Repository, expressed in CIDR notation.
* `cidr_deny` - (Optional) The list of IP Addresses for which to deny access to the Repository, expressed in CIDR notation.
* `country_code_allow` - (Optional) The list of countries for which to allow access to the Repository, expressed in ISO 3166-1 country codes.
* `country_code_deny` - (Optional) The list of countries for which to deny access to the Repository, expressed in ISO 3166-1 country codes.

## Import

This resource can be imported using the organization slug, and the repository slug:

```shell
terraform import cloudsmith_repository_geo_ip_rules.my_rules my-organization.my-repository
```
