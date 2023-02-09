package cloudsmith

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const Namespace string = "namespace"
const Repository string = "repository"
const CidrAllow string = "cidr_allow"
const CidrDeny string = "cidr_deny"
const CountryCodeAllow string = "country_code_allow"
const CountryCodeDeny string = "country_code_deny"

func importRepositoryGeoIpRules(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<repository_slug>, got: %s", d.Id(),
		)
	}

	d.Set("namespace", idParts[0])
	d.Set("repository", idParts[1])
	return []*schema.ResourceData{d}, nil
}

func resourceRepositoryGeoIpRulesCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)

	// Ensure that Geo/IP rules are enabled for the Repository
	req := pc.APIClient.ReposApi.ReposGeoipEnable(pc.Auth, namespace, repository)
	_, err := pc.APIClient.ReposApi.ReposGeoipEnableExecute(req)
	if err != nil {
		return err
	}

	// The actual "create" is just the same as "update" for this resource.
	return resourceRepositoryGeoIpRulesUpdate(d, m)
}

func resourceRepositoryGeoIpRulesRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)

	req := pc.APIClient.ReposApi.ReposGeoipRead(pc.Auth, namespace, repository)

	geoIpRules, resp, err := pc.APIClient.ReposApi.ReposGeoipReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return err
	}

	cidr := geoIpRules.GetCidr()
	countryCode := geoIpRules.GetCountryCode()

	_ = d.Set(CidrAllow, flattenStrings(cidr.GetAllow()))
	_ = d.Set(CidrDeny, flattenStrings(cidr.GetDeny()))
	_ = d.Set(CountryCodeAllow, flattenStrings(countryCode.GetAllow()))
	_ = d.Set(CountryCodeDeny, flattenStrings(countryCode.GetDeny()))

	// namespace and repository are not returned from the read
	// endpoint, so we can use the values stored in resource state. We rely on
	// ForceNew to ensure if either changes a new resource is created.
	_ = d.Set(Namespace, namespace)
	_ = d.Set(Repository, repository)

	return nil
}

func resourceRepositoryGeoIpRulesUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, Namespace)
	repository := requiredString(d, Repository)

	updateData := cloudsmith.RepositoryGeoIpRules{
		CountryCode: &cloudsmith.RepositoryGeoIpCountryCodeRules{
			Allow: expandStrings(d, CountryCodeAllow),
			Deny:  expandStrings(d, CountryCodeDeny),
		},
		Cidr: &cloudsmith.RepositoryGeoIpCidrRules{
			Allow: expandStrings(d, CidrAllow),
			Deny:  expandStrings(d, CidrDeny),
		},
	}

	updateRequest := pc.APIClient.ReposApi.ReposGeoipUpdate(pc.Auth, namespace, repository)
	updateRequest = updateRequest.Data(updateData)

	_, updateErr := pc.APIClient.ReposApi.ReposGeoipUpdateExecute(updateRequest)
	if updateErr != nil {
		return updateErr
	}

	d.SetId(fmt.Sprintf("%s.%s", namespace, repository))

	// Workaround for replication lag
	checkerFunc := func() error {
		// Call the read endpoint
		readRequest := pc.APIClient.ReposApi.ReposGeoipRead(pc.Auth, namespace, repository)
		readData, _, readErr := pc.APIClient.ReposApi.ReposGeoipReadExecute(readRequest)
		if readErr != nil {
			return readErr
		}

		// Check that the read response data matches our earlier update request data
		readCidr := readData.GetCidr()
		updateCidr := updateData.GetCidr()

		if !stringSlicesAreEqual(readCidr.GetAllow(), updateCidr.GetAllow(), true) {
			return errKeepWaiting
		}
		if !stringSlicesAreEqual(readCidr.GetDeny(), updateCidr.GetDeny(), true) {
			return errKeepWaiting
		}

		readCountryCode := readData.GetCountryCode()
		updateCountryCode := updateData.GetCountryCode()

		if !stringSlicesAreEqual(readCountryCode.GetAllow(), updateCountryCode.GetAllow(), true) {
			return errKeepWaiting
		}
		if !stringSlicesAreEqual(readCountryCode.GetDeny(), updateCountryCode.GetDeny(), true) {
			return errKeepWaiting
		}

		return nil
	}

	waitErr := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval)
	if waitErr != nil {
		return waitErr
	}

	return resourceRepositoryGeoIpRulesRead(d, m)
}

func resourceRepositoryGeoIpRulesDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	// There isn't a DELETE endpoint, so just update the rules to be empty.
	req := pc.APIClient.ReposApi.ReposGeoipUpdate(pc.Auth, namespace, repository)
	req = req.Data(cloudsmith.RepositoryGeoIpRules{
		CountryCode: &cloudsmith.RepositoryGeoIpCountryCodeRules{
			Allow: []string{},
			Deny:  []string{},
		},
		Cidr: &cloudsmith.RepositoryGeoIpCidrRules{
			Allow: []string{},
			Deny:  []string{},
		},
	})
	_, err := pc.APIClient.ReposApi.ReposGeoipUpdateExecute(req)
	if err != nil {
		return err
	}

	return nil
}

//nolint:funlen
func resourceRepositoryGeoIpRules() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryGeoIpRulesCreate,
		Read:   resourceRepositoryGeoIpRulesRead,
		Update: resourceRepositoryGeoIpRulesUpdate,
		Delete: resourceRepositoryGeoIpRulesDelete,

		Importer: &schema.ResourceImporter{
			StateContext: importRepositoryGeoIpRules,
		},

		Schema: map[string]*schema.Schema{
			CidrAllow: {
				Type:        schema.TypeSet,
				Description: "The list of IP Addresses for which to allow access, expressed in CIDR notation.",
				Optional:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
			CidrDeny: {
				Type:        schema.TypeSet,
				Description: "The list of IP Addresses for which to deny access, expressed in CIDR notation.",
				Optional:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
			CountryCodeAllow: {
				Type:        schema.TypeSet,
				Description: "The list of countries for which to allow access, expressed in ISO 3166-1 country codes.",
				Optional:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
			CountryCodeDeny: {
				Type:        schema.TypeSet,
				Description: "The list of countries for which to deny access, expressed in ISO 3166-1 country codes.",
				Optional:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
			Namespace: {
				Type:         schema.TypeString,
				Description:  "Organization to which the Repository belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			Repository: {
				Type:         schema.TypeString,
				Description:  "Repository to which these Geo/IP rules belong.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}
