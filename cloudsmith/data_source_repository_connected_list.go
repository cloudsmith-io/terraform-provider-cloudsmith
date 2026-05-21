package cloudsmith

import (
	"fmt"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceRepositoryConnectedList() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRepositoryConnectedListRead,

		Schema: map[string]*schema.Schema{
			Namespace: {
				Type:         schema.TypeString,
				Description:  "Organization to which the source Repository belongs.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			Repository: {
				Type:         schema.TypeString,
				Description:  "Source Repository whose connected repositories are listed.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"connected_repositories": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						TargetRepository: {
							Type:        schema.TypeString,
							Description: "The slug of the connected target repository.",
							Computed:    true,
						},
						IsActive: {
							Type:        schema.TypeBool,
							Description: "Whether the connection is active.",
							Computed:    true,
						},
						Priority: {
							Type:        schema.TypeInt,
							Description: "Lookup order priority (ascending).",
							Computed:    true,
						},
						SlugPerm: {
							Type:        schema.TypeString,
							Description: "The immutable slug identifier of the connection.",
							Computed:    true,
						},
						CreatedAt: {
							Type:        schema.TypeString,
							Description: "The date and time when the connection was created.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceRepositoryConnectedListRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)

	connected, err := retrieveAllConnectedRepositories(pc, namespace, repository)
	if err != nil {
		return err
	}

	_ = d.Set("connected_repositories", flattenConnectedRepositories(connected))
	d.SetId(fmt.Sprintf("%s/%s/connected", namespace, repository))

	return nil
}

func retrieveAllConnectedRepositories(pc *providerConfig, namespace, repository string) ([]cloudsmith.ConnectedRepository, error) {
	var all []cloudsmith.ConnectedRepository
	var page int64 = 1
	const pageSize int64 = 100

	for {
		req := pc.APIClient.ReposApi.ReposConnectedList(pc.Auth, namespace, repository)
		req = req.Page(page)
		req = req.PageSize(pageSize)

		resp, _, err := pc.APIClient.ReposApi.ReposConnectedListExecute(req)
		if err != nil {
			return nil, err
		}

		results := resp.GetResults()
		all = append(all, results...)

		if int64(len(results)) < pageSize {
			break
		}
		page++
	}

	return all, nil
}

func flattenConnectedRepositories(items []cloudsmith.ConnectedRepository) []interface{} {
	out := make([]interface{}, len(items))
	for i, item := range items {
		out[i] = map[string]interface{}{
			TargetRepository: item.GetTargetRepository(),
			IsActive:         item.GetIsActive(),
			Priority:         item.GetPriority(),
			SlugPerm:         item.GetSlugPerm(),
			CreatedAt:        timeToString(item.GetCreatedAt()),
		}
	}
	return out
}
