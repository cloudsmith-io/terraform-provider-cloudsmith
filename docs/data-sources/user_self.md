# User Self Data Source

The `cloudsmith_user_self` data source provides information about the currently authenticated user.

## Example Usage

```hcl
data "cloudsmith_user_self" "current" {}

# Reference user attributes
output "current_user_email" {
  value = data.cloudsmith_user_self.current.email
}

output "current_user_slug" {
  value = data.cloudsmith_user_self.current.slug
}
```

## Attribute Reference

* `email` - (String) The email address associated with the authenticated user account.
* `name` - (String) The full name of the authenticated user as configured in their profile.
* `slug` - (String) The URL-friendly identifier used in URIs. This may change if the user's name is updated.
* `slug_perm` - (String) The permanent, immutable identifier that uniquely identifies the user. This value remains constant even if other user properties change.
