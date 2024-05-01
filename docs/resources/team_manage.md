# Team Manage Resource

This resource is used to manage teams in Cloudsmith. It allows you to add, update, and remove team members.

## Example Usage

```hcl
resource "cloudsmith_team_manage" "example" {
  organization = "example_org"
  team_name    = "example_team"
  members {
      role = "Manager"
      user = "user1"
    }
  members {
      role = "Member"
      user = "user2"
    }
}
```

## Argument Reference

> :warning: **Warning**: When running this resource with a **USER API** token (Not applicable to users running this with Service Account API Key) on **newly created teams**, the user will automatically get added to the team causing a `422 error` when specified in the resource as well. We highly advise using service account for this purpose to avoid the error, alternatively, the plan needs to be ran without the user in the resource and re-added after the first apply happens.

The following arguments are supported:

- `organization` - (Required) The slug of the organization.
- `team_name` - (Required) The name of the team.
- `members` - (Required) A list of members to be added to the team. Each member is a map containing `role` and `user`. The role can only be set to "Manager" or "Member".

## Attribute Reference

The following attributes are exported:

- `organization` - The slug of the organization.
- `team_name` - The name of the team.
- `members` - A list of team members. Each member is a map containing `role` and `user`.

## Import

Existing teams can be imported using the organization slug and team name, separated by a dot. For example:

```hcl
terraform import cloudsmith_team_manage.example example_org.example_team
```