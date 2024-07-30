package cloudsmith

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func importRepository(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<repository_slug>, got: %s", d.Id(),
		)
	}

	d.Set("namespace", idParts[0])
	d.SetId(idParts[1])
	return []*schema.ResourceData{d}, nil
}

func resourceRepositoryCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")

	req := pc.APIClient.ReposApi.ReposCreate(pc.Auth, namespace)
	req = req.Data(cloudsmith.RepositoryCreateRequest{
		ContextualAuthRealm:              optionalBool(d, "contextual_auth_realm"),
		CopyOwn:                          optionalBool(d, "copy_own"),
		CopyPackages:                     optionalString(d, "copy_packages"),
		DefaultPrivilege:                 optionalString(d, "default_privilege"),
		DeleteOwn:                        optionalBool(d, "delete_own"),
		DeletePackages:                   optionalString(d, "delete_packages"),
		Description:                      optionalString(d, "description"),
		DockerRefreshTokensEnabled:       optionalBool(d, "docker_refresh_tokens_enabled"),
		IndexFiles:                       optionalBool(d, "index_files"),
		MoveOwn:                          optionalBool(d, "move_own"),
		MovePackages:                     optionalString(d, "move_packages"),
		Name:                             requiredString(d, "name"),
		ProxyNpmjs:                       optionalBool(d, "proxy_npmjs"),
		ProxyPypi:                        optionalBool(d, "proxy_pypi"),
		RawPackageIndexEnabled:           optionalBool(d, "raw_package_index_enabled"),
		RawPackageIndexSignaturesEnabled: optionalBool(d, "raw_package_index_signatures_enabled"),
		ReplacePackages:                  optionalString(d, "replace_packages"),
		ReplacePackagesByDefault:         optionalBool(d, "replace_packages_by_default"),
		RepositoryTypeStr:                optionalString(d, "repository_type"),
		ResyncOwn:                        optionalBool(d, "resync_own"),
		ResyncPackages:                   optionalString(d, "resync_packages"),
		ScanOwn:                          optionalBool(d, "scan_own"),
		ScanPackages:                     optionalString(d, "scan_packages"),
		ShowSetupAll:                     optionalBool(d, "show_setup_all"),
		Slug:                             optionalString(d, "slug"),
		StorageRegion:                    optionalString(d, "storage_region"),
		StrictNpmValidation:              optionalBool(d, "strict_npm_validation"),
		UseDebianLabels:                  optionalBool(d, "use_debian_labels"),
		UseDefaultCargoUpstream:          optionalBool(d, "use_default_cargo_upstream"),
		UseNoarchPackages:                optionalBool(d, "use_noarch_packages"),
		UseSourcePackages:                optionalBool(d, "use_source_packages"),
		UseVulnerabilityScanning:         optionalBool(d, "use_vulnerability_scanning"),
		UserEntitlementsEnabled:          optionalBool(d, "user_entitlements_enabled"),
		ViewStatistics:                   optionalString(d, "view_statistics"),
	})

	repository, _, err := pc.APIClient.ReposApi.ReposCreateExecute(req)
	if err != nil {
		return fmt.Errorf("error creating repository: %w", err)
	}

	d.SetId(repository.GetSlugPerm())

	checkerFunc := func() error {
		req := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, d.Id())
		if _, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req); err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return fmt.Errorf("error reading repository: %w", err)
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for repository (%s) to be created: %w", d.Id(), err)
	}

	return resourceRepositoryRead(d, m)
}

func resourceRepositoryRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")

	req := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, d.Id())
	repository, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error reading repository: %w", err)
	}

	d.Set("cdn_url", repository.GetCdnUrl())
	d.Set("contextual_auth_realm", repository.GetContextualAuthRealm())
	d.Set("copy_own", repository.GetCopyOwn())
	d.Set("copy_packages", repository.GetCopyPackages())
	d.Set("created_at", timeToString(repository.GetCreatedAt()))
	d.Set("default_privilege", repository.GetDefaultPrivilege())
	d.Set("delete_own", repository.GetDeleteOwn())
	d.Set("delete_packages", repository.GetDeletePackages())
	d.Set("docker_refresh_tokens_enabled", repository.GetDockerRefreshTokensEnabled())
	d.Set("deleted_at", timeToString(repository.GetDeletedAt()))
	d.Set("description", repository.GetDescription())
	d.Set("index_files", repository.GetIndexFiles())
	d.Set("is_open_source", repository.GetIsOpenSource())
	d.Set("is_private", repository.GetIsPrivate())
	d.Set("is_public", repository.GetIsPublic())
	d.Set("move_own", repository.GetMoveOwn())
	d.Set("move_packages", repository.GetMovePackages())
	d.Set("name", repository.GetName())
	d.Set("namespace_url", repository.GetNamespaceUrl())
	d.Set("proxy_npmjs", repository.GetProxyNpmjs())
	d.Set("proxy_pypi", repository.GetProxyPypi())
	d.Set("raw_package_index_enabled", repository.GetRawPackageIndexEnabled())
	d.Set("raw_package_index_signatures_enabled", repository.GetRawPackageIndexSignaturesEnabled())
	d.Set("replace_packages", repository.GetReplacePackages())
	d.Set("replace_packages_by_default", repository.GetReplacePackagesByDefault())
	d.Set("repository_type", repository.GetRepositoryTypeStr())
	d.Set("resync_own", repository.GetResyncOwn())
	d.Set("resync_packages", repository.GetResyncPackages())
	d.Set("scan_own", repository.GetScanOwn())
	d.Set("scan_packages", repository.GetScanPackages())
	d.Set("self_html_url", repository.GetSelfHtmlUrl())
	d.Set("self_url", repository.GetSelfUrl())
	d.Set("show_setup_all", repository.GetShowSetupAll())
	d.Set("slug", repository.GetSlug())
	d.Set("slug_perm", repository.GetSlugPerm())
	d.Set("storage_region", repository.GetStorageRegion())
	d.Set("strict_npm_validation", repository.GetStrictNpmValidation())
	d.Set("use_debian_labels", repository.GetUseDebianLabels())
	d.Set("use_default_cargo_upstream", repository.GetUseDefaultCargoUpstream())
	d.Set("use_noarch_packages", repository.GetUseNoarchPackages())
	d.Set("use_source_packages", repository.GetUseSourcePackages())
	d.Set("use_vulnerability_scanning", repository.GetUseVulnerabilityScanning())
	d.Set("user_entitlements_enabled", repository.GetUserEntitlementsEnabled())
	d.Set("view_statistics", repository.GetViewStatistics())

	// namespace returned from the API is always the user-facing slug, but the
	// resource may have been created in terraform with the slug_perm instead,
	// so we don't want to overwrite it with the value from the API ever,
	// instead we rely on ForceNew to ensure if it changes a new resource is
	// created.
	d.Set("namespace", namespace)

	// since we allow import using either the slug or the slug_perm, we want to
	// normalize the ID and always use the slug_perm when we can. This reset
	// allows us to set the ID unconditionally, regardless of what the user
	// passed.
	d.SetId(repository.GetSlugPerm())

	return nil
}

func resourceRepositoryUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")

	req := pc.APIClient.ReposApi.ReposPartialUpdate(pc.Auth, namespace, d.Id())
	req = req.Data(cloudsmith.RepositoryRequestPatch{
		ContextualAuthRealm:              optionalBool(d, "contextual_auth_realm"),
		CopyOwn:                          optionalBool(d, "copy_own"),
		CopyPackages:                     optionalString(d, "copy_packages"),
		DefaultPrivilege:                 optionalString(d, "default_privilege"),
		DeleteOwn:                        optionalBool(d, "delete_own"),
		DeletePackages:                   optionalString(d, "delete_packages"),
		Description:                      optionalString(d, "description"),
		DockerRefreshTokensEnabled:       optionalBool(d, "docker_refresh_tokens_enabled"),
		IndexFiles:                       optionalBool(d, "index_files"),
		MoveOwn:                          optionalBool(d, "move_own"),
		MovePackages:                     optionalString(d, "move_packages"),
		Name:                             optionalString(d, "name"),
		ProxyNpmjs:                       optionalBool(d, "proxy_npmjs"),
		ProxyPypi:                        optionalBool(d, "proxy_pypi"),
		RawPackageIndexEnabled:           optionalBool(d, "raw_package_index_enabled"),
		RawPackageIndexSignaturesEnabled: optionalBool(d, "raw_package_index_signatures_enabled"),
		ReplacePackages:                  optionalString(d, "replace_packages"),
		ReplacePackagesByDefault:         optionalBool(d, "replace_packages_by_default"),
		ResyncOwn:                        optionalBool(d, "resync_own"),
		ResyncPackages:                   optionalString(d, "resync_packages"),
		ScanOwn:                          optionalBool(d, "scan_own"),
		ScanPackages:                     optionalString(d, "scan_packages"),
		ShowSetupAll:                     optionalBool(d, "show_setup_all"),
		RepositoryTypeStr:                optionalString(d, "repository_type"),
		Slug:                             optionalString(d, "slug"),
		StrictNpmValidation:              optionalBool(d, "strict_npm_validation"),
		UseDebianLabels:                  optionalBool(d, "use_debian_labels"),
		UseDefaultCargoUpstream:          optionalBool(d, "use_default_cargo_upstream"),
		UseNoarchPackages:                optionalBool(d, "use_noarch_packages"),
		UseSourcePackages:                optionalBool(d, "use_source_packages"),
		UseVulnerabilityScanning:         optionalBool(d, "use_vulnerability_scanning"),
		UserEntitlementsEnabled:          optionalBool(d, "user_entitlements_enabled"),
		ViewStatistics:                   optionalString(d, "view_statistics"),
	})
	repository, _, err := pc.APIClient.ReposApi.ReposPartialUpdateExecute(req)
	if err != nil {
		return fmt.Errorf("error updating repository: %w", err)
	}

	d.SetId(repository.GetSlugPerm())

	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for a
		// repository being updated (changes incoming on the API side)
		time.Sleep(time.Second * 5)
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for repository (%s) to be updated: %w", d.Id(), err)
	}

	return resourceRepositoryRead(d, m)
}

func resourceRepositoryDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")

	req := pc.APIClient.ReposApi.ReposDelete(pc.Auth, namespace, d.Id())
	_, err := pc.APIClient.ReposApi.ReposDeleteExecute(req)
	if err != nil {
		return fmt.Errorf("error deleting repository: %w", err)
	}

	if requiredBool(d, "wait_for_deletion") {
		checkerFunc := func() error {
			req := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, d.Id())
			if _, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req); err != nil {
				if is404(resp) {
					return nil
				}
				return fmt.Errorf("error reading repository: %w", err)
			}
			return errKeepWaiting
		}
		if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
			return fmt.Errorf("error waiting for repository (%s) to be deleted: %w", d.Id(), err)
		}
	}

	return nil
}

func validateNoSpaces(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if strings.Contains(v, " ") {
		errs = append(errs, fmt.Errorf("%q must not contain spaces: %s", key, v))
	}
	return
}

