# Package Data Source

The `cloudsmith_package` data source allows you to list details and download a specific package from a given repository.

## Example Usage

```hcl
provider "cloudsmith" {
  api_key = "my-api-key"
}

resource "cloudsmith_repository" "test" {
  name      = "terraform-acc-test-package"
  namespace = "<your-namespace>"
}

data "cloudsmith_package_list" "test" {
  repository = cloudsmith_repository.test.name
  namespace  = cloudsmith_repository.test.namespace
  filters = [
				"name:dummy-package",
				"version:1.0.48",
			  ]
}

data "cloudsmith_package" "test" {
  repository  = cloudsmith_repository.test.name
  namespace   = cloudsmith_repository.test.namespace
  identifier  = data.cloudsmith_package_list.test.packages[0].slug_perm
  download    = true
  output_path = "/path/to/save/package"
}
```

## Argument Reference

* `namespace` (Required): The namespace of the package.
* `repository` (Required): The repository of the package.
* `identifier` (Required): The identifier for the package.
* `download` (Optional): If set to true, the package will be downloaded. Defaults to false. If set to false, the CDN url will be available in the `output_path`.
* `output_path` (Optional): The local file system path where the downloaded package will be stored. Defaults to OS's temp direcotry if no `output_path` is provided and `download` set to true.

