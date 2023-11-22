# Repository Privileges Data Source

The `cloudsmith_repository_privileges` data source allows you to retrieve information about repository privileges, including service accounts, teams, and users, for a specific repository.

## Example Usage

```hcl
provider "cloudsmith" {
	api_key = "my-api-key"
}

resource "cloudsmith_repository" "test" {
	name = "terraform-acc-test-privileges"
	namespace = "<your-namespace>"
}

data "cloudsmith_repository_privileges" "test_data" {
	organization = cloudsmith_repository.test.namespace
	repository = cloudsmith_repository.test.slug
}
```

## Argument Reference

* organization (Required): The organization to which the repository belongs.
* repository (Required): The repository for which privileges information is retrieved.

## Attribute Reference

The following attributes are available:

* service: A set containing privileges information for service accounts.
	* privilege: The privilege level (Admin, Write, Read).
	* slug: The unique identifier for the service account.

* team: A set containing privileges information for teams.
	* privilege: The privilege level (Admin, Write, Read).
	* slug: The unique identifier for the team.

* user: A set containing privileges information for users.
	* privilege: The privilege level (Admin, Write, Read).
	* slug: The unique identifier for the user.
