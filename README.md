Terraform Provider for Cloudsmith
=================================

![](https://cloudsmith.com/images/uploads/resources/cloudsmith-logo-master-color.svg)

Terraform provider for managing your Cloudsmith resources.

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.12.x
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

To use a released provider in your Terraform environment, run [`terraform init`](https://www.terraform.io/docs/commands/init.html) and Terraform will automatically install the provider. To specify a particular provider version when installing released providers, see the [Terraform documentation on provider versioning](https://www.terraform.io/docs/configuration/providers.html#version-provider-versions).

To instead use a custom-built provider in your Terraform environment (e.g. the provider binary from the build instructions above), follow the instructions to [install it as a plugin.](https://www.terraform.io/docs/plugins/basics.html#installing-plugins) After placing the custom-built provider into your plugins directory, run `terraform init` to initialize it.

### Examples

Create a repository with a custom entitlement token

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
    namespace   = data.cloudsmith_namespace.my_namespace.slug_perm
    slug        = "my-repository"
}

resource "cloudsmith_entitlement" "my_entitlement" {
    name       = "Test Entitlement"
    namespace  = cloudsmith_repository.test.namespace
    repository = cloudsmith_repository.test.slug_perm
}
```


Retrieve a list of packages from a repository

```
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_namespace" "my_namespace" {
  slug = "my-namespace"
}

data "cloudsmith_repository" "my_repository" {
  namespace  = data.cloudsmith_namespace.my_namespace.slug
  identifier = "my-repository"
}

data "cloudsmith_package_list" "my_packages" {
  namespace     = data.cloudsmith_repository.my_repository.namespace
  repository    = data.cloudsmith_repository.my_repository.slug_perm
  filters       = ["format:docker", "name:^my-package"]
}

output "packages" {
  value = formatlist("%s-%s", data.cloudsmith_packages.my_packages.packages.*.name, data.cloudsmith_packages.my_packages.packages.*.version)
}
```

Testing the Provider
-----------------------

In order to test the provider, you can run `go test`.

```sh
$ go test -v ./...
```

In order to run the full suite of Acceptance tests, you'll need a paid Cloudsmith account.

You'll also need to set a few environment variables:

- `TF_ACC=1`: Used to enable acceptance tests during `go test`.
- `CLOUDSMITH_API_KEY`: API key used to manage resources during test runs.
- `CLOUDSMITH_NAMESPACE`: Cloudsmith namespace in which to create and destroy resources under test.

*Note:* Acceptance tests create real resources, and may cost money to run.

```sh
$ export TF_ACC=1
$ export CLOUDSMITH_API_KEY=mykey
$ export CLOUDSMITH_NAMESPACE=mynamespace
$ go test -v ./...
```

If needed, you can also run individual tests with the `-run` flag:

```sh
$ go test -v -run=TestAccEntitlement_basic ./...
```
