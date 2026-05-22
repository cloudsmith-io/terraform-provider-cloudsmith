package cloudsmith

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func retrieveTeamListPages(pc *providerConfig, organization string) ([]cloudsmith.OrganizationTeam, error) {
	exec := func(page, ps int64) ([]cloudsmith.OrganizationTeam, *http.Response, error) {
		req := pc.APIClient.OrgsApi.OrgsTeamsList(pc.Auth, organization).
			Page(page).
			PageSize(ps)
		return pc.APIClient.OrgsApi.OrgsTeamsListExecute(req)
	}
	return PaginateAllHTTP[cloudsmith.OrganizationTeam](exec, PaginationOptions{})
}

func flattenOrgTeams(in []cloudsmith.OrganizationTeam) []interface{} {
	out := make([]interface{}, len(in))
	for i, team := range in {
		m := make(map[string]interface{})
		m["description"] = team.GetDescription()
		m["name"] = team.GetName()
		m["slug"] = team.GetSlug()
		m["slug_perm"] = team.GetSlugPerm()
		m["visibility"] = team.GetVisibility()
		out[i] = m
	}
	return out
}

func dataSourceTeamListRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")

	teams, err := retrieveTeamListPages(pc, organization)
	if err != nil {
		return fmt.Errorf("error retrieving teams for organization %s: %w", organization, err)
	}

	if err := d.Set("teams", flattenOrgTeams(teams)); err != nil {
		return fmt.Errorf("error setting teams: %w", err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}

func dataSourceTeamList() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceTeamListRead,
		Schema: map[string]*schema.Schema{
			"organization": {
				Type:        schema.TypeString,
				Description: "Organization within which to list teams.",
				Required:    true,
			},
			"teams": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"description": {Type: schema.TypeString, Computed: true},
					"name":        {Type: schema.TypeString, Computed: true},
					"slug":        {Type: schema.TypeString, Computed: true},
					"slug_perm":   {Type: schema.TypeString, Computed: true},
					"visibility":  {Type: schema.TypeString, Computed: true},
				}},
			},
		},
	}
}
