# Service List Data Source

Retrieve all service accounts within a Cloudsmith organization.

## Example Usage

```hcl
provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
  slug = "my-organization"
}

data "cloudsmith_service_list" "services" {
  organization = data.cloudsmith_organization.org.slug_perm
}
```

## Argument Reference

* `organization` - (Required) Organization in which to list service accounts. Provide the organization's `slug` or `slug_perm`.
* `query` - (Optional) Search query to filter services. Supported fields: `name`, `role`. Examples: `name:deploy-bot`, `role:Member`.
* `sort` - (Optional) Field to sort results. Prefix with `-` for descending. Supported fields: `created_at`, `name`, `role`. Defaults to `created_at`.

## Attributes Reference

* `services` - (Computed) A list of service accounts. Each service object includes:
  * `created_at` - Timestamp (RFC3339) when the service was created.
  * `created_by` - The user who created the service.
  * `created_by_url` - API URL for the creating user.
  * `description` - A description of the service's purpose.
  * `key` - The API key for the service (may be redacted unless freshly created). Sensitive.
  * `key_expires_at` - When the API key will expire (blank if no policy applies).
  * `name` - A descriptive name for the service.
  * `role` - The service's role in the organization (e.g., `Member`, `Manager`).
  * `slug` - The slug identifying the service in URIs.
  * `teams` - List of team assignments for the service. Each team has:
    * `role` - Role of the service within the team context.
    * `slug` - Team slug.

