Terraform Provider for Cloudsmith
=================================

![](https://cloudsmith.com/images/uploads/resources/cloudsmith-logo-master-color.svg)

Terraform provider for managing your Cloudsmith resources.

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.11.x
-	[Go](https://golang.org/doc/install) >= 1.13 (to build the provider plugin)

Building The Provider
---------------------

Clone repository:

```sh
$ git clone git@github.com:cloudsmith-io/terraform-provider-cloudsmith
```

Enter the provider directory and build the provider:

```sh
$ cd terraform-provider-cloudsmith
$ go build
```

Using the provider
------------------

Full docs: TODO

Example: Create a repository with a non-default entitlement

```
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
    name                = "Test Entitlement"
    namespace           = "${cloudsmith_repository.test.namespace}"
    repository          = "${cloudsmith_repository.test.slug_perm}"
}
```

Developing the Provider
-----------------------

TODO
