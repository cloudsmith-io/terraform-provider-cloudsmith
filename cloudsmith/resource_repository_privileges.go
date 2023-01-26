package cloudsmith

import (
	"fmt"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"
)

var (
	repositoryPrivileges = []string{
		"Admin",
		"Write",
		"Read",
	}
)

// expandRepositoryPrivilegeServices extracts "services" from TF state as a *schema.Set and converts to
// a slice of structs we can use when interacting with the Cloudsmith API.
func expandRepositoryPrivilegeServices(d *schema.ResourceData) []cloudsmith.RepositoryPrivilegeDict {
	set := d.Get("service").(*schema.Set)

	return lo.Map(set.List(), func(x interface{}, index int) cloudsmith.RepositoryPrivilegeDict {
		m := x.(map[string]interface{})
		p := cloudsmith.RepositoryPrivilegeDict{}
		p.SetPrivilege(m["privilege"].(string))
		p.SetService(m["slug"].(string))

		return p
	})
}

// expandRepositoryPrivilegeTeams extracts "teams" from TF state as a *schema.Set and converts to
// a slice of structs we can use when interacting with the Cloudsmith API.
func expandRepositoryPrivilegeTeams(d *schema.ResourceData) []cloudsmith.RepositoryPrivilegeDict {
	set := d.Get("team").(*schema.Set)

	return lo.Map(set.List(), func(x interface{}, index int) cloudsmith.RepositoryPrivilegeDict {
		m := x.(map[string]interface{})
		p := cloudsmith.RepositoryPrivilegeDict{}
		p.SetPrivilege(m["privilege"].(string))
		p.SetTeam(m["slug"].(string))

		return p
	})
}

// expandRepositoryPrivilegeUsers extracts "users" from TF state as a *schema.Set and converts to
// a slice of structs we can use when interacting with the Cloudsmith API.
func expandRepositoryPrivilegeUsers(d *schema.ResourceData) []cloudsmith.RepositoryPrivilegeDict {
	set := d.Get("user").(*schema.Set)

	return lo.Map(set.List(), func(x interface{}, index int) cloudsmith.RepositoryPrivilegeDict {
		m := x.(map[string]interface{})
		p := cloudsmith.RepositoryPrivilegeDict{}
		p.SetPrivilege(m["privilege"].(string))
		p.SetUser(m["slug"].(string))

		return p
	})
}

// flattenRepositoryPrivilegeServices takes a slice of
// cloudsmith.RepositoryPrivilegeDict as returned by the Cloudsmith API and
// converts to a *schema.Set that can be stored in TF state.
func flattenRepositoryPrivilegeServices(privileges []cloudsmith.RepositoryPrivilegeDict) *schema.Set {
	serviceSchema := resourceRepositoryPrivileges().Schema["service"].Elem.(*schema.Resource)
	set := schema.NewSet(schema.HashResource(serviceSchema), []interface{}{})

	hasService := func(p cloudsmith.RepositoryPrivilegeDict, index int) bool {
		return p.HasService()
	}

	for _, privilege := range lo.Filter(privileges, hasService) {
		set.Add(map[string]interface{}{
			"privilege": privilege.GetPrivilege(),
			"slug":      privilege.GetService(),
		})
	}
	return set
}

// flattenRepositoryPrivilegeTeams takes a slice of
// cloudsmith.RepositoryPrivilegeDict as returned by the Cloudsmith API and
// converts to a *schema.Set that can be stored in TF state.
func flattenRepositoryPrivilegeTeams(privileges []cloudsmith.RepositoryPrivilegeDict) *schema.Set {
	teamSchema := resourceRepositoryPrivileges().Schema["team"].Elem.(*schema.Resource)
	set := schema.NewSet(schema.HashResource(teamSchema), []interface{}{})

	hasTeam := func(p cloudsmith.RepositoryPrivilegeDict, index int) bool {
		return p.HasTeam()
	}

	for _, privilege := range lo.Filter(privileges, hasTeam) {
		set.Add(map[string]interface{}{
			"privilege": privilege.GetPrivilege(),
			"slug":      privilege.GetTeam(),
		})
	}
	return set
}

