package cloudsmith

import (
	"errors"
	"fmt"
	"time"

	"github.com/antihax/optional"
	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

var (
	errMessage404                   = "404 Not Found"
	errRepositoryDeleteTimedOut     = errors.New("timed out")
	repositoryDeletionTimeout       = time.Minute * 20
	repositoryDeletionCheckInterval = time.Second * 10
)

func getIndexFilesFromResource(d *schema.ResourceData) *bool {
	var indexFilesPtr *bool

	indexFiles, exists := d.GetOkExists("index_files") //nolint:staticcheck
	if exists {
		indexFilesBool := indexFiles.(bool)
		indexFilesPtr = &indexFilesBool
	}

	return indexFilesPtr
}

func resourceRepositoryCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	description := d.Get("description").(string)
	indexFiles := getIndexFilesFromResource(d)
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	repositoryType := d.Get("repository_type").(string)
	slug := d.Get("slug").(string)
	storageRegion := d.Get("storage_region").(string)

	opts := &cloudsmith.ReposCreateOpts{
		Data: optional.NewInterface(cloudsmith.ReposCreate{
			Description:       description,
			IndexFiles:        indexFiles,
			Name:              name,
			RepositoryTypeStr: repositoryType,
			Slug:              slug,
			StorageRegion:     storageRegion,
		}),
	}

	repository, _, err := pc.APIClient.ReposApi.ReposCreate(pc.Auth, namespace, opts)
	if err != nil {
		return err
	}

	d.SetId(repository.SlugPerm)

	return resourceRepositoryRead(d, m)
}

func resourceRepositoryRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := d.Get("namespace").(string)

	repository, _, err := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, d.Id())
	if err != nil {
		if err.Error() == errMessage404 {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("cdn_url", repository.CdnUrl)
	d.Set("created_at", repository.CreatedAt)
	d.Set("deleted_at", repository.DeletedAt)
	d.Set("description", repository.Description)
	d.Set("index_files", repository.IndexFiles)
	d.Set("name", repository.Name)
	d.Set("namespace_url", repository.NamespaceUrl)
	d.Set("repository_type", repository.RepositoryTypeStr)
	d.Set("self_html_url", repository.SelfHtmlUrl)
	d.Set("self_url", repository.SelfUrl)
	d.Set("slug", repository.Slug)
	d.Set("slug_perm", repository.SlugPerm)
	d.Set("storage_region", repository.StorageRegion)

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

	description := d.Get("description").(string)
	indexFiles := getIndexFilesFromResource(d)
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	repositoryType := d.Get("repository_type").(string)
	slug := d.Get("slug").(string)

	opts := &cloudsmith.ReposPartialUpdateOpts{
		Data: optional.NewInterface(cloudsmith.ReposPartialUpdate{
			Description:       description,
			IndexFiles:        indexFiles,
			Name:              name,
			RepositoryTypeStr: repositoryType,
			Slug:              slug,
		}),
	}

	repository, _, err := pc.APIClient.ReposApi.ReposPartialUpdate(pc.Auth, namespace, d.Id(), opts)
	if err != nil {
		return err
	}

	d.SetId(repository.SlugPerm)

	return resourceRepositoryRead(d, m)
}

func resourceRepositoryDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := d.Get("namespace").(string)

	_, err := pc.APIClient.ReposApi.ReposDelete(pc.Auth, namespace, d.Id())
	if err != nil {
		return err
	}

	if d.Get("wait_for_deletion").(bool) {
		if err := resourceRepositoryWaitUntilDeleted(d, m); err != nil {
			return fmt.Errorf("error waiting for repository (%s) to be deleted: %w", d.Id(), err)
		}
	}

	return nil
}

func resourceRepositoryWaitUntilDeleted(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := d.Get("namespace").(string)

	for start := time.Now(); time.Since(start) < repositoryDeletionTimeout; {
		_, _, err := pc.APIClient.ReposApi.ReposRead(pc.Auth, namespace, d.Id())
		if err != nil {
			if err.Error() == errMessage404 {
				return nil
			}

			return err
		}

		time.Sleep(repositoryDeletionCheckInterval)
	}

	return errRepositoryDeleteTimedOut
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
