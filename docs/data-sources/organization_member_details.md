# Organization Member Details Data Resource

Get the details for a specific organization member.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

data "cloudsmith_org_member_details" "my_org_member_details" {
  organization = data.cloudsmith_organization.my_organization.slug
  member = "username-or-email"
}
```

## Argument Reference

* `organization` - (Required) Organization to which the org member belongs to.
* `member` - (Required) The username, slug or email of the member.

All of the argument attributes are also exported as result attributes.

The following attribute is additionally exported:

* `email` - The email address of the organization member (string).
* `has_two_factor` - Indicates whether the member has two-factor authentication enabled (boolean).
* `is_active` - Indicates whether the member is active (boolean).
* `joined_at` - The date and time when the member joined the organization (date-time).
* `last_login_at` - The date and time of the member's last login (date-time | null).
* `last_login_method` - The method used for the member's last login. Defaults to "Unknown" and can be one of the following: "Unknown", "Password", "Social", "SAML" (string).
* `role` - The role of the member within the organization. Defaults to "Owner" and can be one of the following: "Owner", "Manager", "Member", "Collaborator" (string).
* `user` - Information about the user associated with the member.
* `user_id` - The ID of the user (string).
* `user_name` - The username of the user (string).
* `user_url` - The URL of the user's profile (uri).
* `visibility` - The visibility of the member's profile. Defaults to "Public" and can be one of the following: "Public", "Private" (string).