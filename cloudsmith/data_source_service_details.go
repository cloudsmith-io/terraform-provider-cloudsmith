package cloudsmith

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// flattenServiceTeamsDS converts []ServiceTeams to []interface{} for the data source.
func flattenServiceTeamsDS(teams []cloudsmith.ServiceTeams) []interface{} {
	out := make([]interface{}, len(teams))
	for i, t := range teams {
		m := make(map[string]interface{})
		m["role"] = t.GetRole()
		m["slug"] = t.GetSlug()
		out[i] = m
	}
	return out
}

func dataSourceServiceDetailsRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	organization := requiredString(d, "organization")
	serviceSlug := requiredString(d, "service")

	req := pc.APIClient.OrgsApi.OrgsServicesRead(pc.Auth, organization, serviceSlug)
	service, resp, err := pc.APIClient.OrgsApi.OrgsServicesReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error retrieving service %s in %s: %w", serviceSlug, organization, err)
	}

	// Map fields (include API key; may be redacted if not freshly created)
	if err := d.Set("created_at", service.GetCreatedAt().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("error setting created_at: %w", err)
	}
	if err := d.Set("created_by", service.GetCreatedBy()); err != nil {
		return fmt.Errorf("error setting created_by: %w", err)
	}
	if err := d.Set("created_by_url", service.GetCreatedByUrl()); err != nil {
		return fmt.Errorf("error setting created_by_url: %w", err)
	}
	if err := d.Set("description", service.GetDescription()); err != nil {
		return fmt.Errorf("error setting description: %w", err)
	}
	if err := d.Set("key", service.GetKey()); err != nil {
		return fmt.Errorf("error setting key: %w", err)
	}
	if service.HasKeyExpiresAt() {
		// key_expires_at only populated if org has API key policy
		if err := d.Set("key_expires_at", service.GetKeyExpiresAt().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("error setting key_expires_at: %w", err)
		}
	} else {
		if err := d.Set("key_expires_at", ""); err != nil {
			return fmt.Errorf("error setting key_expires_at: %w", err)
		}
	}
	if err := d.Set("name", service.GetName()); err != nil {
		return fmt.Errorf("error setting name: %w", err)
	}
	if err := d.Set("role", service.GetRole()); err != nil {
		return fmt.Errorf("error setting role: %w", err)
	}
	if err := d.Set("slug", service.GetSlug()); err != nil {
		return fmt.Errorf("error setting slug: %w", err)
	}
	if err := d.Set("teams", flattenServiceTeamsDS(service.GetTeams())); err != nil {
		return fmt.Errorf("error setting teams: %w", err)
	}

	// ephemeral id
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}

func dataSourceServiceDetails() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceServiceDetailsRead,
		Schema: map[string]*schema.Schema{
			"organization":   {Type: schema.TypeString, Required: true, Description: "Organization to which the service belongs."},
			"service":        {Type: schema.TypeString, Required: true, Description: "Slug of the service to retrieve."},
			"created_at":     {Type: schema.TypeString, Computed: true},
			"created_by":     {Type: schema.TypeString, Computed: true},
			"created_by_url": {Type: schema.TypeString, Computed: true},
			"description":    {Type: schema.TypeString, Computed: true},
			"key":            {Type: schema.TypeString, Computed: true, Sensitive: true},
			"key_expires_at": {Type: schema.TypeString, Computed: true},
			"name":           {Type: schema.TypeString, Computed: true},
			"role":           {Type: schema.TypeString, Computed: true},
			"slug":           {Type: schema.TypeString, Computed: true},
			"teams": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"role": {Type: schema.TypeString, Computed: true},
					"slug": {Type: schema.TypeString, Computed: true},
				}},
			},
		},
	}
}
