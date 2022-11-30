package cloudsmith

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceRepositoryCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")

	req := pc.APIClient.ReposApi.ReposCreate(pc.Auth, namespace)
	req = req.Data(cloudsmith.ReposCreate{
		Description:       optionalString(d, "description"),
		IndexFiles:        optionalBool(d, "index_files"),
		Name:              requiredString(d, "name"),
		RepositoryTypeStr: optionalString(d, "repository_type"),
		Slug:              optionalString(d, "slug"),
		StorageRegion:     optionalString(d, "storage_region"),
	})

	repository, _, err := pc.APIClient.ReposApi.ReposCreateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(repository.GetSlugPerm())

	checkerFunc := func() error {
		req := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, d.Id())
		if _, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req); err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return errKeepWaiting
			}
			return err
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
	repository, _, err := pc.APIClient.ReposApi.ReposReadExecute(req)
	if err != nil {
		if err.Error() == errMessage404 {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("cdn_url", repository.GetCdnUrl())
	d.Set("created_at", repository.GetCreatedAt())
	d.Set("deleted_at", repository.GetDeletedAt())
	d.Set("description", repository.GetDescription())
	d.Set("index_files", repository.GetIndexFiles())
	d.Set("name", repository.GetName())
	d.Set("namespace_url", repository.GetNamespaceUrl())
	d.Set("repository_type", repository.GetRepositoryTypeStr())
	d.Set("self_html_url", repository.GetSelfHtmlUrl())
	d.Set("self_url", repository.GetSelfUrl())
	d.Set("slug", repository.GetSlug())
	d.Set("slug_perm", repository.GetSlugPerm())
	d.Set("storage_region", repository.GetStorageRegion())

	// namespace returned from the API is always the user-facing slug, but the
	// resource may have been created in terraform with the slug_perm instead,
	// so we don't want to overwrite it with the value from the API ever,
	// instead we rely on ForceNew to ensure if it changes a new resource is
	// created.
	d.Set("namespace", namespace)

	return nil
}

func resourceRepositoryUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")

	req := pc.APIClient.ReposApi.ReposPartialUpdate(pc.Auth, namespace, d.Id())
	req = req.Data(cloudsmith.ReposPartialUpdate{
		Description:       optionalString(d, "description"),
		IndexFiles:        optionalBool(d, "index_files"),
		Name:              optionalString(d, "name"),
		RepositoryTypeStr: optionalString(d, "repository_type"),
		Slug:              optionalString(d, "slug"),
	})
	repository, _, err := pc.APIClient.ReposApi.ReposPartialUpdateExecute(req)
	if err != nil {
		return err
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
		return err
	}

	if requiredBool(d, "wait_for_deletion") {
		checkerFunc := func() error {
			req := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, d.Id())
			if _, resp, err := pc.APIClient.ReposApi.ReposReadExecute(req); err != nil {
				if resp.StatusCode == http.StatusNotFound {
					return nil
				}
				return err
			}
			return errKeepWaiting
		}
		if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
			return fmt.Errorf("error waiting for repository (%s) to be deleted: %w", d.Id(), err)
		}
	}

	return nil
}

//nolint:funlen
func resourceRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryCreate,
		Read:   resourceRepositoryRead,
		Update: resourceRepositoryUpdate,
		Delete: resourceRepositoryDelete,

		Schema: map[string]*schema.Schema{
			"cdn_url": {
				Type:        schema.TypeString,
				Description: "Base URL from which packages and other artifacts are downloaded.",
				Computed:    true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "ISO 8601 timestamp at which the repository was created.",
				Computed:    true,
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
			"index_files": {
				Type: schema.TypeBool,
				Description: "If checked, files contained in packages will be indexed, which increase the " +
					"synchronisation time required for packages. Note that it is recommended you keep this " +
					"enabled unless the synchronisation time is significantly impacted.",
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:         schema.TypeString,
				Description:  "A descriptive name for the repository.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
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
			"repository_type": {
				Type: schema.TypeString,
				Description: "The repository type changes how it is accessed and billed. Private repositories " +
					"can only be used on paid plans, but are visible only to you or authorised delegates. Public " +
					"repositories are free to use on all plans and visible to all Cloudsmith users.",
				Optional:     true,
				Default:      "Private",
				ValidateFunc: validation.StringInSlice([]string{"Private", "Public"}, false),
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
			"slug": {
				Type:         schema.TypeString,
				Description:  "The slug identifies the repository in URIs.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"slug_perm": {
				Type: schema.TypeString,
				Description: "The slug_perm immutably identifies the repository. " +
					"It will never change once a repository has been created.",
				Computed: true,
			},
			"storage_region": {
				Type:         schema.TypeString,
				Description:  "The Cloudsmith region in which package files are stored.",
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
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
