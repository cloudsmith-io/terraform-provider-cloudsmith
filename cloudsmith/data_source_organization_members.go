package cloudsmith

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/cloudsmith-io/cloudsmith-api-go"
)

func retrieveOrgMemeberListPages(pc *providerConfig, organization string, isActive bool) ([]cloudsmith.OrganizationMembership, error) {
	exec := func(page, ps int64) ([]cloudsmith.OrganizationMembership, *http.Response, error) {
		req := pc.APIClient.OrgsApi.OrgsMembersList(pc.Auth, organization).
			Page(page).
			PageSize(ps).
			IsActive(isActive)
		return pc.APIClient.OrgsApi.OrgsMembersListExecute(req)
	}
	return PaginateAllHTTP[cloudsmith.OrganizationMembership](exec, PaginationOptions{})
}

func dataSourceOrganizationMembersListRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := d.Get("namespace").(string)
	isActive := d.Get("is_active").(bool)

	members, err := retrieveOrgMemeberListPages(pc, namespace, isActive)
	if err != nil {
		return fmt.Errorf("error retrieving organization members: %s", err)
	}

	if err := d.Set("members", flattenOrganizationMembers(members)); err != nil {
		return fmt.Errorf("error setting members: %s", err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}

func flattenOrganizationMembers(members []cloudsmith.OrganizationMembership) []interface{} {
	var out []interface{}
	for _, member := range members {
		m := make(map[string]interface{})
		m["email"] = member.GetEmail()
		m["has_two_factor"] = member.GetHasTwoFactor()
		m["is_active"] = member.GetIsActive()
		m["joined_at"] = member.GetJoinedAt().Format(time.RFC3339) // Assuming time.Time should be formatted as a string
		m["last_login_at"] = member.GetLastLoginAt().Format(time.RFC3339)
		m["last_login_method"] = member.GetLastLoginMethod()
		m["role"] = member.GetRole()
		m["user"] = member.GetUser()
		m["user_id"] = member.GetUserId()
		out = append(out, m)
	}
	return out
}

func dataSourceOrganizationMembersList() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceOrganizationMembersListRead,
		Schema: map[string]*schema.Schema{
			"namespace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"is_active": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"members": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
							Type:     schema.TypeString, // Assuming time.Time should be represented as a string
							Computed: true,
						},
						"last_login_at": {
							Type:     schema.TypeString, // Assuming time.Time should be represented as a string
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
					},
				},
			},
		},
	}
}
