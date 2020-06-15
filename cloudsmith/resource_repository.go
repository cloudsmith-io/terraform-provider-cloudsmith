// Package cloudsmith ...
package cloudsmith

import (
	"github.com/antihax/optional"
	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceRepositoryCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	description := d.Get("description").(string)
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	repositoryType := d.Get("repository_type").(string)
	slug := d.Get("slug").(string)

	opts := &cloudsmith.ReposCreateOpts{
		Data: optional.NewInterface(cloudsmith.ReposCreate{
			Description:       description,
			Name:              name,
			RepositoryTypeStr: repositoryType,
			Slug:              slug,
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
		if err.Error() == "404 Not Found" {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("description", repository.Description)
	d.Set("name", repository.Name)
	d.Set("repository_type", repository.RepositoryTypeStr)
	d.Set("slug", repository.Slug)
	d.Set("slug_perm", repository.SlugPerm)

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
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	repositoryType := d.Get("repository_type").(string)
	slug := d.Get("slug").(string)

	opts := &cloudsmith.ReposPartialUpdateOpts{
		Data: optional.NewInterface(cloudsmith.ReposPartialUpdate{
			Description:       description,
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

	return nil
}

func resourceRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryCreate,
		Read:   resourceRepositoryRead,
		Update: resourceRepositoryUpdate,
		Delete: resourceRepositoryDelete,

		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "Private",
				ValidateFunc: validation.StringInSlice([]string{"Private", "Public"}, false),
			},
			"slug": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"slug_perm": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
