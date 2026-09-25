# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

provider "cloudsmith" {
  api_key = "my-api-key"
}

resource "cloudsmith_saml_auth" "example" {
  organization       = "my-organization"
  saml_auth_enabled  = true
  saml_auth_enforced = false

  # Use either saml_metadata_url OR saml_metadata_inline
  saml_metadata_url = "https://idp.example.com/metadata.xml"

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
