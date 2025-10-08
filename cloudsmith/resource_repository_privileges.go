package cloudsmith

import (
	"context"
	"fmt"
	"log"
	"strings"
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

// containsAccountSlug returns true if any privilege entry contains the provided slug
// either as a user or service.
func containsAccountSlug(privs []cloudsmith.RepositoryPrivilegeDict, slug string) bool {
	for _, p := range privs {
		if p.HasUser() && p.GetUser() == slug {
			return true
		}
		if p.HasService() && p.GetService() == slug {
			return true
		}
	}
	return false
}

// containsTeam returns true if any privilege entry references a team.
func containsTeam(privs []cloudsmith.RepositoryPrivilegeDict) bool {
	for _, p := range privs {
		if p.HasTeam() {
			return true
		}
	}
	return false
}

// setContainsSlug returns true if the *schema.Set contains an element whose slug matches key.
func setContainsSlug(set *schema.Set, key string) bool {
	if set == nil {
		return false
	}
	for _, x := range set.List() {
		m := x.(map[string]interface{})
		if m["slug"].(string) == key {
			return true
		}
	}
	return false
}

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

func importRepositoryPrivileges(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ".")
	if len(idParts) != 2 {
		return nil, fmt.Errorf(
			"invalid import ID, must be of the form <organization_slug>.<repository_slug>, got: %s", d.Id(),
		)
	}

	d.Set("organization", idParts[0])
	d.Set("repository", idParts[1])
	return []*schema.ResourceData{d}, nil
}

func resourceRepositoryPrivilegesCreateUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	organization := requiredString(d, "organization")
	repository := requiredString(d, "repository")

	privileges := []cloudsmith.RepositoryPrivilegeDict{}
	privileges = append(privileges, expandRepositoryPrivilegeServices(d)...)
	privileges = append(privileges, expandRepositoryPrivilegeTeams(d)...)
	privileges = append(privileges, expandRepositoryPrivilegeUsers(d)...)

	// Only hard error if the authenticated account is NOT present in any user/service block
	// AND there are NO team blocks defined. If team blocks are present, emit a warning only.
	userReq := pc.APIClient.UserApi.UserSelf(pc.Auth)
	userSelf, _, err := pc.APIClient.UserApi.UserSelfExecute(userReq)
	if err != nil {
		return fmt.Errorf("error retrieving authenticated account for lockout prevention: %w", err)
	}
	currentSlug := userSelf.GetSlug()

	if !containsAccountSlug(privileges, currentSlug) {
		if !containsTeam(privileges) {
			return fmt.Errorf(
				"repository_privileges (%s.%s): configuration must include authenticated account slug '%s' (user or service block) OR at least one team block to avoid potential lockout",
				organization, repository, currentSlug,
			)
		}
		log.Printf("[WARN] repository_privileges (%s.%s): authenticated account slug '%s' not explicitly included via user/service; ensure access via configured teams to avoid lockout.", organization, repository, currentSlug)
	}

	req := pc.APIClient.ReposApi.ReposPrivilegesUpdate(pc.Auth, organization, repository)
	req = req.Data(cloudsmith.RepositoryPrivilegeInputRequest{
		Privileges: privileges,
	})

	_, err = pc.APIClient.ReposApi.ReposPrivilegesUpdateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%s.%s", organization, repository))

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

	var allPrivileges []cloudsmith.RepositoryPrivilegeDict
	page := int64(1)
	pageSize := int64(1000)

	for {
		req := pc.APIClient.ReposApi.ReposPrivilegesList(pc.Auth, organization, repository)
		req = req.Page(page)
		req = req.PageSize(pageSize)
		privileges, resp, err := pc.APIClient.ReposApi.ReposPrivilegesListExecute(req)
		if err != nil {
			if is404(resp) {
				d.SetId("")
				return nil
			}
			return err
		}

		allPrivileges = append(allPrivileges, privileges.GetPrivileges()...)

		// Check if we have retrieved all pages
		if int64(len(privileges.GetPrivileges())) < pageSize {
			break
		}
		page++
	}

	d.Set("service", flattenRepositoryPrivilegeServices(allPrivileges))
	d.Set("team", flattenRepositoryPrivilegeTeams(allPrivileges))
	d.Set("user", flattenRepositoryPrivilegeUsers(allPrivileges))

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

		// Plan-time validation to surface lockout risk earlier than apply. We still
		// keep the apply-time safety net in Create/Update for defense in depth.
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			pc := meta.(*providerConfig)
			userReq := pc.APIClient.UserApi.UserSelf(pc.Auth)
			userSelf, _, err := pc.APIClient.UserApi.UserSelfExecute(userReq)
			if err != nil {
				// If we cannot determine the current user, defer to apply-time logic.
				return nil
			}
			currentSlug := userSelf.GetSlug()

			var userSet *schema.Set
			if v, ok := d.GetOk("user"); ok {
				userSet = v.(*schema.Set)
			}
			var serviceSet *schema.Set
			if v, ok := d.GetOk("service"); ok {
				serviceSet = v.(*schema.Set)
			}
			var teamSet *schema.Set
			if v, ok := d.GetOk("team"); ok {
				teamSet = v.(*schema.Set)
			}

			hasUserOrService := setContainsSlug(userSet, currentSlug) || setContainsSlug(serviceSet, currentSlug)
			teamCount := 0
			if teamSet != nil {
				teamCount = teamSet.Len()
			}

			if !hasUserOrService {
				if teamCount == 0 {
					return fmt.Errorf("repository_privileges: authenticated account slug '%s' must be included (user or service block) OR at least one team block must be defined to avoid potential lockout", currentSlug)
				}
				log.Printf("[WARN] repository_privileges (plan): authenticated account slug '%s' not explicitly included via user/service; ensure team-based access is sufficient to avoid lockout.", currentSlug)
			}

			return nil
		},

		Importer: &schema.ResourceImporter{
			StateContext: importRepositoryPrivileges,
		},

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
