# Service Resource

The service resource allows the creation and management of services for a given Cloudsmith organization. Services allow users to create API keys that can be used for machine-to-machine or other programmatic access without requiring a real user account.

See [help.cloudsmith.io](https://help.cloudsmith.io/docs/service-accounts) for full service documentation.

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

resource "cloudsmith_service" "my_service" {
	name         = "My Service"
	organization = data.cloudsmith_organization.my_org.slug_perm

	team {
		slug = cloudsmith_team.my_team.slug
	}
}
```

## Argument Reference

The following arguments are supported:

* `description` - (Optional) A description of the service's purpose.
* `name` - (Required) A descriptive name for the service.
* `organization` - (Required) Organization to which this service belongs.
* `role` - (Optional) The service's role in the organization. If defined, must be one of `Member` or `Manager`.
* `team` - (Optional) Variable number of blocks containing team assignments for this service.
	* `role` - (Optional) The service's role in the team. If defined, must be one of `Member` or `Manager`.
	* `slug` - (Required) The team the service should be added to.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `key` - The service's API key.
* `slug` - The slug identifies the service in URIs or where a username is required.

## Import

This resource can be imported using the organization slug, and the service slug:

```shell
terraform import cloudsmith_service.my_service my-organization.my-service
```

NOTE: It's not possible to retrieve a service's API key via the Cloudsmith API after creation, so when we import a service the key is unavailable. If the API key is needed for use within Terraform (to be passed to other resources) then the resource needs to be tainted and recreated (or otherwise created fresh within Terraform).
