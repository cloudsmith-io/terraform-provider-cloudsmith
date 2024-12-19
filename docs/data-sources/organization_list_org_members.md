# Organization Member List Data Source

Get the details for all organization members.

## Example Usage

```hcl

provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

data "cloudsmith_list_org_members" "test" {
    namespace = my_organization.slug_perm
    is_active = true
}
```

## Argument Reference

* `namespace` - (Required) Namespace to which the org members belong to.
* `is_active` - (Optional) Filter for active/inactive users. Default is `true`.

All of the argument attributes are also exported as result attributes.

The following attribute is additionally exported:

* `members` - (Computed) A list of organization members. Each member has the following attributes:
  * `email` - The email address of the member.
  * `has_two_factor` - Indicates if the member has two-factor authentication enabled.
  * `is_active` - Indicates if the member is active.
  * `joined_at` - The date and time when the member joined the organization.
  * `last_login_at` - The date and time when the member last logged in.
  * `last_login_method` - The method used by the member for the last login.
  * `role` - The role of the member within the organization.
  * `user` - The username of the member.
  * `user_id` - The unique identifier of the member.
