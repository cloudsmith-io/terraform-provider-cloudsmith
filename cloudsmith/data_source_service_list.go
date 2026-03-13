package cloudsmith

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// retrieveServiceListPage fetches a single page of services.
func retrieveServiceListPage(pc *providerConfig, organization string, pageSize int64, page int64, query, sort *string) ([]cloudsmith.Service, int64, error) {
	req := pc.APIClient.OrgsApi.OrgsServicesList(pc.Auth, organization)
	req = req.Page(page)
	req = req.PageSize(pageSize)
	if query != nil && *query != "" {
		req = req.Query(*query)
	}
	if sort != nil && *sort != "" {
		req = req.Sort(*sort)
	}

	servicesPage, httpResp, err := pc.APIClient.OrgsApi.OrgsServicesListExecute(req)
	if err != nil {
		return nil, 0, err
	}
	pageTotal, err := strconv.ParseInt(httpResp.Header.Get("X-Pagination-Pagetotal"), 10, 64)
	if err != nil {
		return nil, 0, err
	}
	return servicesPage, pageTotal, nil
}

// retrieveServiceListPages retrieves all pages of services.
func retrieveServiceListPages(pc *providerConfig, organization string, pageSize, pageCount int64, query, sort *string) ([]cloudsmith.Service, error) {
	var current int64 = 1
	services := []cloudsmith.Service{}
	if pageSize <= 0 {
		pageSize = 100
	}
	if pageCount <= 0 {
		first, total, err := retrieveServiceListPage(pc, organization, pageSize, 1, query, sort)
		if err != nil {
			return nil, err
		}
		services = append(services, first...)
		pageCount = total
		current++
	}
	for current <= pageCount {
		pageData, _, err := retrieveServiceListPage(pc, organization, pageSize, current, query, sort)
		if err != nil {
			return nil, err
		}
		services = append(services, pageData...)
		current++
	}
	return services, nil
}

// flattenServices converts []Service into []interface{} for TF state.
func flattenServices(in []cloudsmith.Service) []interface{} {
	out := make([]interface{}, len(in))
	for i, s := range in {
		m := make(map[string]interface{})
		m["created_at"] = s.GetCreatedAt().Format(time.RFC3339)
		m["created_by"] = s.GetCreatedBy()
		m["created_by_url"] = s.GetCreatedByUrl()
		m["description"] = s.GetDescription()
		// API key may be redacted unless freshly created; include as-is for completeness.
		m["key"] = s.GetKey()
		if s.HasKeyExpiresAt() {
			m["key_expires_at"] = s.GetKeyExpiresAt().Format(time.RFC3339)
		} else {
			m["key_expires_at"] = ""
		}
		m["name"] = s.GetName()
		m["role"] = s.GetRole()
		m["slug"] = s.GetSlug()
		// Flatten teams
		teams := make([]interface{}, len(s.GetTeams()))
		for ti, t := range s.GetTeams() {
			teams[ti] = map[string]interface{}{
				"role": t.GetRole(),
				"slug": t.GetSlug(),
			}
		}
		m["teams"] = teams
		out[i] = m
	}
	return out
}

func dataSourceServiceListRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	organization := requiredString(d, "organization")

	// Optional filtering/sorting arguments
	var queryPtr, sortPtr *string
	if v, ok := d.GetOk("query"); ok {
		qs := v.(string)
		queryPtr = &qs
	}
	if v, ok := d.GetOk("sort"); ok {
		ss := v.(string)
		sortPtr = &ss
	}

	// Always iterate all pages (Terraform expectation).
	services, err := retrieveServiceListPages(pc, organization, -1, -1, queryPtr, sortPtr)
	if err != nil {
		return fmt.Errorf("error retrieving services for %s: %w", organization, err)
	}

	if err := d.Set("services", flattenServices(services)); err != nil {
		return fmt.Errorf("error setting services: %w", err)
	}

	// ephemeral ID
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}

func dataSourceServiceList() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceServiceListRead,
		Schema: map[string]*schema.Schema{
			"organization": {Type: schema.TypeString, Required: true, Description: "Organization within which to list service accounts."},
			"query":        {Type: schema.TypeString, Optional: true, Description: "Search query (e.g. 'name:my-service' or 'role:Member')."},
			"sort":         {Type: schema.TypeString, Optional: true, Description: "Sort field (e.g. 'created_at', '-created_at', 'name', '-name', 'role')."},
			"services": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
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
				}},
			},
		},
	}
}
