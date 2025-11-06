# Team Members Data Source

Retrieve all members of a specific team within a Cloudsmith organization.

## Example Usage

```hcl
provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
  slug = "my-organization"
}

data "cloudsmith_team_members" "members" {
  organization = data.cloudsmith_organization.org.slug_perm
  team_name    = cloudsmith_team.example.name
}
```

## Argument Reference

* `organization` - (Required) The organization to which the team belongs. Use the organization's `slug` or `slug_perm`.
* `team_name` - (Required) The name (slug) of the team whose members you want to list.

## Attributes Reference

* `members` - (Computed) A list of team members. Each member has the following attributes:
  * `role` - The role assigned to the user within the team (e.g., `Member`, `Manager`).
  * `user` - The username of the member.

