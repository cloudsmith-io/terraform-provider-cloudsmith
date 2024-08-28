# Repository Resource

The repository resource allows creation and management of package repositories within a Cloudsmith namespace. Repositories store packages and are the main entities with which Cloudsmith users interact.

See [help.cloudsmith.io](https://help.cloudsmith.io/docs/manage-a-repository) for full repository documentation.

# Example Usage

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

## Argument Reference

* `contextual_auth_realm` - (Optional) If set to `true`, missing credentials for this repository where basic authentication is required shall present an enriched value in the 'WWW-Authenticate' header containing the namespace and repository. This can be useful for tooling such as SBT where the authentication realm is used to distinguish and disambiguate credentials.
* `copy_own` - (Optional) If set to `true`, users can copy any of their own packages that they have uploaded, assuming that they still have write privilege for the repository. This takes precedence over privileges configured in the 'Access Controls' section of the repository, and any inherited from the org.
* `copy_packages` - (Optional) This defines the minimum level of privilege required for a user to copy packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific copy setting. Valid values include `Admin`, `Read`, and `Write`.
* `default_privilege` - (Optional) This defines the default level of privilege that all of your organization members have for this repository(`Admin`, `Read`, `Write`,and `None`). This does not include collaborators, but applies to any member of the org regardless of their own membership role (i.e. it applies to owners, managers and members). Be careful if setting this to admin, because any member will be able to change settings.
* `delete_own` - (Optional) If set to `true`, users can delete any of their own packages that they have uploaded, assuming that they still have write privilege for the repository. This takes precedence over privileges configured in the 'Access Controls' section of the repository, and any inherited from the org.
* `delete_packages` - (Optional) This defines the minimum level of privilege required for a user to delete packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific delete setting. Valid values include `Admin` and `Write`.
* `description` - (Optional) A description of the repository's purpose/contents.
* `docker_refresh_tokens_enabled` - (Optional) If set to `true`, refresh tokens will be issued in addition to access tokens for Docker authentication. This allows unlimited extension of the lifetime of access tokens.
* `index_files` - (Optional) If set to `true`, files contained in packages will be indexed, which increase the synchronisation time required for packages. Note that it is recommended you keep this enabled unless the synchronisation time is significantly impacted.
* `move_own` - (Optional) If set to `true`, users can move any of their own packages that they have uploaded, assuming that they still have write privilege for the repository. This takes precedence over privileges configured in the 'Access Controls' section of the repository, and any inherited from the org.
* `move_packages` - (Optional) This defines the minimum level of privilege required for a user to move packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific move setting. Valid values include `Admin` and `Write`.
* `name` - (Required) A descriptive visual name for the repository.
* `namespace` - (Required) Namespace (or organization) to which this repository belongs.
* `proxy_npmjs` - (Optional) If set to `true`, Npm packages that are not in the repository when requested by clients will automatically be proxied from the public npmjs.org registry. If there is at least one version for a package, others will not be proxied.
* `proxy_pypi` - (Optional) If set to `true`, Python packages that are not in the repository when requested by clients will automatically be proxied from the public pypi.python.org registry. If there is at least one version for a package, others will not be proxied.
* `raw_package_index_enabled` - (Optional) If set to `true`, HTML and JSON indexes will be generated that list all available raw packages in the repository.
* `raw_package_index_signatures_enabled` - (Optional) If set to `true`, the HTML and JSON indexes will display raw package GPG signatures alongside the index packages.
* `replace_packages` - (Optional) This defines the minimum level of privilege required for a user to republish packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific republish setting. Please note that the user still requires the privilege to delete packages that will be replaced by the new package; otherwise the republish will fail. Valid values include `Admin` and `Write`.
* `replace_packages_by_default` - (Optional) If set to `true`, uploaded packages will overwrite/replace any others with the same attributes (e.g. same version) by default. This only applies if the user has the required privilege for the republishing AND has the required privilege to delete existing packages that they don't own.
* `repository_type` - (Optional) The repository type changes how it is accessed and billed. Private repositories can only be used on paid plans, but are visible only to you or authorised delegates. Public repositories are free to use on all plans and visible to all Cloudsmith users.
* `resync_own` - (Optional) If set to `true`, users can resync any of their own packages that they have uploaded, assuming that they still have write privilege for the repository. This takes precedence over privileges configured in the 'Access Controls' section of the repository, and any inherited from the org.
* `resync_packages` - (Optional) This defines the minimum level of privilege required for a user to resync packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific resync setting. Valid values include `Admin` and `Write`.
* `scan_own` - (Optional) If set to `true`, users can scan any of their own packages that they have uploaded, assuming that they still have write privilege for the repository. This takes precedence over privileges configured in the 'Access Controls' section of the repository, and any inherited from the org.
* `scan_packages` - (Optional) This defines the minimum level of privilege required for a user to scan packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific scan setting.
* `show_setup_all` - (Optional) If set to `true`, the Set Me Up help for all formats will always be shown, even if you don't have packages of that type uploaded. Otherwise, help will only be shown for packages that are in the repository. For example, if you have uploaded only NuGet packages, then the Set Me Up help for NuGet packages will be shown only.
* `slug` - (Optional) The slug identifies the repository in URIs.
* `storage_region` - (Optional) The Cloudsmith region in which package files are stored.
  * `default` - Default Region
  * `au-sydney` - Sydney, Australia
  * `sg-singapore` - Singapore
  * `ca-montreal` - Montreal, Canada
  * `de-frankfurt` - Frankfurt, Germany
  * `us-oregon` - Oregon, United States
  * `us-ohio` - Ohio, United States
  * `ie-dublin` - Dublin, Ireland
* `strict_npm_validation` - (Optional) If set to `true`, npm packages will be validated strictly to ensure the package matches specifcation. You can turn this off if you have packages that are old or otherwise mildly off-spec, but we can't guarantee the packages will work with npm-cli or other tooling correctly. Turn off at your own risk!
* `tag_pre_releases_as_latest` - (Default `false`) If `true`, packages pushed with a pre-release component on that version will be marked with the 'latest' tag. Note that if unchecked, a repository containing ONLY pre-release versions, will have no version marked latest which may cause incompatibility with native tools
* `use_debian_labels` - (Optional) If set to `true`, a 'Label' field will be present in Debian-based repositories. It will contain a string that identifies the entitlement token used to authenticate the repository, in the form of 'source=t-'; or 'source=none' if no token was used. You can use this to help with pinning.
* `use_entitlements_privilege` - (Optional) Possible values: `Read`, `Write`, `Admin`. This defines the minimum level of privilege required for a user to see/use entitlement tokens with private repositories. If a user does not have the permission, they will only be able to download packages using other credentials, such as email/password via basic authentication. Use this if you want to force users to only use their user-based token, which is tied to their access (if removed, they can't use it).
* `use_default_cargo_upstream` - (Optional) If set to `true`, dependencies of uploaded Cargo crates which do not set an explicit value for \"registry\" will be assumed to be available from crates.io. If unset to `true`, dependencies with unspecified \"registry\" values will be assumed to be available in the registry being uploaded to. Uncheck this if you want to ensure that dependencies are only ever installed from Cloudsmith unless explicitly specified as belong to another registry.
* `use_noarch_packages` - (Optional) If set to `true`, noarch packages (if supported) are enabled in installations/configurations. A noarch package is one that is not tied to specific system architecture (like i686).
* `use_source_packages` - (Optional) If set to `true`, source packages (if supported) are enabled in installations/configurations. A source package is one that contains source code rather than built binaries.
* `use_vulnerability_scanning` - (Optional) If set to `true`, vulnerability scanning will be enabled for all supported packages within this repository.
* `user_entitlements_enabled` - (Optional) If set to `true`, users can use and manage their own user-specific entitlement token for the repository (if private). Otherwise, user-specific entitlements are disabled for all users.
* `view_statistics` - (Optional) This defines the minimum level of privilege required for a user to view repository statistics, to include entitlement-based usage, if applicable. If a user does not have the permission, they won't be able to view any statistics, either via the UI, API or CLI. Valid values include `Admin`, `Write`, and `Read`.
* `wait_for_deletion` - (Optional) If true, terraform will wait for a repository to be permanently deleted before finishing.

## Attribute Reference

* `cdn_url` - Base URL from which packages and other artifacts are downloaded.
* `contextual_auth_realm` - If set to `true`, missing credentials for this repository where basic authentication is required shall present an enriched value in the 'WWW-Authenticate' header containing the namespace and repository. This can be useful for tooling such as SBT where the authentication realm is used to distinguish and disambiguate credentials.
* `copy_own` - If set to `true`, users can copy any of their own packages that they have uploaded, assuming that they still have write privilege for the repository. This takes precedence over privileges configured in the 'Access Controls' section of the repository, and any inherited from the org.
* `copy_packages` - This defines the minimum level of privilege required for a user to copy packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific copy setting.
* `created_at` - ISO 8601 timestamp at which the repository was created.
* `default_privilege` - This defines the default level of privilege that all of your organization members have for this repository(Admin, Read, Write, None). This does not include collaborators, but applies to any member of the org regardless of their own membership role (i.e. it applies to owners, managers and members). Be careful if setting this to admin, because any member will be able to change settings.
* `deleted_at` - ISO 8601 timestamp at which the repository was deleted.
* `delete_own` - If set to `true`, users can delete any of their own packages that they have uploaded, assuming that they still have write privilege for the repository. This takes precedence over privileges configured in the 'Access Controls' section of the repository, and any inherited from the org.
* `delete_packages` - This defines the minimum level of privilege required for a user to delete packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific delete setting.
* `description` - A description of the repository's purpose/contents.
* `docker_refresh_tokens_enabled` - If set to `true`, refresh tokens will be issued in addition to access tokens for Docker authentication. This allows unlimited extension of the lifetime of access tokens.
* `index_files` - When `true`, package indexing is enabled for this repository.
* `is_open_source` - True if this repository is open source.
* `is_private` - True if this repository is private.
* `is_public` - True if this repository is public.
* `move_own` - If set to `true`, users can move any of their own packages that they have uploaded, assuming that they still have write privilege for the repository. This takes precedence over privileges configured in the 'Access Controls' section of the repository, and any inherited from the org.
* `move_packages` - This defines the minimum level of privilege required for a user to move packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific move setting.
* `name` - A descriptive name for the repository.
* `namespace_url` - API endpoint to where data about this namespace can be retrieved.
* `proxy_npmjs` - If set to `true`, Npm packages that are not in the repository when requested by clients will automatically be proxied from the public npmjs.org registry. If there is at least one version for a package, others will not be proxied.
* `proxy_pypi` - If set to `true`, Python packages that are not in the repository when requested by clients will automatically be proxied from the public pypi.python.org registry. If there is at least one version for a package, others will not be proxied.
* `raw_package_index_enabled` - If set to `true`, HTML and JSON indexes will be generated that list all available raw packages in the repository.
* `raw_package_index_signatures_enabled` - If set to `true`, the HTML and JSON indexes will display raw package GPG signatures alongside the index packages.
* `replace_packages` - This defines the minimum level of privilege required for a user to republish packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific republish setting. Please note that the user still requires the privilege to delete packages that will be replaced by the new package; otherwise the republish will fail.
* `replace_packages_by_default` - If set to `true`, uploaded packages will overwrite/replace any others with the same attributes (e.g. same version) by default. This only applies if the user has the required privilege for the republishing AND has the required privilege to delete existing packages that they don't own.
* `repository_type` - The repository type changes how it is accessed and billed. Private repositories can only be used on paid plans, but are visible only to you or authorised delegates. Public repositories are free to use on all plans and visible to all Cloudsmith users.
* `resync_own` - If set to `true`, users can resync any of their own packages that they have uploaded, assuming that they still have write privilege for the repository. This takes precedence over privileges configured in the 'Access Controls' section of the repository, and any inherited from the org.
* `resync_packages` - This defines the minimum level of privilege required for a user to resync packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific resync setting.
* `scan_own` - If set to `true`, users can scan any of their own packages that they have uploaded, assuming that they still have write privilege for the repository. This takes precedence over privileges configured in the 'Access Controls' section of the repository, and any inherited from the org.
* `scan_packages` - This defines the minimum level of privilege required for a user to scan packages. Unless the package was uploaded by that user, in which the permission may be overridden by the user-specific scan setting.
* `self_html_url` - The Cloudsmith web URL for this repository.
* `self_url` - The Cloudsmith API endpoint for this repository.
* `show_setup_all` - If set to `true`, the Set Me Up help for all formats will always be shown, even if you don't have packages of that type uploaded. Otherwise, help will only be shown for packages that are in the repository. For example, if you have uploaded only NuGet packages, then the Set Me Up help for NuGet packages will be shown only.
* `slug` - The slug identifies the repository in URIs.
* `slug_perm` - The internal immutable identifier for this repository.
* `storage_region` - The Cloudsmith region in which package files are stored.
* `strict_npm_validation` - If set to `true`, npm packages will be validated strictly to ensure the package matches specifcation. You can turn this off if you have packages that are old or otherwise mildly off-spec, but we can't guarantee the packages will work with npm-cli or other tooling correctly. Turn off at your own risk!
* `tag_pre_releases_as_latest` - (Default `false`) If `true`, packages pushed with a pre-release component on that version will be marked with the 'latest' tag. Note that if unchecked, a repository containing ONLY pre-release versions, will have no version marked latest which may cause incompatibility with native tools
* `use_debian_labels` - If set to `true`, a 'Label' field will be present in Debian-based repositories. It will contain a string that identifies the entitlement token used to authenticate the repository, in the form of 'source=t-'; or 'source=none' if no token was used. You can use this to help with pinning.
* `use_entitlements_privilege` - This defines the minimum level of privilege required for a user to see/use entitlement tokens with private repositories. If a user does not have the permission, they will only be able to download packages using other credentials, such as email/password via basic authentication. Use this if you want to force users to only use their user-based token, which is tied to their access (if removed, they can't use it). Possible values: Read, Write, Admin.
* `use_default_cargo_upstream` - If set to `true`, dependencies of uploaded Cargo crates which do not set an explicit value for \"registry\" will be assumed to be available from crates.io. If unset to `true`, dependencies with unspecified \"registry\" values will be assumed to be available in the registry being uploaded to. Uncheck this if you want to ensure that dependencies are only ever installed from Cloudsmith unless explicitly specified as belong to another registry.
* `use_noarch_packages` - If set to `true`, noarch packages (if supported) are enabled in installations/configurations. A noarch package is one that is not tied to specific system architecture (like i686).
* `use_source_packages` - If set to `true`, source packages (if supported) are enabled in installations/configurations. A source package is one that contains source code rather than built binaries.
* `use_vulnerability_scanning` - If set to `true`, vulnerability scanning will be enabled for all supported packages within this repository.
* `user_entitlements_enabled` - If set to `true`, users can use and manage their own user-specific entitlement token for the repository (if private). Otherwise, user-specific entitlements are disabled for all users.
* `view_statistics` - This defines the minimum level of privilege required for a user to view repository statistics, to include entitlement-based usage, if applicable. If a user does not have the permission, they won't be able to view any statistics, either via the UI, API or CLI.

## Import

This resource can be imported using the organization slug, and the repository slug:

```shell
terraform import cloudsmith_repository.my_repository my-organization.my-repository
```
