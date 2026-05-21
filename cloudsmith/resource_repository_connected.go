package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	TargetRepository = "target_repository"
)

func importRepositoryConnected(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 3 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <namespace>.<repository>.<slug_perm>, got: %s", d.Id(),
		)
	}

	_ = d.Set(Namespace, idParts[0])
	_ = d.Set(Repository, idParts[1])
	d.SetId(idParts[2])
	return []*schema.ResourceData{d}, nil
}

func resourceRepositoryConnectedCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)
	targetRepository := requiredString(d, TargetRepository)

	data := cloudsmith.NewConnectedRepositoryRequest(targetRepository)
	data.SetIsActive(requiredBool(d, IsActive))
	if priority := optionalInt64(d, Priority); priority != nil {
		data.SetPriority(*priority)
	}

	req := pc.APIClient.ReposApi.ReposConnectedCreate(pc.Auth, namespace, repository)
	req = req.Data(*data)

	connected, _, err := pc.APIClient.ReposApi.ReposConnectedCreateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(connected.GetSlugPerm())

	readFunc := func() (*http.Response, error) {
		readReq := pc.APIClient.ReposApi.ReposConnectedRead(pc.Auth, namespace, repository, d.Id())
		_, resp, err := pc.APIClient.ReposApi.ReposConnectedReadExecute(readReq)
		return resp, err
	}
	if err := waitForCreation(readFunc, "connected repository", d.Id()); err != nil {
		return err
	}

	return resourceRepositoryConnectedRead(d, m)
}

func resourceRepositoryConnectedRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)

	req := pc.APIClient.ReposApi.ReposConnectedRead(pc.Auth, namespace, repository, d.Id())
	connected, resp, err := pc.APIClient.ReposApi.ReposConnectedReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}
		return err
	}

	_ = d.Set(Namespace, namespace)
	_ = d.Set(Repository, repository)
	_ = d.Set(TargetRepository, connected.GetTargetRepository())
	_ = d.Set(IsActive, connected.GetIsActive())
	_ = d.Set(Priority, connected.GetPriority())
	_ = d.Set(SlugPerm, connected.GetSlugPerm())
	_ = d.Set(CreatedAt, timeToString(connected.GetCreatedAt()))

	return nil
}

func resourceRepositoryConnectedUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)

	data := cloudsmith.NewConnectedRepositoryRequestPatch()
	if d.HasChange(IsActive) {
		data.SetIsActive(requiredBool(d, IsActive))
	}
	if d.HasChange(Priority) {
		if priority := optionalInt64(d, Priority); priority != nil {
			data.SetPriority(*priority)
		}
	}

	req := pc.APIClient.ReposApi.ReposConnectedPartialUpdate(pc.Auth, namespace, repository, d.Id())
	req = req.Data(*data)

	_, _, err := pc.APIClient.ReposApi.ReposConnectedPartialUpdateExecute(req)
	if err != nil {
		return err
	}

	if err := waitForUpdate("connected repository", d.Id()); err != nil {
		return err
	}

	return resourceRepositoryConnectedRead(d, m)
}

func resourceRepositoryConnectedDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)

	req := pc.APIClient.ReposApi.ReposConnectedDelete(pc.Auth, namespace, repository, d.Id())
	_, err := pc.APIClient.ReposApi.ReposConnectedDeleteExecute(req)
	if err != nil {
		return err
	}

	readFunc := func() (*http.Response, error) {
		readReq := pc.APIClient.ReposApi.ReposConnectedRead(pc.Auth, namespace, repository, d.Id())
		_, resp, err := pc.APIClient.ReposApi.ReposConnectedReadExecute(readReq)
		return resp, err
	}
	return waitForDeletion(readFunc, "connected repository", d.Id())
}

//nolint:funlen
func resourceRepositoryConnected() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryConnectedCreate,
		Read:   resourceRepositoryConnectedRead,
		Update: resourceRepositoryConnectedUpdate,
		Delete: resourceRepositoryConnectedDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importRepositoryConnected,
		},

		Schema: map[string]*schema.Schema{
			Namespace: {
				Type:         schema.TypeString,
				Description:  "Organization to which the source Repository belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			Repository: {
				Type:         schema.TypeString,
				Description:  "Source Repository from which to add the connection.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			TargetRepository: {
				Type:         schema.TypeString,
				Description:  "The slug of the target Repository to connect to.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			IsActive: {
				Type:        schema.TypeBool,
				Description: "Whether the connection is active.",
				Optional:    true,
				Default:     true,
			},
			Priority: {
				Type:         schema.TypeInt,
				Description:  "Order in which connected repositories are checked (ascending, starting at 1). Ties are broken by age (oldest first).",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntBetween(1, 32767),
			},
			SlugPerm: {
				Type:        schema.TypeString,
				Description: "The immutable slug identifier of the connection.",
				Computed:    true,
			},
			CreatedAt: {
				Type:        schema.TypeString,
				Description: "The date and time when the connection was created (RFC 3339).",
				Computed:    true,
			},
		},
	}
}