// flattenRepositoryPrivilegeUsers takes a slice of
// cloudsmith.RepositoryPrivilegeDict as returned by the Cloudsmith API and
// converts to a *schema.Set that can be stored in TF state.
func flattenRepositoryPrivilegeUsers(privileges []cloudsmith.RepositoryPrivilegeDict) *schema.Set {
	userSchema := resourceRepositoryPrivileges().Schema["user"].Elem.(*schema.Resource)
	set := schema.NewSet(schema.HashResource(userSchema), []interface{}{})

	hasUser := func(p cloudsmith.RepositoryPrivilegeDict, index int) bool {
		return p.HasUser()
	}

	for _, privilege := range lo.Filter(privileges, hasUser) {
		set.Add(map[string]interface{}{
			"privilege": privilege.GetPrivilege(),
			"slug":      privilege.GetUser(),
		})
	}
	return set
}

func resourceRepositoryPrivilegesCreateUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")
	repository := requiredString(d, "repository")

	privileges := []cloudsmith.RepositoryPrivilegeDict{}
	privileges = append(privileges, expandRepositoryPrivilegeServices(d)...)
	privileges = append(privileges, expandRepositoryPrivilegeTeams(d)...)
	privileges = append(privileges, expandRepositoryPrivilegeUsers(d)...)

	req := pc.APIClient.ReposApi.ReposPrivilegesUpdate(pc.Auth, organization, repository)
	req = req.Data(cloudsmith.RepositoryPrivilegeInputRequest{
		Privileges: privileges,
	})

	_, err := pc.APIClient.ReposApi.ReposPrivilegesUpdateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%s/%s", organization, repository))

	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for
		// repository privileges being updated (changes incoming on the API side)
		time.Sleep(time.Second * 5)
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for privileges (%s) to be updated: %w", d.Id(), err)
	}

	return resourceRepositoryPrivilegesRead(d, m)
}

func resourceRepositoryPrivilegesRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")
	repository := requiredString(d, "repository")

	req := pc.APIClient.ReposApi.ReposPrivilegesList(pc.Auth, organization, repository)

	// TODO: add a proper loop here to ensure we always get all privs,
	// regardless of how many are configured.
	req = req.Page(1)
	req = req.PageSize(1000)

	privileges, resp, err := pc.APIClient.ReposApi.ReposPrivilegesListExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("service", flattenRepositoryPrivilegeServices(privileges.GetPrivileges()))
	d.Set("team", flattenRepositoryPrivilegeTeams(privileges.GetPrivileges()))
	d.Set("user", flattenRepositoryPrivilegeUsers(privileges.GetPrivileges()))

	// namespace and repository are not returned from the privileges read
	// endpoint, so we can use the values stored in resource state. We rely on
	// ForceNew to ensure if either changes a new resource is created.
	d.Set("organization", organization)
	d.Set("repository", repository)

	return nil
}

func resourceRepositoryPrivilegesDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")
	repository := requiredString(d, "repository")

	req := pc.APIClient.ReposApi.ReposPrivilegesUpdate(pc.Auth, organization, repository)
	req = req.Data(cloudsmith.RepositoryPrivilegeInputRequest{
		Privileges: []cloudsmith.RepositoryPrivilegeDict{},
	})

	_, err := pc.APIClient.ReposApi.ReposPrivilegesUpdateExecute(req)
	if err != nil {
		return err
	}

	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for
		// repository privileges being deleted (changes incoming on the API side)
		time.Sleep(time.Second * 5)
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for privileges (%s) to be deleted: %w", d.Id(), err)
	}

	return nil
}

//nolint:funlen
func resourceRepositoryPrivileges() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryPrivilegesCreateUpdate,
		Read:   resourceRepositoryPrivilegesRead,
		Update: resourceRepositoryPrivilegesCreateUpdate,
		Delete: resourceRepositoryPrivilegesDelete,

		Schema: map[string]*schema.Schema{
			"organization": {
				Type:         schema.TypeString,
				Description:  "Organization to which this repository belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"repository": {
				Type:         schema.TypeString,
				Description:  "Repository to which these privileges belong.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"service": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"privilege": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(repositoryPrivileges, false),
						},
						"slug": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Optional: true,
			},
			"team": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"privilege": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(repositoryPrivileges, false),
						},
						"slug": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Optional: true,
			},
			"user": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"privilege": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(repositoryPrivileges, false),
						},
						"slug": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Optional: true,
			},
		},
	}
}
