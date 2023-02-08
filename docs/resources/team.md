# Team Resource

The teams resource allows creation and management of teams within a Cloudsmith organization.

See [help.cloudsmith.io](https://help.cloudsmith.io/docs/teams) for full team documentation.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_org" {
    slug = "my-organization"
}

resource "cloudsmith_team" "my_team" {
    organization = data.cloudsmith_organization.my_org.slug_perm
    name         = "My Team"
}
```

## Argument Reference

The following arguments are supported:

* `description` - (Optional) A description of the team's purpose.
* `name` - (Required) A descriptive name for the team.
* `organization` - (Required) Organization to which this team belongs.
* `slug` - (Optional) The slug identifies the team in URIs.
* `visibility` - (Optional) Controls if the team is visible or hidden from non-members.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `slug_perm` - The slug_perm immutably identifies the team. It will never change once a team has been created.

## Import

This resource can be imported using the organization slug, and the team slug:

```shell
terraform import cloudsmith_team.my_team my-organization.my-team
```
