package cloudsmith

import (
	"strconv"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func retrieveEntitlmentTokenListPage(pc *providerConfig, namespace string, repository string, page int64, pageSize int64, showToken bool, query string, activeToken bool) ([]cloudsmith.RepositoryToken, int64, error) {
	req := pc.APIClient.EntitlementsApi.EntitlementsList(pc.Auth, namespace, repository)
	req = req.Page(page)
	req = req.PageSize(pageSize)
	req = req.ShowTokens(showToken)
	req = req.Query(query)
	req = req.Active(activeToken)

	tokensPage, httpResponse, err := pc.APIClient.EntitlementsApi.EntitlementsListExecute(req)
	if err != nil {
		return nil, 0, err
	}
	pageTotal, err := strconv.ParseInt(httpResponse.Header.Get("X-Pagination-Pagetotal"), 10, 64)
	if err != nil {
		return nil, 0, err
	}
	return tokensPage, pageTotal, nil
}

func retrieveEntitlmentListPages(pc *providerConfig, namespace string, repository string, query string, pageSize int64, pageCount int64, showToken bool, activeToken bool) ([]cloudsmith.RepositoryToken, error) {

	var pageCurrentCount int64 = 1

	// A negative or zero count is assumed to mean retrieve the largest size page
	tokensList := []cloudsmith.RepositoryToken{}
	if pageSize == -1 || pageSize == 0 {
		pageSize = 100
	}

	// If no count is supplied assmumed to mean retrieve all pages
	// we have to retreive a page to get this count
	if pageCount == -1 || pageCount == 0 {
		var tokensPage []cloudsmith.RepositoryToken
		var err error
		tokensPage, pageCount, err = retrieveEntitlmentTokenListPage(pc, namespace, repository, 1, pageSize, showToken, query, activeToken)
		if err != nil {
			return nil, err
		}
		tokensList = append(tokensList, tokensPage...)
		pageCurrentCount++
	}

	for pageCurrentCount <= pageCount {
		tokensPage, _, err := retrieveEntitlmentTokenListPage(pc, namespace, repository, pageCount, pageSize, showToken, query, activeToken)
		if err != nil {
			return nil, err
		}
		tokensList = append(tokensList, tokensPage...)
		pageCurrentCount++
	}

	return tokensList, nil
}

func flattenEntitlementToken(token []cloudsmith.RepositoryToken) []interface{} {
	tokenList := make([]interface{}, len(token))

	for i, t := range token {
		tokenMap := make(map[string]interface{})
		tokenMap["access_private_broadcasts"] = t.GetAccessPrivateBroadcasts()
		tokenMap["clients"] = t.GetClients()
		tokenMap["created_at"] = t.GetCreatedAt().Format(time.RFC3339)
		tokenMap["created_by"] = t.GetCreatedBy()
		tokenMap["default"] = t.GetDefault()
		tokenMap["downloads"] = t.GetDownloads()
		tokenMap["disable_url"] = t.GetDisableUrl()
		tokenMap["enable_url"] = t.GetEnableUrl()
		tokenMap["eula_required"] = t.GetEulaRequired()
		tokenMap["has_limits"] = t.GetHasLimits()
		tokenMap["identifier"] = t.GetIdentifier()
		tokenMap["is_active"] = t.GetIsActive()
		tokenMap["is_limited"] = t.GetIsLimited()
		tokenMap["limit_bandwidth"] = t.GetLimitBandwidth()
		tokenMap["limit_bandwidth_unit"] = t.GetLimitBandwidthUnit()
		tokenMap["limit_date_range_from"] = t.GetLimitDateRangeFrom().Format(time.RFC3339)
		tokenMap["limit_date_range_to"] = t.GetLimitDateRangeTo().Format(time.RFC3339)
		tokenMap["limit_num_clients"] = t.GetLimitNumClients()
		tokenMap["limit_num_downloads"] = t.GetLimitNumDownloads()
		tokenMap["limit_package_query"] = t.GetLimitPackageQuery()
		tokenMap["limit_path_query"] = t.GetLimitPathQuery()
		tokenMap["metadata"] = t.GetMetadata()
		tokenMap["name"] = t.GetName()
		tokenMap["refresh_url"] = t.GetRefreshUrl()
		tokenMap["reset_url"] = t.GetResetUrl()
		tokenMap["scheduled_reset_at"] = t.GetScheduledResetAt().Format(time.RFC3339)
		tokenMap["scheduled_reset_period"] = t.GetScheduledResetPeriod()
		tokenMap["self_url"] = t.GetSelfUrl()
		tokenMap["slug_perm"] = t.GetSlugPerm()
		tokenMap["token"] = t.GetToken()
		tokenMap["updated_at"] = t.GetUpdatedAt().Format(time.RFC3339)
		tokenMap["updated_by"] = t.GetUpdatedBy()
		tokenMap["updated_by_url"] = t.GetUpdatedByUrl()
		tokenMap["usage"] = t.GetUsage()
		tokenMap["user"] = t.GetUser()
		tokenMap["user_url"] = t.GetUserUrl()

		tokenList[i] = tokenMap
	}

	return tokenList
}

func dataSourceEntitlementRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")
	query := buildQueryString(d.Get("query").(*schema.Set))
	showTokenVal := optionalBool(d, "show_token")
	activeTokenVal := optionalBool(d, "active_token")

	var pageCount, pageSize int64 = -1, -1

	entitlementList, err := retrieveEntitlmentListPages(pc, namespace, repository, query, pageSize, pageCount, *showTokenVal, *activeTokenVal)
	if err != nil {
		return err
	}

	tokens := flattenEntitlementToken(entitlementList)
	if err := d.Set("entitlement_tokens", tokens); err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil

}

