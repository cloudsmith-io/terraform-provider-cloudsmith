package cloudsmith

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// retrieveTeamListPage retrieves a single page of teams for an organization.
func retrieveTeamListPage(pc *providerConfig, organization string, pageSize int64, page int64) ([]cloudsmith.OrganizationTeam, int64, error) {
	req := pc.APIClient.OrgsApi.OrgsTeamsList(pc.Auth, organization)
	req = req.Page(page)
	req = req.PageSize(pageSize)

	teamsPage, httpResponse, err := pc.APIClient.OrgsApi.OrgsTeamsListExecute(req)
	if err != nil {
		return nil, 0, err
	}
	pageTotal, err := strconv.ParseInt(httpResponse.Header.Get("X-Pagination-Pagetotal"), 10, 64)
	if err != nil {
		return nil, 0, err
	}
	return teamsPage, pageTotal, nil
}

// retrieveTeamListPages retrieves all pages (or a single page if pageCount provided) of teams.
func retrieveTeamListPages(pc *providerConfig, organization string, pageSize int64, pageCount int64) ([]cloudsmith.OrganizationTeam, error) {
	var pageCurrent int64 = 1

	// Default page size if none provided
	teams := []cloudsmith.OrganizationTeam{}
	if pageSize <= 0 {
		pageSize = 100
	}

	// If no pageCount passed, discover total pages
	if pageCount <= 0 {
		var firstPage []cloudsmith.OrganizationTeam
		var err error
		firstPage, pageCount, err = retrieveTeamListPage(pc, organization, pageSize, 1)
		if err != nil {
			return nil, err
		}
		teams = append(teams, firstPage...)
		pageCurrent++
	}

	for pageCurrent <= pageCount {
		pageData, _, err := retrieveTeamListPage(pc, organization, pageSize, pageCurrent)
		if err != nil {
			return nil, err
		}
		teams = append(teams, pageData...)
		pageCurrent++
	}

	return teams, nil
}

// flattenTeams converts the API team model into a list of maps for Terraform state.
// flattenOrgTeams converts a slice of organization teams to terraform list representation.
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

// dataSourceTeamListRead reads all teams for an organization.
func dataSourceTeamListRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")

	teams, err := retrieveTeamListPages(pc, organization, -1, -1)
	if err != nil {
		return fmt.Errorf("error retrieving teams for organization %s: %w", organization, err)
	}

	if err := d.Set("teams", flattenOrgTeams(teams)); err != nil {
		return fmt.Errorf("error setting teams: %w", err)
	}

	// Use timestamp as ephemeral ID – list data source
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}

// dataSourceTeamList defines the schema for listing teams.
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
