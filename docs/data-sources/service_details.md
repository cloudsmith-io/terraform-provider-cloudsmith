# Service Details Data Source

Retrieve detailed information about a single service account in a Cloudsmith organization.

## Example Usage

```hcl
provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_organization" "org" {
  slug = "my-organization"
}

data "cloudsmith_service_details" "service" {
  organization = data.cloudsmith_organization.org.slug_perm
  service      = cloudsmith_service.example.slug
}
```

## Argument Reference

* `organization` - (Required) The organization to which the service belongs. Provide `slug` or `slug_perm`.
* `service` - (Required) The slug of the service account to retrieve.

## Attributes Reference

All arguments are also exported as attributes. In addition, the following are exported:

* `created_at` - Timestamp (RFC3339) when the service was created.
* `created_by` - The user who created the service.
* `created_by_url` - API URL for the creating user.
* `description` - A description of the service's purpose.
* `key` - The API key for the service (may be redacted unless freshly created). Sensitive.
* `key_expires_at` - When the API key will expire (blank if no policy applies).
* `name` - The service's descriptive name.
* `role` - The service's role within the organization.
* `slug` - The slug identifying the service in URIs.
* `teams` - (Computed) A list of teams the service is assigned to. Each team object:
  * `role` - The service's role in that team context.
  * `slug` - The team slug.
