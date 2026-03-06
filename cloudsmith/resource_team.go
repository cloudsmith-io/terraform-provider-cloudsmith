package cloudsmith

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func importTeam(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<team_slug>, got: %s", d.Id(),
		)
	}

	d.Set("organization", idParts[0])
	d.SetId(idParts[1])
	return []*schema.ResourceData{d}, nil
}

func resourceTeamCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, "organization")

	req := pc.APIClient.OrgsApi.OrgsTeamsCreate(pc.Auth, org)
	req = req.Data(cloudsmith.OrganizationTeamRequest{
		Description: nullableString(d, "description"),
		Name:        requiredString(d, "name"),
		Slug:        optionalString(d, "slug"),
		Visibility:  optionalString(d, "visibility"),
	})

	team, resp, err := pc.APIClient.OrgsApi.OrgsTeamsCreateExecute(req)
	if err != nil {
		if resp != nil && resp.StatusCode == 422 {
			bodyBytes, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				return fmt.Errorf("encountered an error: %w", err)
			}
			var apiError map[string]interface{}
			jsonErr := json.Unmarshal(bodyBytes, &apiError)
			if jsonErr != nil {
				return fmt.Errorf("encountered an error: %w", err)
			}
			if fields, ok := apiError["fields"].(map[string]interface{}); ok {
				for _, v := range fields {
					if messages, ok := v.([]interface{}); ok && len(messages) > 0 {
						if message, ok := messages[0].(string); ok {
							return fmt.Errorf("error: %s", message)
						}
					}
				}
			}
			return fmt.Errorf("encountered an error: %v", apiError)
		}
		return err
	}

	d.SetId(team.GetSlugPerm())

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsTeamsRead(pc.Auth, org, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsTeamsReadExecute(req); err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return err
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for team (%s) to be created: %w", d.Id(), err)
	}

	return resourceTeamRead(d, m)
}

func resourceTeamRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, "organization")

	req := pc.APIClient.OrgsApi.OrgsTeamsRead(pc.Auth, org, d.Id())
	team, resp, err := pc.APIClient.OrgsApi.OrgsTeamsReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("description", team.GetDescription())
	d.Set("name", team.GetName())
	d.Set("slug", team.GetSlug())
	d.Set("slug_perm", team.GetSlugPerm())
	d.Set("visibility", team.GetVisibility())

	// organization is not returned from the team read endpoint, so we can use
	// the value stored in resource state. We rely on ForceNew to ensure if it
	// changes a new resource is created.
	d.Set("organization", org)

	// since we allow import using either the slug or the slug_perm, we want to
	// normalize the ID and always use the slug_perm when we can. This reset
	// allows us to set the ID unconditionally, regardless of what the user
	// passed.
	d.SetId(team.GetSlugPerm())

	return nil
}

func resourceTeamUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, "organization")

	req := pc.APIClient.OrgsApi.OrgsTeamsPartialUpdate(pc.Auth, org, d.Id())
	req = req.Data(cloudsmith.OrganizationTeamRequestPatch{
		Description: nullableString(d, "description"),
		Name:        optionalString(d, "name"),
		Slug:        optionalString(d, "slug"),
		Visibility:  optionalString(d, "visibility"),
	})
	team, _, err := pc.APIClient.OrgsApi.OrgsTeamsPartialUpdateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(team.GetSlugPerm())

	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for a
		// team being updated (changes incoming on the API side)
		time.Sleep(time.Second * 5)
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for team (%s) to be updated: %w", d.Id(), err)
	}

	return resourceTeamRead(d, m)
}

func resourceTeamDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	org := requiredString(d, "organization")

	req := pc.APIClient.OrgsApi.OrgsTeamsDelete(pc.Auth, org, d.Id())
	_, err := pc.APIClient.OrgsApi.OrgsTeamsDeleteExecute(req)
	if err != nil {
		return err
	}

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsTeamsRead(pc.Auth, org, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsTeamsReadExecute(req); err != nil {
			if is404(resp) {
				return nil
			}
			return err
		}
		return errKeepWaiting
	}
	if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
		return fmt.Errorf("error waiting for team (%s) to be deleted: %w", d.Id(), err)
	}

	return nil
}

//nolint:funlen
func resourceTeam() *schema.Resource {
	return &schema.Resource{
		Create: resourceTeamCreate,
		Read:   resourceTeamRead,
		Update: resourceTeamUpdate,
		Delete: resourceTeamDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importTeam,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Type:         schema.TypeString,
				Description:  "A description of the team's purpose.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"name": {
				Type:         schema.TypeString,
				Description:  "A descriptive name for the team.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"organization": {
				Type:         schema.TypeString,
				Description:  "Organization to which this team belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"slug": {
				Type:         schema.TypeString,
				Description:  "The slug identifies the team in URIs.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"slug_perm": {
				Type: schema.TypeString,
				Description: "The slug_perm immutably identifies the team. " +
					"It will never change once a team has been created.",
				Computed: true,
			},
			"visibility": {
				Type:         schema.TypeString,
				Description:  "Controls if the team is visible or hidden from non-members.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"Visible", "Hidden"}, false),
			},
		},
	}
}
