# SAML Authentication Resource

The SAML Authentication resource allows the configuration of SAML-based authentication for a Cloudsmith organization. This enables organizations to integrate with SAML identity providers for user authentication.

## Example Usage

```hcl
provider "cloudsmith" {
  api_key = "my-api-key"
}

resource "cloudsmith_saml_auth" "example" {
  organization        = "my-organization"
  saml_auth_enabled  = true
  saml_auth_enforced = false

  # Use either saml_metadata_url OR saml_metadata_inline
  saml_metadata_url  = "https://idp.example.com/metadata.xml"

  # Alternative: Use inline metadata
  # saml_metadata_inline = <<EOF
  # <?xml version="1.0"?>
  # <EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata">
  #   <IDPSSODescriptor>
  #     <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
  #                         Location="https://idp.example.com/sso"/>
  #   </IDPSSODescriptor>
  # </EntityDescriptor>
  # EOF
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Required) Organization slug for SAML authentication. This value cannot be changed after creation.
* `saml_auth_enabled` - (Required) Enable or disable SAML authentication for the organization.
* `saml_auth_enforced` - (Required) Whether to enforce SAML authentication for the organization.
* `saml_metadata_url` - (Optional) URL to fetch SAML metadata from the identity provider. Exactly one of `saml_metadata_url` or `saml_metadata_inline` must be specified.
* `saml_metadata_inline` - (Optional) Inline SAML metadata XML from the identity provider. Exactly one of `saml_metadata_url` or `saml_metadata_inline` must be specified.

## Import

SAML authentication configuration can be imported using the organization slug:

```shell
terraform import cloudsmith_saml_auth.example my-organization
```
