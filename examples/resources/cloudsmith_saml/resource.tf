resource "cloudsmith_saml" "my_saml" {
  organization = "my-organization"
  idp_key      = "role"
  idp_value    = "example"
  role         = "Member"
  enabled      = true
  team         = "owners"
}
