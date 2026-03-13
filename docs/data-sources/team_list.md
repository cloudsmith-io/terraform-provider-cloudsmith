# Team List Data Source

Retrieve all teams within a Cloudsmith organization.

## Example Usage

```hcl
provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
  slug = "my-organization"
}

data "cloudsmith_team_list" "teams" {
  organization = data.cloudsmith_organization.org.slug_perm
}
```

## Argument Reference

* `organization` - (Required) Organization within which to list teams. Provide the organization's `slug` or `slug_perm`.

## Attributes Reference

The following attribute is exported:

* `teams` - (Computed) A list of teams. Each team object provides:
  * `description` - A description of the team's purpose.
  * `name` - A descriptive name for the team.
  * `slug` - The mutable slug identifying the team in URIs.
  * `slug_perm` - The immutable slug permanently identifying the team.
  * `visibility` - Whether the team is `Visible` or `Hidden` to non-members.
