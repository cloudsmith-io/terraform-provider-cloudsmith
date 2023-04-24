# Package Download Data Source

The `package_download` data source allows you to download a specific package from a given repository.

## Example Usage

```hcl
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

data "cloudsmith_package_download" "my_package" {
  namespace         = data.cloudsmith_repository.my_repository.namespace
  repository        = data.cloudsmith_repository.my_repository.slug_perm
  package_name      = "my-package"
  package_version   = "latest"
  query             = "version:1.0"
  destination_path  = "/path/to/download"
}
```

## Argument Reference

* `namespace` - (Required) Namespace to which the package belongs.
* `repository` - (Required) Repository slug_perm to which the package belongs.
* `package_name` - (Required) Name of the package to download.
* `package_version` - (Required) Version of the package to download.
* `query` - (Optional) Specific tag for a package
* `destination_path` - (Required) Local file system path where the downloaded package will be stored.

