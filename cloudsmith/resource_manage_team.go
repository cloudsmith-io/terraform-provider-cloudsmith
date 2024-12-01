package cloudsmith

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The purpose of this resource is to add/remove users from a team in Cloudsmith

func importManageTeam(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<team_slug>, got: %s", d.Id(),
		)
	}

	d.Set("organization", idParts[0])
	d.Set("team_name", idParts[1])
	return []*schema.ResourceData{d}, nil
}

func resourceManageTeamAdd(d *schema.ResourceData, m interface{}) error {
	// this function will add users to an existing team
	pc := m.(*providerConfig)
	organization := requiredString(d, "organization")
	teamName := requiredString(d, "team_name")

	// Fetching members from the Set, converting to a list
	teamMembersSet := d.Get("members").(*schema.Set).List()
	teamMembersList := make([]cloudsmith.OrganizationTeamMembership, len(teamMembersSet))

	for i, v := range teamMembersSet {
		teamMember := v.(map[string]interface{})
		teamMembersList[i] = cloudsmith.OrganizationTeamMembership{
			Role: teamMember["role"].(string),
			User: teamMember["user"].(string),
		}
	}

	teamMembersData := cloudsmith.OrganizationTeamMembers{
		Members: teamMembersList,
	}

	req := pc.APIClient.OrgsApi.OrgsTeamsMembersCreate(pc.Auth, organization, teamName)
	req = req.Data(teamMembersData)

	_, _, err := pc.APIClient.OrgsApi.OrgsTeamsMembersCreateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%s.%s", organization, teamName))

	return nil
}

// We're using the replace members endpoint here so we need to compare the existing members with the new members and adjust the delta
func resourceManageTeamUpdateRemove(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	organization := requiredString(d, "organization")
	teamName := requiredString(d, "team_name")

	// Fetching members from the Set, converting to a list
	teamMembersSet := d.Get("members").(*schema.Set).List()
	teamMembersList := make([]cloudsmith.OrganizationTeamMembership, len(teamMembersSet))

	for i, v := range teamMembersSet {
		teamMember := v.(map[string]interface{})
		teamMembersList[i] = cloudsmith.OrganizationTeamMembership{
			Role: teamMember["role"].(string),
			User: teamMember["user"].(string),
		}
	}

	teamMembersData := cloudsmith.OrganizationTeamMembers{
		Members: teamMembersList,
	}

	req := pc.APIClient.OrgsApi.OrgsTeamsMembersUpdate(pc.Auth, organization, teamName)
	req = req.Data(teamMembersData)

	_, _, err := pc.APIClient.OrgsApi.OrgsTeamsMembersUpdateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%s.%s", organization, teamName))

	return nil
}

func resourceManageTeamRead(d *schema.ResourceData, m interface{}) error {
	// This function will read the team
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")
	teamName := requiredString(d, "team_name")

	req := pc.APIClient.OrgsApi.OrgsTeamsMembersList(pc.Auth, organization, teamName)

	teamMembers, resp, err := pc.APIClient.OrgsApi.OrgsTeamsMembersListExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}
		return err
	}

	// Map the members correctly
	members := make([]map[string]interface{}, len(teamMembers.GetMembers()))
	for i, member := range teamMembers.GetMembers() {
		members[i] = map[string]interface{}{
			"role": member.Role,
			"user": member.User,
		}
	}

	// Setting the values into the resource data
	d.Set("organization", organization)
	d.Set("team_name", teamName)
	d.Set("members", members)

	// Set the ID to the organization and team name, no slug returned from the API
	d.SetId(fmt.Sprintf("%s.%s", organization, teamName))

	return nil
}

func resourceManageTeam() *schema.Resource {
	return &schema.Resource{
		Create: resourceManageTeamAdd,
		Read:   resourceManageTeamRead,
		Update: resourceManageTeamUpdateRemove,
		Delete: resourceManageTeamUpdateRemove,
		Importer: &schema.ResourceImporter{
			StateContext: importManageTeam,
		},

		Schema: map[string]*schema.Schema{
			"organization": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"team_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"members": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": {
							Type:     schema.TypeString,
							Required: true,
						},
						"user": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Required: true,
			},
		},
	}
}
