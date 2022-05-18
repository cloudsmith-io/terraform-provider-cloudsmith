package cloudsmith

import (
	"github.com/antihax/optional"
	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func getIsActiveFromResource(d *schema.ResourceData) *bool {
	var isActivePtr *bool

	isActive, exists := d.GetOkExists("is_active") //nolint:staticcheck
	if exists {
		isActiveBool := isActive.(bool)
		isActivePtr = &isActiveBool
	}

	return isActivePtr
}

func resourceEntitlementCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	isActive := getIsActiveFromResource(d)
	limitDateRangeFrom := d.Get("limit_date_range_from").(string)
	limitDateRangeTo := d.Get("limit_date_range_to").(string)
	limitNumClients := int64(d.Get("limit_num_clients").(int))
	limitNumDownloads := int64(d.Get("limit_num_downloads").(int))
	limitPackageQuery := d.Get("limit_package_query").(string)
	limitPathQuery := d.Get("limit_path_query").(string)
	namespace := d.Get("namespace").(string)
	repository := d.Get("repository").(string)
	name := d.Get("name").(string)
	token := d.Get("token").(string)

	opts := &cloudsmith.EntitlementsCreateOpts{
		Data: optional.NewInterface(cloudsmith.EntitlementsCreate{
			IsActive:           isActive,
			LimitDateRangeFrom: limitDateRangeFrom,
			LimitDateRangeTo:   limitDateRangeTo,
			LimitNumClients:    limitNumClients,
			LimitNumDownloads:  limitNumDownloads,
			LimitPackageQuery:  limitPackageQuery,
			LimitPathQuery:     limitPathQuery,
			Name:               name,
			Token:              token,
		}),
		ShowTokens: optional.NewBool(true),
	}

	entitlement, _, err := pc.APIClient.EntitlementsApi.EntitlementsCreate(pc.Auth, namespace, repository, opts)
	if err != nil {
		return err
	}

	d.SetId(entitlement.SlugPerm)

	return resourceEntitlementRead(d, m)
}

func resourceEntitlementRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := d.Get("namespace").(string)
	repository := d.Get("repository").(string)

	opts := &cloudsmith.EntitlementsReadOpts{
		ShowTokens: optional.NewBool(true),
	}

	entitlement, _, err := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, d.Id(), opts)
	if err != nil {
		if err.Error() == errMessage404 {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("is_active", entitlement.IsActive)
	d.Set("limit_date_range_from", entitlement.LimitDateRangeFrom)
	d.Set("limit_date_range_to", entitlement.LimitDateRangeTo)
	d.Set("limit_num_clients", entitlement.LimitNumClients)
	d.Set("limit_num_downloads", entitlement.LimitNumDownloads)
	d.Set("limit_package_query", entitlement.LimitPackageQuery)
	d.Set("limit_path_query", entitlement.LimitPathQuery)
	d.Set("name", entitlement.Name)
	d.Set("token", entitlement.Token)

	// namespace and repository are not returned from the entitlement read
	// endpoint, so we can use the values stored in resource state. We rely on
	// ForceNew to ensure if either changes a new resource is created.
	d.Set("namespace", namespace)
	d.Set("repository", repository)

	return nil
}

func resourceEntitlementUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	isActive := getIsActiveFromResource(d)
	limitDateRangeFrom := d.Get("limit_date_range_from").(string)
	limitDateRangeTo := d.Get("limit_date_range_to").(string)
	limitNumClients := int64(d.Get("limit_num_clients").(int))
	limitNumDownloads := int64(d.Get("limit_num_downloads").(int))
	limitPackageQuery := d.Get("limit_package_query").(string)
	limitPathQuery := d.Get("limit_path_query").(string)
	namespace := d.Get("namespace").(string)
	repository := d.Get("repository").(string)
	name := d.Get("name").(string)
	token := d.Get("token").(string)

	opts := &cloudsmith.EntitlementsPartialUpdateOpts{
		Data: optional.NewInterface(cloudsmith.EntitlementsPartialUpdate{
			IsActive:           isActive,
			LimitDateRangeFrom: limitDateRangeFrom,
			LimitDateRangeTo:   limitDateRangeTo,
			LimitNumClients:    limitNumClients,
			LimitNumDownloads:  limitNumDownloads,
			LimitPackageQuery:  limitPackageQuery,
			LimitPathQuery:     limitPathQuery,
			Name:               name,
			Token:              token,
		}),
		ShowTokens: optional.NewBool(true),
	}

	entitlement, _, err := pc.APIClient.EntitlementsApi.EntitlementsPartialUpdate(pc.Auth, namespace, repository, d.Id(), opts)
	if err != nil {
		return err
	}

	d.SetId(entitlement.SlugPerm)

	return resourceEntitlementRead(d, m)
}

func resourceEntitlementDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := d.Get("namespace").(string)
	repository := d.Get("repository").(string)

	_, err := pc.APIClient.EntitlementsApi.EntitlementsDelete(pc.Auth, namespace, repository, d.Id())
	if err != nil {
		return err
	}

	return nil
}

//nolint:funlen
func resourceEntitlement() *schema.Resource {
	return &schema.Resource{
		Create: resourceEntitlementCreate,
		Read:   resourceEntitlementRead,
		Update: resourceEntitlementUpdate,
		Delete: resourceEntitlementDelete,

		Schema: map[string]*schema.Schema{
			"is_active": {
				Type:        schema.TypeBool,
				Description: "If enabled, the token will allow downloads based on configured restrictions (if any).",
				Optional:    true,
				Computed:    true,
			},
			"limit_date_range_from": {
				Type:        schema.TypeString,
				Description: "The starting date/time the token is allowed to be used from.",
				Optional:    true,
			},
			"limit_date_range_to": {
				Type:        schema.TypeString,
				Description: "The ending date/time the token is allowed to be used until.",
				Optional:    true,
			},
			"limit_num_clients": {
				Type: schema.TypeInt,
				Description: "The maximum number of unique clients allowed for the token. Please " +
					"note that since clients are calculated asynchronously (after the download " +
					"happens), the limit may not be imposed immediately but at a later point.",
				Optional: true,
				Computed: true,
			},
			"limit_num_downloads": {
				Type: schema.TypeInt,
				Description: "The maximum number of downloads allowed for the token. Please note " +
					"that since downloads are calculated asynchronously (after the download " +
					"happens), the limit may not be imposed immediately but at a later point.",
				Optional: true,
				Computed: true,
			},
			"limit_package_query": {
				Type: schema.TypeString,
				Description: "The package-based search query to apply to restrict downloads to. " +
					"This uses the same syntax as the standard search used for repositories, and " +
					"also supports boolean logic operators such as OR/AND/NOT and parentheses for " +
					"grouping. This will still allow access to non-package files, such as metadata.",
				Optional: true,
			},
			"limit_path_query": {
				Type: schema.TypeString,
				Description: "The path-based search query to apply to restrict downloads to. This " +
					"supports boolean logic operators such as OR/AND/NOT and parentheses for " +
					"grouping. The path evaluated does not include the domain name, the namespace, " +
					"the entitlement code used, the package format, etc. and it always starts with " +
					"a forward slash.",
				Optional: true,
			},
			"name": {
				Type:         schema.TypeString,
				Description:  "A descriptive name for the entitlement.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "Namespace to which this entitlement belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"repository": {
				Type:         schema.TypeString,
				Description:  "Repository to which this entitlement belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"token": {
				Type:         schema.TypeString,
				Description:  "The literal value of the token to be created.",
				Optional:     true,
				Computed:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}
