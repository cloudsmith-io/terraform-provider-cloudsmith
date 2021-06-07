# Package List Data Source

The `package_list` data source allows for retrieval of a list of packages within a given repository.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_namespace" "my_namespace" {
    slug = "my-namespace"
}

data "cloudsmith_repository" "my_repository" {
    namespace  = data.cloudsmith_namespace.my_namespace.slug_perm
    identifier = "my-repository"
}

data "cloudsmith_package_list" "my_packages" {
    namespace  = data.cloudsmith_repository.my_repository.namespace
    repository = data.cloudsmith_repository.my_repository.slug_perm

    package_group = "my-package"
    filters       = ["format:docker"]
}

output "packages" {
    value = formatlist("%s-%s", data.cloudsmith_package_list.my_packages.packages.*.name, data.cloudsmith_package_List.my_packages.*.version)
}
```

## Argument Reference

* `namespace` - (Required) Namespace to which the packages belong.
* `repository` - (Required) Repository `slug_perm` to which the packages belong.
* `filters` - (Optional) A list of Cloudsmith search filters (e.g `format:docker`, `name:^foo`).
* `most_recent` - (Optional) When `true`, only the most recent package resolved will be returned.

## Attribute Reference

All of the argument attributes are also exported as result attributes.

The following attribute is additionally exported:

* `packages` - A list of `package` entries as discovered by the data source.

