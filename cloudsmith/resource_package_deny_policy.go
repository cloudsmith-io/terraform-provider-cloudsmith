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
		PackageQueryString: nullableString(d, "package_query"),
	})
	packageDenyPolicy, _, err := pc.APIClient.OrgsApi.OrgsDenyPolicyCreateExecute(req)
	if err != nil {
		return err
	}
	d.SetId(packageDenyPolicy.GetSlugPerm())
	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsDenyPolicyRead(pc.Auth, namespace, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsDenyPolicyReadExecute(req); err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return err
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for package deny policy (%s) to be created: %w", d.Id(), err)
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
		PackageQueryString: nullableString(d, "package_query"),
	})
	packageDenyPolicy, _, err := pc.APIClient.OrgsApi.OrgsDenyPolicyPartialUpdateExecute(req)
	if err != nil {
		return err
	}
	d.SetId(packageDenyPolicy.GetSlugPerm())
	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for a
		// deny policy being updated
		time.Sleep(time.Second * 5)
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for deny policy (%s) to be updated: %w", d.Id(), err)
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

	checkerFunc := func() error {
		req := pc.APIClient.OrgsApi.OrgsDenyPolicyRead(pc.Auth, namespace, d.Id())
		if _, resp, err := pc.APIClient.OrgsApi.OrgsDenyPolicyReadExecute(req); err != nil {
			if is404(resp) {
				return nil
			}
			return err
		}
		return errKeepWaiting
	}

	if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
		return fmt.Errorf("error waiting for deny policy (%s) to be deleted: %w", d.Id(), err)
	}
	return nil
}

//nolint:funlen
func packageDenyPolicy() *schema.Resource {
	return &schema.Resource{
		Create: packageDenyPolicyCreate,
		Read:   packageDenyPolicyRead,
		Update: packageDenyPolicyUpdate,
		Delete: packageDenyPolicyDelete,

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
