package cloudsmith

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// dataSourceTeamMembersRead lists members for a given team within an organization.
func dataSourceTeamMembersRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")
	teamName := requiredString(d, "team_name")

	req := pc.APIClient.OrgsApi.OrgsTeamsMembersList(pc.Auth, organization, teamName)
	teamMembers, resp, err := pc.APIClient.OrgsApi.OrgsTeamsMembersListExecute(req)
	if is404(resp) {
		// If either org or team not found, clear state
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error retrieving team members for %s/%s: %w", organization, teamName, err)
	}

	if err := d.Set("members", flattenOrganizationTeamMembers(teamMembers.GetMembers())); err != nil {
		return fmt.Errorf("error setting members: %w", err)
	}

	// Ephemeral ID for the data source
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}

// flattenOrganizationTeamMembers converts []OrganizationTeamMembership into []interface{} for state.
func flattenOrganizationTeamMembers(members []cloudsmith.OrganizationTeamMembership) []interface{} {
	out := make([]interface{}, len(members))
	for i, member := range members {
		m := make(map[string]interface{})
		m["role"] = member.GetRole()
		m["user"] = member.GetUser()
		out[i] = m
	}
	return out
}

func dataSourceTeamMembers() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceTeamMembersRead,
		Schema: map[string]*schema.Schema{
			"organization": {
				Type:        schema.TypeString,
				Description: "Organization to which this team belongs.",
				Required:    true,
			},
			"team_name": {
				Type:        schema.TypeString,
				Description: "Name (slug) of the team whose members to list.",
				Required:    true,
			},
			"members": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"role": {Type: schema.TypeString, Computed: true},
					"user": {Type: schema.TypeString, Computed: true},
				}},
			},
		},
	}
}
