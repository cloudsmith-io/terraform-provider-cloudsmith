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
  download_dir = "/path/to/your/directory"
}
```

## Argument Reference

-   `namespace` (Required): The namespace of the package.
-   `repository` (Required): The repository of the package.
-   `identifier` (Required): The identifier for the package.
-   `download` (Optional): If set to true, the package will be downloaded. Defaults to false. If set to false, the CDN URL will be available in the `output_path`.
-   `download_dir` (Optional): The directory where the file will be downloaded to. If not set and `download` is set to `true`, it will default to the operating system's default temporary directory and save the file there.

## Attribute Reference

-   `cdn_url`: The URL of the package to download. This attribute is computed and available only when the `download` argument is set to `false`.
-   `checksum_md5`: MD5 hash of the downloaded package. If `download` is set to `false`, the checksum is returned from the package API instead.
-   `checksum_sha1`: SHA1 hash of the downloaded package.If `download` is set to `false`, the checksum is returned from the package API instead.
-   `checksum_sha256`: SHA256 hash of the downloaded package.If `download` is set to `false`, the checksum is returned from the package API instead.
-   `checksum_sha512`: SHA512 hash of the downloaded package.If `download` is set to `false`, the checksum is returned from the package API instead.
-   `format`: The format of the package.
-   `is_sync_awaiting`: Indicates whether the package is awaiting synchronization.
-   `is_sync_completed`: Indicates whether the package synchronization has completed.
-   `is_sync_failed`: Indicates whether the package synchronization has failed.
-   `is_sync_in_flight`: Indicates whether the package synchronization is currently in-flight.
-   `is_sync_in_progress`: Indicates whether the package synchronization is currently in-progress.
-   `name`: The name of the package.
-   `output_path`: The location of the package. If the `download` argument is set to `true`, this will provide the path where the package is downloaded.
-   `output_directory`: The directory where the package is downloaded.
-   `slug`: The public unique identifier for the package.
-   `slug_perm`: The slug_perm that immutably identifies the package.
-   `version`: The version of the package.