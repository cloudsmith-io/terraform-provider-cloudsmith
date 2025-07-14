# Cloudsmith Provider

This provider allows Cloudsmith users to automate the provisioning of resources using Terraform. Users can create and manage repositories, along with entitlement tokens to grant access to repository contents.

See [docs.cloudsmith.com](https://docs.cloudsmith.com/) for full documentation (including an API reference).

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_namespace" "my_namespace" {
    slug = "my-namespace"
}

resource "cloudsmith_repository" "my_repository" {
    description = "A certifiably-awesome private package repository"
    name        = "My Repository"
    namespace   = "${data.cloudsmith_namespace.my_namespace.slug_perm}"
    slug        = "my-repository"
}

resource "cloudsmith_entitlement" "my_entitlement" {
    name       = "Test Entitlement"
    namespace  = "${cloudsmith_repository.test.namespace}"
    repository = "${cloudsmith_repository.test.slug_perm}"
}
```

## Argument Reference

* `api_key` - (Required) The API key for authenticating with the Cloudsmith API.
* `api_host` - (Optional) The API host to connect to (used to connect to a non-production Cloudsmith instance, mostly useful for testing).