func dataSourceEntitlementList() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceEntitlementRead,

		Schema: map[string]*schema.Schema{
			"namespace": {
				Type:        schema.TypeString,
				Description: "The namespace slug.",
				Required:    true,
			},
			"repository": {
				Type:        schema.TypeString,
				Description: "The repository slug.",
				Required:    true,
			},
			"query": {
				Type:        schema.TypeSet,
				Description: "A search term for querying names of entitlements.",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"show_token": {
				Type:        schema.TypeBool,
				Description: "Show entitlement token strings in results.",
				Optional:    true,
				Default:     false,
			},
			"active_token": {
				Type:        schema.TypeBool,
				Description: "If true, only include active tokens",
				Optional:    true,
				Default:     false,
			},
			"entitlement_tokens": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_private_broadcasts": {
							Type:        schema.TypeBool,
							Description: "If enabled, this token can be used for private broadcasts.",
							Computed:    true,
						},
						"clients": {
							Type:        schema.TypeInt,
							Description: "Number of clients associated with the entitlement token.",
							Computed:    true,
						},
						"created_at": {
							Type:        schema.TypeString,
							Description: "The datetime the token was created at.",
							Computed:    true,
						},
						"created_by": {
							Type:        schema.TypeString,
							Description: "The user who created the entitlement token.",
							Computed:    true,
						},
						"default": {
							Type:        schema.TypeBool,
							Description: "If selected this is the default token for this repository.",
							Computed:    true,
						},
						"downloads": {
							Type:        schema.TypeInt,
							Description: "Number of downloads associated with the entitlement token.",
							Computed:    true,
						},
						"disable_url": {
							Type:        schema.TypeString,
							Description: "URL to disable the entitlement token.",
							Computed:    true,
						},
						"enable_url": {
							Type:        schema.TypeString,
							Description: "URL to enable the entitlement token.",
							Computed:    true,
						},
						"eula_required": {
							Type:        schema.TypeBool,
							Description: "If checked, a EULA acceptance is required for this token.",
							Computed:    true,
						},
						"has_limits": {
							Type:        schema.TypeBool,
							Description: "Indicates if there are limits set for the token.",
							Computed:    true,
						},
						"identifier": {
							Type:        schema.TypeInt,
							Description: "A unique identifier for the entitlement token.",
							Computed:    true,
						},
						"is_active": {
							Type:        schema.TypeBool,
							Description: "If enabled, the token will allow downloads based on configured restrictions (if any).",
							Computed:    true,
						},
						"is_limited": {
							Type:        schema.TypeBool,
							Description: "Indicates if the token is limited.",
							Computed:    true,
						},
						"limit_bandwidth": {
							Type:        schema.TypeInt,
							Description: "The maximum download bandwidth allowed for the token.",
							Computed:    true,
						},
						"limit_bandwidth_unit": {
							Type:        schema.TypeString,
							Description: "Unit of bandwidth for the maximum download bandwidth.",
							Computed:    true,
						},
						"limit_date_range_from": {
							Type:        schema.TypeString,
							Description: "The starting date/time the token is allowed to be used from.",
							Computed:    true,
						},
						"limit_date_range_to": {
							Type:        schema.TypeString,
							Description: "The ending date/time the token is allowed to be used until.",
							Computed:    true,
						},
						"limit_num_clients": {
							Type:        schema.TypeInt,
							Description: "The maximum number of unique clients allowed for the token.",
							Computed:    true,
						},
						"limit_num_downloads": {
							Type:        schema.TypeInt,
							Description: "The maximum number of downloads allowed for the token.",
							Computed:    true,
						},
						"limit_package_query": {
							Type:        schema.TypeString,
							Description: "The package-based search query to apply to restrict downloads.",
							Computed:    true,
						},
						"limit_path_query": {
							Type:        schema.TypeString,
							Description: "The path-based search query to apply to restrict downloads.",
							Computed:    true,
						},
						"metadata": {
							Type:        schema.TypeMap,
							Description: "Additional metadata associated with the entitlement token.",
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"name": {
							Type:        schema.TypeString,
							Description: "The name of the entitlement token.",
							Computed:    true,
						},
						"refresh_url": {
							Type:        schema.TypeString,
							Description: "URL to refresh the entitlement token.",
							Computed:    true,
						},
						"reset_url": {
							Type:        schema.TypeString,
							Description: "URL to reset the entitlement token.",
							Computed:    true,
						},
						"scheduled_reset_at": {
							Type:        schema.TypeString,
							Description: "The time at which the scheduled reset period has elapsed and the token limits were automatically reset to zero.",
							Computed:    true,
						},
						"scheduled_reset_period": {
							Type:        schema.TypeString,
							Description: "The period after which the token limits are automatically reset to zero.",
							Computed:    true,
						},
						"self_url": {
							Type:        schema.TypeString,
							Description: "URL for the entitlement token itself.",
							Computed:    true,
						},
						"slug_perm": {
							Type:        schema.TypeString,
							Description: "Slug permission associated with the entitlement token.",
							Computed:    true,
						},
						"token": {
							Type:        schema.TypeString,
							Description: "The entitlement token string.",
							Computed:    true,
							Sensitive:   true,
						},
						"updated_at": {
							Type:        schema.TypeString,
							Description: "The datetime the token was updated at.",
							Computed:    true,
						},
						"updated_by": {
							Type:        schema.TypeString,
							Description: "The user who updated the entitlement token.",
							Computed:    true,
						},
						"updated_by_url": {
							Type:        schema.TypeString,
							Description: "URL for the user who updated the entitlement token.",
							Computed:    true,
						},
						"usage": {
							Type:        schema.TypeString,
							Description: "The usage associated with the token.",
							Computed:    true,
						},
						"user": {
							Type:        schema.TypeString,
							Description: "The user associated with the token.",
							Computed:    true,
						},
						"user_url": {
							Type:        schema.TypeString,
							Description: "URL for the user associated with the token.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
