package cloudsmith

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/cloudsmith-io/cloudsmith-api-go"
)

func retrieveOrgMemeberListPage(pc *providerConfig, organization string, isActive bool, pageSize int64, pageCount int64) ([]cloudsmith.OrganizationMembership, int64, error) {
	req := pc.APIClient.OrgsApi.OrgsMembersList(pc.Auth, organization)
	req = req.Page(pageCount)
	req = req.PageSize(pageSize)
	req = req.IsActive(isActive)

	membersPage, httpResponse, err := pc.APIClient.OrgsApi.OrgsMembersListExecute(req)
	if err != nil {
		return nil, 0, err
	}
	pageTotal, err := strconv.ParseInt(httpResponse.Header.Get("X-Pagination-Pagetotal"), 10, 64)
	if err != nil {
		return nil, 0, err
	}
	return membersPage, pageTotal, nil
}

func retrieveOrgMemeberListPages(pc *providerConfig, organization string, isActive bool, pageSize int64, pageCount int64) ([]cloudsmith.OrganizationMembership, error) {

	var pageCurrentCount int64 = 1

	// A negative or zero count is assumed to mean retrieve the largest size page
	membersList := []cloudsmith.OrganizationMembership{}
	if pageSize == -1 || pageSize == 0 {
		pageSize = 100
	}

	// If no count is supplied assmumed to mean retrieve all pages
	// we have to retreive a page to get this count
	if pageCount == -1 || pageCount == 0 {
		var membersPage []cloudsmith.OrganizationMembership
		var err error
		membersPage, pageCount, err = retrieveOrgMemeberListPage(pc, organization, isActive, pageSize, 1)
		if err != nil {
			return nil, err
		}
		membersList = append(membersList, membersPage...)
		pageCurrentCount++
	}

	for pageCurrentCount <= pageCount {
		membersPage, _, err := retrieveOrgMemeberListPage(pc, organization, isActive, pageSize, pageCount)
		if err != nil {
			return nil, err
		}
		membersList = append(membersList, membersPage...)
		pageCurrentCount++
	}

	return membersList, nil
}

// dataSourceOrganizationMembersListRead reads the organization members from the API and filters them based on the provided query.
func dataSourceOrganizationMembersListRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := d.Get("namespace").(string)
	isActive := d.Get("is_active").(bool)

	// Retrieve all organization members
	members, err := retrieveOrgMemeberListPages(pc, namespace, isActive, -1, -1)
	if err != nil {
		return fmt.Errorf("error retrieving organization members: %s", err)
	}

	// Map the filtered members to the schema
	if err := d.Set("members", flattenOrganizationMembers(members)); err != nil {
		return fmt.Errorf("error setting members: %s", err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}

// flattenOrganizationMembers maps organization members to a format suitable for the schema.
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
