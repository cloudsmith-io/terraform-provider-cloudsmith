# SAML Group Sync Resource

The SAML resource allows the creation and management of SAML Group Sync configurations for a given Cloudsmith organization. SAML Group sync configuration allows for easy mapping of your current AD groups and assign these groups into Cloudsmith teams.

## Example Usage

```hcl

provider "cloudsmith" {
    api_key = "my-api-key"
}

resource "cloudsmith_saml" "my_saml" {
  organization = "org-name"
  idp_key = "role"
  idp_value = "example"
  role = "Member"
  team = "owners"
}
```

## Argument Reference

* `organization` - (Required) Organization (namespace) to which this SAML Group Sync configuration belongs
* `idp_key` - (Required) The attribute key from your provider
* `idp_value` - (Required) The attribute value from your provider
* `role` - (Optional) (Default to Member) The role assigned for the team (Member or Manager)
* `team` - (Required) The team associated with the configuration (The team must exist prior to creating SAML Group sync config)

## Attribute Reference

* `slug_perm` - The slug identifier

## Import

This resource can be imported using the organization slug and the SAML slug_perm:

```shell
terraform import cloudsmith_saml.my_saml my-organization.my-saml-slug-perm
```
