package cloudsmith

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"
)

var (
	importSentinel = "__import__"
	roles          = []string{
		"Manager",
		"Member",
	}
)

// expandTeams extracts team assignments from TF state as a *schema.Set and
// converts to a slice of strings we can use when interacting with the
// Cloudsmith API.
func expandTeams(d *schema.ResourceData) []cloudsmith.ServiceTeams {
	set := d.Get("team").(*schema.Set)

	teams := lo.Map(set.List(), func(x interface{}, index int) cloudsmith.ServiceTeams {
		m := x.(map[string]interface{})
		t := cloudsmith.ServiceTeams{}
		role := m["role"].(string)
		if role != "" {
			t.SetRole(role)
		}
		t.SetSlug(m["slug"].(string))

		return t
	})

	if len(teams) == 0 {
		return nil
	}

	return teams
}

// flattenTeams takes a slice of cloudsmith.ServiceTeams as returned by the
// Cloudsmith API and converts to a *schema.Set that can be stored in TF state.
func flattenTeams(teams []cloudsmith.ServiceTeams) *schema.Set {
	teamSchema := resourceService().Schema["team"].Elem.(*schema.Resource)
	set := schema.NewSet(schema.HashResource(teamSchema), []interface{}{})
	for _, team := range teams {
		set.Add(map[string]interface{}{
			"role": team.GetRole(),
			"slug": team.GetSlug(),
		})
	}
	return set
}

func importService(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<service_slug>, got: %s", d.Id(),
		)
	}

	// it's not possible to retrieve a service's API key via API after creation,
	// so when we import a service the key is unavailable. Setting to a known
	// sentinel value here allows us to detect this case later and warn the user
	// that things may not work as expected.
	d.Set("key", importSentinel)

	d.Set("organization", idParts[0])
	d.SetId(idParts[1])
	return []*schema.ResourceData{d}, nil
}

func resourceServiceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)

	org := requiredString(d, "organization")

	req := pc.APIClient.OrgsApi.OrgsServicesCreate(pc.Auth, org)
	req = req.Data(cloudsmith.ServiceRequest{
		Description: optionalString(d, "description"),
		Name:        requiredString(d, "name"),
		Role:        optionalString(d, "role"),
		Teams:       expandTeams(d),
	})

	service, _, err := pc.APIClient.OrgsApi.OrgsServicesCreateExecute(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(service.GetSlug())

	// normally we'd read this value back on read, but it's only returned over
	// the API when the resource is created, otherwise it's redacted.
	// Unfortunately this means that if the user needs to rotate the service's
	// API key then the only way to do it and have it reflected properly in
	// Terraform is to taint the whole resource and let Terraform recreate it.
	if requiredBool(d, "store_api_key") {
		d.Set("key", service.GetKey())
	}
	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsServicesRead(pc.Auth, org, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsServicesReadExecute(req); err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return err
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return diag.Errorf("error waiting for service (%s) to be created: %s", d.Id(), err)
	}

	return resourceServiceRead(ctx, d, m)
}

func resourceServiceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)

	org := requiredString(d, "organization")

	req := pc.APIClient.OrgsApi.OrgsServicesRead(pc.Auth, org, d.Id())

	const lastFourChars = 4

	service, resp, err := pc.APIClient.OrgsApi.OrgsServicesReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	d.Set("description", service.GetDescription())
	d.Set("name", service.GetName())
	d.Set("role", service.GetRole())
	d.Set("slug", service.GetSlug())
	d.Set("team", flattenTeams(service.GetTeams()))

	// since we don't get the full API key when reading a service back from the
	// API, we need to check if it has changed and if so warn the user that
	// they'll need to recreate the resource if they want to pull the new key
	// into Terraform. This can be accomplished by tainting.
	var diags diag.Diagnostics

	if requiredBool(d, "store_api_key") {
		if key, ok := d.GetOk("key"); ok {
			// "key" attribute exists in Terraform state
			existingKey := key.(string)
			existingLastFour := existingKey[len(existingKey)-lastFourChars:]
			newLastFour := service.GetKey()[len(service.GetKey())-lastFourChars:]

			if existingLastFour != newLastFour {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "API key has changed",
					Detail: "API key for this service has changed outside of Terraform. If this " +
						"key is used within Terraform, the resource must be tainted or otherwise " +
						"recreated to retrieve the new value.",
					AttributePath: cty.Path{cty.GetAttrStep{Name: "key"}},
				})
			}
		} else {
			// "key" attribute does not exist in Terraform state, grab it from the API response
			d.Set("key", service.GetKey())
		}
	} else {
		// "store_api_key" is set to False, set the "key" to "**redacted**"
		d.Set("key", "**redacted**")
	}

	// organization is not returned from the service read endpoint, so we can
	// use the value stored in resource state. We rely on ForceNew to ensure if
	// it changes a new resource is created.
	d.Set("organization", org)

	return diags
}

func resourceServiceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)

	org := requiredString(d, "organization")

	req := pc.APIClient.OrgsApi.OrgsServicesPartialUpdate(pc.Auth, org, d.Id())
	req = req.Data(cloudsmith.ServiceRequestPatch{
		Description: optionalString(d, "description"),
		Name:        optionalString(d, "name"),
		Role:        optionalString(d, "role"),
		Teams:       expandTeams(d),
	})

	service, _, err := pc.APIClient.OrgsApi.OrgsServicesPartialUpdateExecute(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(service.GetSlug())

	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for a
		// service being updated (changes incoming on the API side)
		time.Sleep(time.Second * 5)
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return diag.Errorf("error waiting for service (%s) to be updated: %s", d.Id(), err)
	}

	return resourceServiceRead(ctx, d, m)
}

func resourceServiceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)

	org := requiredString(d, "organization")

	req := pc.APIClient.OrgsApi.OrgsServicesDelete(pc.Auth, org, d.Id())
	_, err := pc.APIClient.OrgsApi.OrgsServicesDeleteExecute(req)
	if err != nil {
		return diag.FromErr(err)
	}

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsServicesRead(pc.Auth, org, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsServicesReadExecute(req); err != nil {
			if is404(resp) {
				return nil
			}
			return err
		}
		return errKeepWaiting
	}
	if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
		return diag.Errorf("error waiting for service (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

//nolint:funlen
func resourceService() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServiceCreate,
		ReadContext:   resourceServiceRead,
		UpdateContext: resourceServiceUpdate,
		DeleteContext: resourceServiceDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importService,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Type:         schema.TypeString,
				Description:  "A description of the service's purpose.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"key": {
				Type:        schema.TypeString,
				Description: "The service's API key.",
				Computed:    true,
				Sensitive:   true,
			},
			"name": {
				Type:         schema.TypeString,
				Description:  "A descriptive name for the service.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"organization": {
				Type:         schema.TypeString,
				Description:  "Organization to which this service belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"role": {
				Type:         schema.TypeString,
				Description:  "The service's role in the organization.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(roles, false),
			},
			"slug": {
				Type:        schema.TypeString,
				Description: "The slug identifies the service in URIs.",
				Computed:    true,
			},
			"team": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": {
							Type:         schema.TypeString,
							Description:  "The service's role in the organization.",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.StringInSlice(roles, false),
						},
						"slug": {
							Type:         schema.TypeString,
							Description:  "The team the service should be added to.",
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},
					},
				},
				Optional: true,
			},
			"store_api_key": {
				Type:        schema.TypeBool,
				Description: "Whether to include the service's API key in Terraform state.",
				Optional:    true,
				Default:     true,
			},
		},
	}
}