//nolint:funlen
func resourceRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryCreate,
		Read:   resourceRepositoryRead,
		Update: resourceRepositoryUpdate,
		Delete: resourceRepositoryDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importRepository,
		},
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			if d.HasChange("storage_region") && d.Id() != "" {
				d.SetNewComputed("storage_region")
				return fmt.Errorf("warning: updating the 'storage_region' on an existing repository is currently unsupported via terraform, please update the region manually via the UI")
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			"cdn_url": {
				Type:        schema.TypeString,
				Description: "Base URL from which packages and other artifacts are downloaded.",
				Computed:    true,
			},
			"contextual_auth_realm": {
				Type: schema.TypeBool,
				Description: "If checked, missing credentials for this repository where basic authentication " +
					"is required shall present an enriched value in the 'WWW-Authenticate' header containing " +
					"the namespace and repository. This can be useful for tooling such as SBT where the " +
					"authentication realm is used to distinguish and disambiguate credentials.",
				Optional: true,
				Computed: true,
			},
			"copy_own": {
				Type: schema.TypeBool,
				Description: "If checked, users can copy any of their own packages that they have uploaded, " +
					"assuming that they still have write privilege for the repository. This takes precedence " +
					"over privileges configured in the 'Access Controls' section of the repository, and any " +
					"inherited from the org.",
				Optional: true,
				Computed: true,
			},
			"copy_packages": {
				Type: schema.TypeString,
				Description: "This defines the minimum level of privilege required for a user to copy packages. " +
					"Unless the package was uploaded by that user, in which the permission may be overridden by " +
					"the user-specific copy setting.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"Admin", "Read", "Write"}, false),
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "ISO 8601 timestamp at which the repository was created.",
				Computed:    true,
			},
			"default_privilege": {
				Type: schema.TypeString,
				Description: "This defines the default level of privilege that all of your organization members " +
					"have for this repository. This does not include collaborators, but applies to any member of the " +
					"org regardless of their own membership role (i.e. it applies to owners, managers and members). " +
					"Be careful if setting this to admin, because any member will be able to change settings." +
					"Valid values include: `Admin`, `Read`, `Write`, `None`.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"Admin", "Read", "Write", "None"}, false),
			},
			"delete_own": {
				Type: schema.TypeBool,
				Description: "If checked, users can delete any of their own packages that they have uploaded, " +
					"assuming that they still have write privilege for the repository. This takes precedence over " +
					"privileges configured in the 'Access Controls' section of the repository, and any inherited " +
					"from the org.",
				Optional: true,
				Computed: true,
			},
			"delete_packages": {
				Type: schema.TypeString,
				Description: "This defines the minimum level of privilege required for a user to delete packages. " +
					"Unless the package was uploaded by that user, in which the permission may be overridden by the " +
					"user-specific delete setting.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"Admin", "Write"}, false),
			},
			"deleted_at": {
				Type: schema.TypeString,
				Description: "ISO 8601 timestamp at which the repository was deleted " +
					"(repositories are soft deleted temporarily to allow cancelling).",
				Computed: true,
			},
			"description": {
				Type:         schema.TypeString,
				Description:  "A description of the repository's purpose/contents.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"docker_refresh_tokens_enabled": {
				Type: schema.TypeBool,
				Description: "If checked, refresh tokens will be issued in addition to access tokens for Docker " +
					"authentication. This allows unlimited extension of the lifetime of access tokens.",
				Optional: true,
				Computed: true,
			},
			"index_files": {
				Type: schema.TypeBool,
				Description: "If checked, files contained in packages will be indexed, which increase the " +
					"synchronisation time required for packages. Note that it is recommended you keep this " +
					"enabled unless the synchronisation time is significantly impacted.",
				Optional: true,
				Computed: true,
			},
			"is_open_source": {
				Type:        schema.TypeBool,
				Description: "True if this repository is open source.",
				Computed:    true,
			},
			"is_private": {
				Type:        schema.TypeBool,
				Description: "True if this repository is private.",
				Computed:    true,
			},
			"is_public": {
				Type:        schema.TypeBool,
				Description: "True if this repository is public.",
				Computed:    true,
			},
			"move_own": {
				Type: schema.TypeBool,
				Description: "If checked, users can move any of their own packages that they have uploaded, assuming " +
					"that they still have write privilege for the repository. This takes precedence over privileges " +
					"configured in the 'Access Controls' section of the repository, and any inherited from the org.",
				Optional: true,
				Computed: true,
			},
			"move_packages": {
				Type: schema.TypeString,
				Description: "This defines the minimum level of privilege required for a user to move packages. Unless " +
					"the package was uploaded by that user, in which the permission may be overridden by the " +
					"user-specific move setting.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"Admin", "Write"}, false),
			},
			"name": {
				Type:         schema.TypeString,
				Description:  "A descriptive name for the repository.",
				Required:     true,
				ValidateFunc: validation.All(validation.StringIsNotEmpty, validateNoSpaces),
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "Namespace to which this repository belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"namespace_url": {
				Type:        schema.TypeString,
				Description: "API endpoint where data about this namespace can be retrieved.",
				Computed:    true,
			},
			"proxy_npmjs": {
				Type: schema.TypeBool,
				Description: "If checked, Npm packages that are not in the repository when requested by clients will " +
					"automatically be proxied from the public npmjs.org registry. If there is at least one version for " +
					"a package, others will not be proxied.",
				Optional: true,
				Computed: true,
			},
			"proxy_pypi": {
				Type: schema.TypeBool,
				Description: "If checked, Python packages that are not in the repository when requested by clients will " +
					"automatically be proxied from the public pypi.python.org registry. If there is at least one version " +
					"for a package, others will not be proxied.",
				Optional: true,
				Computed: true,
			},
			"raw_package_index_enabled": {
				Type: schema.TypeBool,
				Description: "If checked, HTML and JSON indexes will be generated that list all available raw packages in " +
					"the repository.",
				Optional: true,
				Computed: true,
			},
			"raw_package_index_signatures_enabled": {
				Type: schema.TypeBool,
				Description: "If checked, the HTML and JSON indexes will display raw package GPG signatures alongside the " +
					"index packages.",
				Optional: true,
				Computed: true,
			},
			"replace_packages": {
				Type: schema.TypeString,
				Description: "This defines the minimum level of privilege required for a user to republish packages. " +
					"Unless the package was uploaded by that user, in which the permission may be overridden by the " +
					"user-specific republish setting. Please note that the user still requires the privilege to delete " +
					"packages that will be replaced by the new package; otherwise the republish will fail.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"Admin", "Write"}, false),
			},
			"replace_packages_by_default": {
				Type: schema.TypeBool,
				Description: "If checked, uploaded packages will overwrite/replace any others with the same attributes " +
					"(e.g. same version) by default. This only applies if the user has the required privilege for the " +
					"republishing AND has the required privilege to delete existing packages that they don't own.",
				Optional: true,
				Computed: true,
			},
			"repository_type": {
				Type: schema.TypeString,
				Description: "The repository type changes how it is accessed and billed. Private repositories " +
					"can only be used on paid plans, but are visible only to you or authorised delegates. Public " +
					"repositories are free to use on all plans and visible to all Cloudsmith users." +
					"Valid values include: `Private` or `Public`.",
				Optional:     true,
				Default:      "Private",
				ValidateFunc: validation.StringInSlice([]string{"Private", "Public"}, false),
			},
			"resync_own": {
				Type: schema.TypeBool,
				Description: "If checked, users can resync any of their own packages that they have uploaded, assuming " +
					"that they still have write privilege for the repository. This takes precedence over privileges " +
					"configured in the 'Access Controls' section of the repository, and any inherited from the org.",
				Optional: true,
				Computed: true,
			},
			"resync_packages": {
				Type: schema.TypeString,
				Description: "This defines the minimum level of privilege required for a user to resync packages. Unless " +
					"the package was uploaded by that user, in which the permission may be overridden by the user-specific " +
					"resync setting.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"Admin", "Write"}, false),
			},
			"scan_own": {
				Type: schema.TypeBool,
				Description: "If checked, users can scan any of their own packages that they have uploaded, assuming that " +
					"they still have write privilege for the repository. This takes precedence over privileges configured " +
					"in the 'Access Controls' section of the repository, and any inherited from the org.",
				Optional: true,
				Computed: true,
			},
			"scan_packages": {
				Type: schema.TypeString,
				Description: "This defines the minimum level of privilege required for a user to scan packages. Unless the " +
					"package was uploaded by that user, in which the permission may be overridden by the user-specific " +
					"scan setting.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"Admin", "Write", "Read"}, false),
			},
			"self_html_url": {
				Type:        schema.TypeString,
				Description: "Website URL for this repository.",
				Computed:    true,
			},
			"self_url": {
				Type:        schema.TypeString,
				Description: "API endpoint where data about this repository can be retrieved.",
				Computed:    true,
			},
			"show_setup_all": {
				Type: schema.TypeBool,
				Description: "If checked, the Set Me Up help for all formats will always be shown, even if you don't have " +
					"packages of that type uploaded. Otherwise, help will only be shown for packages that are in the " +
					"repository. For example, if you have uploaded only NuGet packages, then the Set Me Up help for NuGet " +
					"packages will be shown only.",
				Optional: true,
				Computed: true,
			},
			"slug": {
				Type:         schema.TypeString,
				Description:  "The slug identifies the repository in URIs.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.All(validation.StringIsNotEmpty, validateNoSpaces),
			},
			"slug_perm": {
				Type: schema.TypeString,
				Description: "The slug_perm immutably identifies the repository. " +
					"It will never change once a repository has been created.",
				Computed: true,
			},
			"storage_region": {
				Type: schema.TypeString,
				Description: "The Cloudsmith region in which package files are stored." +
					"Supported regions include: Sydney, Australia (au-sydney)," +
					"Singapore (sg-singapore), Montreal, Canada (ca-montreal), Frankfurt, Germany (de-frankfurt), Oregon," +
					"United States (us-oregon), Ohio, United States (us-ohio), Dublin, Ireland (ie-dublin)",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"au-sydney", "sg-singapore", "ca-montreal", "de-frankfurt", "us-oregon", "us-ohio", "ie-dublin"}, false),
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress diff if the resource is already created
					return d.Id() != ""
				},
			},
			"strict_npm_validation": {
				Type: schema.TypeBool,
				Description: "If checked, npm packages will be validated strictly to ensure the package matches " +
					"specifcation. You can turn this off if you have packages that are old or otherwise mildly off-spec, " +
					"but we can't guarantee the packages will work with npm-cli or other tooling correctly. Turn off at " +
					"your own risk!",
				Optional: true,
				Computed: true,
			},
			"use_debian_labels": {
				Type: schema.TypeBool,
				Description: "If checked, a 'Label' field will be present in Debian-based repositories. It will contain a " +
					"string that identifies the entitlement token used to authenticate the repository, in the form of " +
					"'source=t-'; or 'source=none' if no token was used. You can use this to help with pinning.",
				Optional: true,
				Computed: true,
			},
			"use_default_cargo_upstream": {
				Type: schema.TypeBool,
				Description: "If checked, dependencies of uploaded Cargo crates which do not set an explicit value for " +
					"\"registry\" will be assumed to be available from crates.io. If unchecked, dependencies with " +
					"unspecified \"registry\" values will be assumed to be available in the registry being uploaded to. " +
					"Uncheck this if you want to ensure that dependencies are only ever installed from Cloudsmith unless " +
					"explicitly specified as belong to another registry.",
				Optional: true,
				Computed: true,
			},
			"use_noarch_packages": {
				Type: schema.TypeBool,
				Description: "If checked, noarch packages (if supported) are enabled in installations/configurations. A " +
					"noarch package is one that is not tied to specific system architecture (like i686).",
				Optional: true,
				Computed: true,
			},
			"use_source_packages": {
				Type: schema.TypeBool,
				Description: "If checked, source packages (if supported) are enabled in installations/configurations. A " +
					"source package is one that contains source code rather than built binaries.",
				Optional: true,
				Computed: true,
			},
			"use_vulnerability_scanning": {
				Type: schema.TypeBool,
				Description: "If checked, vulnerability scanning will be enabled for all supported packages within " +
					"this repository.",
				Optional: true,
				Computed: true,
			},
			"user_entitlements_enabled": {
				Type: schema.TypeBool,
				Description: "If checked, users can use and manage their own user-specific entitlement token for the " +
					"repository (if private). Otherwise, user-specific entitlements are disabled for all users.",
				Optional: true,
				Computed: true,
			},
			"view_statistics": {
				Type: schema.TypeString,
				Description: "This defines the minimum level of privilege required for a user to view repository statistics, " +
					"to include entitlement-based usage, if applicable. If a user does not have the permission, they won't be " +
					"able to view any statistics, either via the UI, API or CLI." +
					"Valid values include: `Admin`, `Write`, `Read`.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"Admin", "Write", "Read"}, false),
			},
			"wait_for_deletion": {
				Type:        schema.TypeBool,
				Description: "If true, terraform will wait for a repository to be permanently deleted before finishing.",
				Optional:    true,
				Default:     true,
			},
		},
	}
}
