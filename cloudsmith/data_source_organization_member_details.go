package cloudsmith

import (
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOrganizationMemberDetailsRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	organization := d.Get("organization").(string)
	member := d.Get("member").(string)

	req := pc.APIClient.OrgsApi.OrgsMembersRead(pc.Auth, organization, member)
	memberDetails, _, err := pc.APIClient.OrgsApi.OrgsMembersReadExecute(req)
	if err != nil {
		return err
	}

	d.Set("email", memberDetails.GetEmail())
	d.Set("has_two_factor", memberDetails.GetHasTwoFactor())
	d.Set("is_active", memberDetails.GetIsActive())
	d.Set("joined_at", memberDetails.GetJoinedAt().Format(time.RFC3339))
	d.Set("last_login_at", memberDetails.GetLastLoginAt().Format(time.RFC3339))
	d.Set("last_login_method", memberDetails.GetLastLoginMethod())
	d.Set("role", memberDetails.GetRole())
	d.Set("user", memberDetails.GetUser())
	d.Set("user_id", memberDetails.GetUserId())
	d.Set("user_name", memberDetails.GetUserName())
	d.Set("user_url", memberDetails.GetUserUrl())
	d.Set("visibility", memberDetails.GetVisibility())

	d.SetId(memberDetails.GetUserId())

	return nil
}

func dataSourceMemberDetails() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceOrganizationMemberDetailsRead,

		Schema: map[string]*schema.Schema{
			"organization": {
				Type:     schema.TypeString,
				Required: true,
			},
			"member": {
				Type:     schema.TypeString,
				Required: true,
			},
			"email": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"has_two_factor": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"is_active": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"joined_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_login_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_login_method": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"role": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"user": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"user_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"user_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"user_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"visibility": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
