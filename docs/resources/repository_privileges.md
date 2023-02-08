# Respository Privileges Resource

The repository privileges resource allows the management of privileges for a given Cloudsmith repository. Using this resource it is possible to assign users, teams, or service accounts to a repository, and define the appropriate permission level for each.

Note that while users can be added to repositories in this manner, since Terraform does not (and cannot currently) manage those user accounts, you may encounter issues if the users change or are deleted outside of Terraform.

See [help.cloudsmith.io](https://help.cloudsmith.io/docs/permissions#repository-permissions) for full permissions documentation.

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
    namespace   = data.cloudsmith_organization.my_organization.slug_perm
    slug        = "my-repository"
}

resource "cloudsmith_team" "my_team" {
	organization = data.cloudsmith_organization.my_organization.slug_perm
	name         = "My Team"
}

resource "cloudsmith_team" "my_other_team" {
	organization = data.cloudsmith_organization.my_organization.slug_perm
	name         = "My Other Team"
}

resource "cloudsmith_service" "my_service" {
	name         = "My Service"
	organization = data.cloudsmith_organization.my_organization.slug_perm
}

resource "cloudsmith_repository_privileges" "privs" {
    organization = data.cloudsmith_organization.my_organization.slug
    repository   = cloudsmith_repository.my_repository.slug

	service {
		privilege = "Write"
		slug      = cloudsmith_service.my_service.slug
	}

	team {
		privilege = "Write"
		slug      = cloudsmith_team.my_team.slug
	}

	team {
		privilege = "Read"
		slug      = cloudsmith_team.my_other_team.slug
	}

    user {
        privilege = "Read"
        slug      = "some-user-slug"
    }
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Required) Organization to which this repository belongs.
* `repository` - (Required) Repository to which these privileges apply.
* `service` - (Optional) Variable number of blocks containing service accounts that should have repository privileges.
	* `privilege` - (Required) The service's privilege level in the repository. Must be one of `Admin`, `Write`, or `Read`.
	* `slug` - (Required) The slug/identifier of the service.
* `team` - (Optional) Variable number of blocks containing teams that should have repository privileges.
	* `privilege` - (Required) The team's privilege level in the repository. Must be one of `Admin`, `Write`, or `Read`.
	* `slug` - (Required) The slug/identifier of the team.
* `user` - (Optional) Variable number of blocks containing users that should have repository privileges.
	* `privilege` - (Required) The user's privilege level in the repository. Must be one of `Admin`, `Write`, or `Read`.
	* `slug` - (Required) The slug/identifier of the user.

## Import

This resource can be imported using the organization slug, and the repository slug:

```shell
terraform import cloudsmith_repository_privileges.privs my-organization.my-repository
```
