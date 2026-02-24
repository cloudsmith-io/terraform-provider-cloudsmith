package cloudsmith

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func importEntitlement(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 3 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<repository_slug>.<entitlement_slug>, got: %s", d.Id(),
		)
	}

	d.Set("namespace", idParts[0])
	d.Set("repository", idParts[1])
	d.SetId(idParts[2])
	return []*schema.ResourceData{d}, nil
}

func resourceEntitlementCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	req := pc.APIClient.EntitlementsApi.EntitlementsCreate(pc.Auth, namespace, repository)
	req = req.Data(cloudsmith.RepositoryTokenRequest{
		AccessPrivateBroadcasts: optionalBool(d, "access_private_broadcasts"),
		IsActive:                optionalBool(d, "is_active"),
		LimitDateRangeFrom:      nullableTime(d, "limit_date_range_from"),
		LimitDateRangeTo:        nullableTime(d, "limit_date_range_to"),
		LimitNumClients:         nullableInt64(d, "limit_num_clients"),
		LimitNumDownloads:       nullableInt64(d, "limit_num_downloads"),
		LimitPackageQuery:       nullableString(d, "limit_package_query"),
		LimitPathQuery:          nullableString(d, "limitPathQuery"),
		Name:                    requiredString(d, "name"),
		Token:                   optionalString(d, "token"),
	})
	req = req.ShowTokens(true)

	entitlement, _, err := pc.APIClient.EntitlementsApi.EntitlementsCreateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(entitlement.GetSlugPerm())

	checkerFunc := func() error {
		req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, d.Id())
		if _, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req); err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return err
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for entitlement (%s) to be created: %w", d.Id(), err)
	}

	return resourceEntitlementRead(d, m)
}

func resourceEntitlementRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, d.Id())
	req = req.ShowTokens(true)

	entitlement, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("access_private_broadcasts", entitlement.GetAccessPrivateBroadcasts())
	d.Set("is_active", entitlement.GetIsActive())
	d.Set("limit_date_range_from", timeToString(entitlement.GetLimitDateRangeFrom()))
	d.Set("limit_date_range_to", timeToString(entitlement.GetLimitDateRangeTo()))
	d.Set("limit_num_clients", entitlement.GetLimitNumClients())
	d.Set("limit_num_downloads", entitlement.GetLimitNumDownloads())
	d.Set("limit_package_query", entitlement.GetLimitPackageQuery())
	d.Set("limit_path_query", entitlement.GetLimitPathQuery())
	d.Set("name", entitlement.GetName())
	d.Set("token", entitlement.GetToken())
	d.Set("slug_perm", entitlement.GetSlugPerm())

	// namespace and repository are not returned from the entitlement read
	// endpoint, so we can use the values stored in resource state. We rely on
	// ForceNew to ensure if either changes a new resource is created.
	d.Set("namespace", namespace)
	d.Set("repository", repository)

	return nil
}

func resourceEntitlementUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	req := pc.APIClient.EntitlementsApi.EntitlementsPartialUpdate(pc.Auth, namespace, repository, d.Id())
	req = req.Data(cloudsmith.RepositoryTokenRequestPatch{
		AccessPrivateBroadcasts: optionalBool(d, "access_private_broadcasts"),
		IsActive:                optionalBool(d, "is_active"),
		LimitDateRangeFrom:      nullableTime(d, "limit_date_range_from"),
		LimitDateRangeTo:        nullableTime(d, "limit_date_range_to"),
		LimitNumClients:         nullableInt64(d, "limit_num_clients"),
		LimitNumDownloads:       nullableInt64(d, "limit_num_downloads"),
		LimitPackageQuery:       nullableString(d, "limit_package_query"),
		LimitPathQuery:          nullableString(d, "limit_path_query"),
		Name:                    optionalString(d, "name"),
		Token:                   optionalString(d, "token"),
	})
	req = req.ShowTokens(true)

	entitlement, _, err := pc.APIClient.EntitlementsApi.EntitlementsPartialUpdateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(entitlement.GetSlugPerm())

	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for an
		// entitlement being updated (changes incoming on the API side)
		time.Sleep(time.Second * 5)
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for entitlement (%s) to be updated: %w", d.Id(), err)
	}

	return resourceEntitlementRead(d, m)
}

func resourceEntitlementDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	req := pc.APIClient.EntitlementsApi.EntitlementsDelete(pc.Auth, namespace, repository, d.Id())
	_, err := pc.APIClient.EntitlementsApi.EntitlementsDeleteExecute(req)
	if err != nil {
		return err
	}

	checkerFunc := func() error {
		req := pc.APIClient.EntitlementsApi.EntitlementsRead(pc.Auth, namespace, repository, d.Id())
		if _, resp, err := pc.APIClient.EntitlementsApi.EntitlementsReadExecute(req); err != nil {
			if is404(resp) {
				return nil
			}
			return err
		}
		return errKeepWaiting
	}
	if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
		return fmt.Errorf("error waiting for entitlement (%s) to be deleted: %w", d.Id(), err)
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

		Importer: &schema.ResourceImporter{
			StateContext: importEntitlement,
		},

		Schema: map[string]*schema.Schema{
			"access_private_broadcasts": {
				Type:        schema.TypeBool,
				Description: "If enabled, this token can be used for private broadcasts.",
				Optional:    true,
				Computed:    true,
			},
			"is_active": {
				Type:        schema.TypeBool,
				Description: "If enabled, the token will allow downloads based on configured restrictions (if any).",
				Optional:    true,
				Computed:    true,
			},
			"limit_date_range_from": {
				Type:         schema.TypeString,
				Description:  "The starting date/time the token is allowed to be used from.",
				Optional:     true,
				ValidateFunc: validation.IsRFC3339Time,
			},
			"limit_date_range_to": {
				Type:         schema.TypeString,
				Description:  "The ending date/time the token is allowed to be used until.",
				Optional:     true,
				ValidateFunc: validation.IsRFC3339Time,
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
			"slug_perm": {
				Type:        schema.TypeString,
				Description: "The permanent slug identifier for the entitlement.",
				Computed:    true,
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
