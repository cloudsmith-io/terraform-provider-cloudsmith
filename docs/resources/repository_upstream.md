# Respository Upstream Resource

The repository upstream resource allows the management of upstreams for a Cloudsmith repository. Using this resource, it is possible to proxy and/or cache packages hosted in one or more third-party package registries through a single Cloudsmith repository.

See [docs.cloudsmith.com](https://docs.cloudsmith.com/tbc/upstream-proxying-and-caching#supported-formats) for full upstream proxying documentation.

## Example Usage

Given the following hcl...

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

resource "cloudsmith_repository" "my_repository" {
    description = "A certifiably-awesome private package repository"
    name        = "My Repository"
    namespace   = "${data.cloudsmith_organization.my_organization.slug_perm}"
    slug        = "my-repository"
}
```

...minimal configuration for various upstream types might be added as per the following examples for popular package registries.

### Cargo

```hcl
resource "cloudsmith_repository_upstream" "crates_io" {
    name          = "Crates.io"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "cargo"
    upstream_url  = "https://index.crates.io"
}
```

### Composer

```hcl
resource "cloudsmith_repository_upstream" "packagist" {
    name          = "Packagist"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "composer"
    upstream_url  = "https://packagist.org"
}
```

### Conda

```hcl
resource "cloudsmith_repository_upstream" "conda_forge" {
    name          = "Conda Forge"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "conda"
    upstream_url  = "https://conda.anaconda.org/conda-forge"
}
```

### Cran

```hcl
resource "cloudsmith_repository_upstream" "cran_registry" {
    name          = "cran_registry"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "cran"
    upstream_url  = "https://cran.r-project.org"
}
```

### Dart

```hcl
resource "cloudsmith_repository_upstream" "pub_dev" {
    name          = "pub.dev"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "dart"
    upstream_url  = "https://pub.dev"
}
```

### Deb

```hcl
resource "cloudsmith_repository_upstream" "enpass" {
    component             = "main"
    distro_versions       = ["debian/bullseye"]
    include_sources       = false
    name                  = "Enpass"
    namespace             = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository            = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_distribution = "stable"
    upstream_type         = "deb"
    upstream_url          = "https://apt.enpass.io"
}
```

### Docker

> **Note:** Dockerhub requires username and password authentication or the creation of resource will fail.

```hcl
resource "cloudsmith_repository_upstream" "docker_hub" {
    name          = "Docker Hub"
    auth_mode     = "Username and Password"
    auth_username = "my-username"
    auth_secret   = "my-password"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "docker"
    upstream_url  = "https://index.docker.io"
}
```

> **Note:** Certificate and Key authentication is only supported for Docker.

```hcl
resource "cloudsmith_repository_upstream" "other_docker_upstream" {
    name          = "Other Docker Upstream"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "docker"
    upstream_url  = "https://other.docker.io"
    auth_mode     = "Certificate and Key"
    auth_certificate = file("${path.module}/certs/client.crt")
    auth_certificate_key = file("${path.module}/certs/client.key")
}
```

### Go

```hcl
resource "cloudsmith_repository_upstream" "go_proxy" {
    name          = "Go Proxy"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "go"
    upstream_url  = "https://proxy.golang.org"
}
```

### Helm

```hcl
resource "cloudsmith_repository_upstream" "helm_charts" {
    name          = "Helm Charts"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "helm"
    upstream_url  = "https://charts.helm.sh/stable"
}
```

### HuggingFace

```hcl
resource "cloudsmith_repository_upstream" "hugging_face" {
    name          = "HuggingFace"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "huggingface"
    upstream_url  = "https://huggingface.co"
}
```

### Maven

```hcl
resource "cloudsmith_repository_upstream" "maven_central" {
    name          = "Maven Central"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "maven"
    upstream_url  = "https://repo1.maven.org/maven2"
}
```

### NPM

```hcl
resource "cloudsmith_repository_upstream" "npmjs" {
    name          = "npmjs"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "npm"
    upstream_url  = "https://registry.npmjs.org"
}
```

### NuGet

```hcl
resource "cloudsmith_repository_upstream" "nuget" {
    name          = "NuGet.org"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "nuget"
    upstream_url  = "https://api.nuget.org/v3/index.json"
}
```

### Python

```hcl
resource "cloudsmith_repository_upstream" "pypi" {
    name          = "Python Package Index"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "python"
    upstream_url  = "https://pypi.org"
}
```

### RedHat/RPM

```hcl
resource "cloudsmith_repository_upstream" "rpm_fusion" {
    distro_version  = "fedora/35"
    include_sources = true
    name            = "RPM Fusion"
    namespace       = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository      = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type   = "rpm"
    upstream_url    = "https://download1.rpmfusion.org/free/fedora/releases/35/Everything/x86_64/os"
}
```

### Ruby

```hcl
resource "cloudsmith_repository_upstream" "ruby_gems" {
    name          = "RubyGems"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "ruby"
    upstream_url  = "https://rubygems.org"
}
```

### Swift

```hcl
resource "cloudsmith_repository_upstream" "swift_registry" {
    name          = "swift_registry"
    namespace     = "${data.cloudsmith_organization.my_organization.slug_perm}"
    repository    = "${resource.cloudsmith_repository.my_repository.slug_perm}"
    upstream_type = "swift"
    upstream_url  = "https://swift.cloudsmith.io/swift/swiftpackageindex-mirror/"
}
```

## Argument Reference

The following arguments are supported:

|        Argument         | Required |     Type     |                                                       Enumeration                                                       |                                                                                                                      Description                                                                                                                      |
|:-----------------------:|:--------:|:------------:|:-----------------------------------------------------------------------------------------------------------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|
|       `auth_mode`       |    N     |    string    |                                   `"None"`<br>`"Username and Password"`<br>`"Token"`<br>`"Certificate and Key"`                                    |                                                                                              The authentication mode to use when accessing the upstream.                                                                                              |
|      `auth_secret`      |    N     |    string    |                                                           N/A                                                           |                                                   Used in conjunction with an `auth_mode` of `"Username and Password"` or `"Token"` to hold the password or token used when accessing the upstream.                                                   |
|     `auth_username`     |    N     |    string    |                                                           N/A                                                           |                                                          Used only in conjunction with an `auth_mode` of `"Username and Password"` to declare the username used when accessing the upstream.                                                          |
|    `auth_certificate`   |    N     |    string    |                                                           N/A                                                           |                                                          Used only in conjunction with an `auth_mode` of `"Certificate and Key"` to provide the PEM-encoded certificate content for mTLS authentication. Use with the `file()` function.                                                          |
|  `auth_certificate_key` |    N     |    string    |                                                           N/A                                                           |                                                          Used only in conjunction with an `auth_mode` of `"Certificate and Key"` to provide the PEM-encoded private key content for mTLS authentication. Use with the `file()` function.                                                          |
|       `component`       |    N     |    string    |                                                           N/A                                                           |                                    Used only in conjunction with an `upstream_type` of `"deb"` to declare the [component](https://wiki.debian.org/DebianRepository/Format#Components) to fetch from the upstream.                                     |
|    `distro_version`     |    N     |    string    |                                                           N/A                                                           |                                             Used only in conjunction with an `upstream_type` of `"rpm"` to declare the distribution/version that packages found on this upstream will be associated with.                                             |
|    `distro_versions`    |    N     | list<string> |                                                           N/A                                                           |                                       Used only in conjunction with an `upstream_type` of `"deb"` to declare the array of distributions/versions that packages found on this upstream will be associated with.                                        |
|    `extra_header_1`     |    N     |    string    |                                                           N/A                                                           |                                                                                                   The key for extra header #1 to send to upstream.                                                                                                    |
|    `extra_header_2`     |    N     |    string    |                                                           N/A                                                           |                                                                                                   The key for extra header #2 to send to upstream.                                                                                                    |
|     `extra_value_1`     |    N     |    string    |                                                           N/A                                                           |                                                                         The value for extra header #1 to send to upstream. This is stored as plaintext, and is NOT encrypted.                                                                         |
|     `extra_value_2`     |    N     |    string    |                                                           N/A                                                           |                                                                         The value for extra header #2 to send to upstream. This is stored as plaintext, and is NOT encrypted.                                                                         |
|    `include_sources`    |    N     |     bool     |                                                           N/A                                                           |                                                       Used only in conjunction with an `upstream_type` of `"deb"` or `"rpm"`. When true, source packages will be available from this upstream.                                                        |
|       `is_active`       |    N     |     bool     |                                                           N/A                                                           |                                                                                            Whether or not this upstream is active and ready for requests.                                                                                             |
|         `mode`          |    N     |    string    |                                 `"Proxy Only"`<br>`"Cache and Proxy"`<br>`"Cache Only"`                                 |                                            The mode that this upstream should operate in. Upstream sources can be used to proxy resolved packages, as well as operate in a proxy/cache or cache only mode.                                            |
|         `name`          |    Y     |    string    |                                                           N/A                                                           |                                                 A descriptive name for this upstream source. A shortened version of this name will be used for tagging cached packages retrieved from this upstream.                                                  |
|       `namespace`       |    Y     |    string    |                                                           N/A                                                           |                                                                                                    The Organization to which the upstream belongs.                                                                                                    |
|       `priority`        |    N     |    number    |                                                           N/A                                                           |                                                                      Upstream sources are selected for resolving requests by sequential order (1..n), followed by creation date.                                                                      |
|      `repository`       |    Y     |    string    |                                                           N/A                                                           |                                                                                                     The Repository to which the upstream belongs.                                                                                                     |
| `upstream_distribution` |    N     |    string    |                                                           N/A                                                           |                                    Used only in conjunction with an `upstream_type` of `"deb"` to declare the [distribution](https://wiki.debian.org/DebianRepository/Format#Overview) to fetch from the upstream.                                    |
|     `upstream_type`     |    Y     |    string    | `"cargo"`<br>`"composer"`<br>`"conda"`<br>`"cran"`<br>`"dart"`<br>`"deb"`<br>`"docker"`<br>`"go"`<br>`"helm"`<br>`"huggingface"`<br>`"maven"`<br>`"npm"`<br>`"nuget"`<br>`"python"`<br>`"rpm"`<br>`"ruby"`<br>`"swift"` | The type of Upstream. |
|     `upstream_url`      |    Y     |    string    |                                                           N/A                                                           |                                                    The URL for this upstream source. This must be a fully qualified URL including any path elements required to reach the root of the repository. The URL cannot end with a trailing slash.                                                     |
|      `verify_ssl`       |    N     |     bool     |                                                           N/A                                                           | If enabled, SSL certificates are verified when requests are made to this upstream. It's recommended to leave this enabled for all public sources to help mitigate Man-In-The-Middle (MITM) attacks. Please note this only applies to HTTPS upstreams. |

## Import

This resource can be imported using the organization slug, the repository slug, the upstream type and the upstream slug_perm:

```shell
terraform import cloudsmith_repository_upstream.my_upstream my-organization.my-repository.upstream-type.slug-perm
```
