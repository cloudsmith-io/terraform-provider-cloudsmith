package cloudsmith

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func packageDenyPolicyImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<package_deny_policy_slug>, got: %s", d.Id(),
		)
	}

	d.Set("namespace", idParts[0])
	d.SetId(idParts[1])
	return []*schema.ResourceData{d}, nil
}

func packageDenyPolicyCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	req := pc.APIClient.OrgsApi.OrgsDenyPolicyCreate(pc.Auth, namespace)
	req = req.Data(cloudsmith.PackageDenyPolicyRequest{
		Name:               nullableString(d, "name"),
		Enabled:            optionalBool(d, "enabled"),
		Description:        nullableString(d, "description"),
		PackageQueryString: *optionalString(d, "package_query"),
	})
	packageDenyPolicy, _, err := pc.APIClient.OrgsApi.OrgsDenyPolicyCreateExecute(req)
	if err != nil {
		return err
	}
	d.SetId(packageDenyPolicy.GetSlugPerm())
	if err := waitForCreation(func() (*http.Response, error) {
		req := pc.APIClient.OrgsApi.OrgsDenyPolicyRead(pc.Auth, namespace, d.Id())
		_, resp, err := pc.APIClient.OrgsApi.OrgsDenyPolicyReadExecute(req)
		return resp, err
	}, "package deny policy", d.Id()); err != nil {
		return err
	}
	return packageDenyPolicyRead(d, m)
}

func packageDenyPolicyRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	req := pc.APIClient.OrgsApi.OrgsDenyPolicyRead(pc.Auth, namespace, d.Id())
	packageDenyPolicy, resp, err := pc.APIClient.OrgsApi.OrgsDenyPolicyReadExecute(req)

	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}
		return err

	}

	d.Set("name", packageDenyPolicy.GetName())
	d.Set("description", packageDenyPolicy.GetDescription())
	d.Set("package_query", packageDenyPolicy.GetPackageQueryString())
	d.Set("enabled", packageDenyPolicy.GetEnabled())

	return nil
}

func packageDenyPolicyUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)
	namespace := requiredString(d, "namespace")
	req := pc.APIClient.OrgsApi.OrgsDenyPolicyPartialUpdate(pc.Auth, namespace, d.Id())
	req = req.Data(cloudsmith.PackageDenyPolicyRequestPatch{
		Name:               nullableString(d, "name"),
		Enabled:            optionalBool(d, "enabled"),
		Description:        nullableString(d, "description"),
		PackageQueryString: optionalString(d, "package_query"),
	})
	packageDenyPolicy, _, err := pc.APIClient.OrgsApi.OrgsDenyPolicyPartialUpdateExecute(req)
	if err != nil {
		return err
	}
	d.SetId(packageDenyPolicy.GetSlugPerm())
	if err := waitForUpdate("deny policy", d.Id()); err != nil {
		return err
	}
	return packageDenyPolicyRead(d, m)
}

func packageDenyPolicyDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")

	req := pc.APIClient.OrgsApi.OrgsDenyPolicyDelete(pc.Auth, namespace, d.Id())
	_, err := pc.APIClient.OrgsApi.OrgsDenyPolicyDeleteExecute(req)
	if err != nil {
		return err
	}

	if err := waitForDeletion(func() (*http.Response, error) {
		req := pc.APIClient.OrgsApi.OrgsDenyPolicyRead(pc.Auth, namespace, d.Id())
		_, resp, err := pc.APIClient.OrgsApi.OrgsDenyPolicyReadExecute(req)
		return resp, err
	}, "deny policy", d.Id()); err != nil {
		return err
	}
	return nil
}

//nolint:funlen
func packageDenyPolicy() *schema.Resource {
	return &schema.Resource{
		Create:      packageDenyPolicyCreate,
		Read:        packageDenyPolicyRead,
		Update:      packageDenyPolicyUpdate,
		Delete:      packageDenyPolicyDelete,
		Description: "Package deny policies control which packages can be downloaded within their repositories.",

		Importer: &schema.ResourceImporter{
			StateContext: packageDenyPolicyImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "A descriptive name for the package deny policy.",
				Optional:    true,
				Default:     nil,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "Description of the package deny policy.",
				Optional:    true,
				Default:     nil,
			},
			"package_query": {
				Type:         schema.TypeString,
				Description:  "The query to match the packages to be blocked.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"enabled": {
				Type:        schema.TypeBool,
				Description: "Is the package deny policy enabled?.",
				Optional:    true,
				Default:     true,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "Namespace to which this package deny policy belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}
