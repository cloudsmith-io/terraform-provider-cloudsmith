// Package cloudsmith provides Terraform provider functionality for managing Cloudsmith resources.
package cloudsmith

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func samlAuthRequestContext(ctx context.Context, pc *providerConfig) context.Context {
	if pc.Auth != nil {
		if keys := pc.Auth.Value(cloudsmith.ContextAPIKeys); keys != nil {
			return context.WithValue(ctx, cloudsmith.ContextAPIKeys, keys)
		}
	}
	return ctx
}

// waitForSAMLAuthState returns the observed snapshot that matches the expected values.
func waitForSAMLAuthState(ctx context.Context, pc *providerConfig, organization string, wantEnabled bool, wantEnforced bool, wantInline string, wantURL string, timeout time.Duration) (*cloudsmith.OrganizationSAMLAuth, error) {
	ctx, cancel := context.WithTimeout(samlAuthRequestContext(ctx, pc), timeout)
	defer cancel()
	wantInline = strings.TrimSpace(wantInline)
	wantURL = strings.TrimSpace(wantURL)
	waitError := func() error {
		redactedInline := "empty"
		if wantInline != "" {
			sum := sha256.Sum256([]byte(wantInline))
			redactedInline = fmt.Sprintf("len=%d sha256=%s", len(wantInline), hex.EncodeToString(sum[:8]))
		}
		return fmt.Errorf("waiting for SAML auth state (enabled=%v, enforced=%v, wantInline=%s, wantURL=%q): %w", wantEnabled, wantEnforced, redactedInline, wantURL, ctx.Err())
	}
	for {
		if ctx.Err() != nil {
			return nil, waitError()
		}
		samlAuth, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationRead(ctx, organization).Execute()
		if resp != nil && resp.Body != nil {
			resp.Body.Close() // close immediately to avoid stacking defers in the loop
		}
		if ctx.Err() != nil {
			return nil, waitError()
		}
		if err != nil {
			return nil, handleSAMLAuthError(err, "waiting for SAML authentication")
		}
		if samlAuth == nil {
			return nil, fmt.Errorf("empty SAML authentication response while waiting for state")
		}
		if samlAuth.GetSamlAuthEnabled() == wantEnabled && samlAuth.GetSamlAuthEnforced() == wantEnforced {
			inlineMetadata := strings.TrimSpace(samlAuth.GetSamlMetadataInline())
			url, _ := samlAuth.GetSamlMetadataUrlOk()
			urlValue := ""
			if url != nil {
				urlValue = strings.TrimSpace(*url)
			}

			metadataMatch := false
			if wantInline != "" {
				metadataMatch = inlineMetadata == wantInline && urlValue == ""
			} else if wantURL != "" {
				metadataMatch = urlValue == wantURL && inlineMetadata == ""
			} else {
				metadataMatch = inlineMetadata == "" && urlValue == ""
			}

			if metadataMatch {
				return samlAuth, nil
			}
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, waitError()
		case <-timer.C:
		}
	}
}

// samlAuthCreate handles the creation of a new SAML authentication configuration
func samlAuthCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)
	organization := d.Get("organization").(string)

	samlAuth, err := buildSAMLAuthPatch(d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error building SAML auth request: %w", err))
	}

	req := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdate(samlAuthRequestContext(ctx, pc), organization).Data(*samlAuth)
	result, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdateExecute(req)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return diag.FromErr(handleSAMLAuthError(err, "creating SAML authentication"))
	}
	if result == nil {
		return diag.Errorf("empty SAML authentication response while creating")
	}

	d.SetId(generateSAMLAuthID(organization, result))
	// Wait for the backend to reflect enabled/metadata state
	confirmed, err := waitForSAMLAuthState(
		ctx,
		pc,
		organization,
		d.Get("saml_auth_enabled").(bool),
		d.Get("saml_auth_enforced").(bool),
		d.Get("saml_metadata_inline").(string),
		d.Get("saml_metadata_url").(string),
		30*time.Second,
	)
	if err != nil {
		return diag.FromErr(err)
	}
	// Do not replace the confirmed snapshot with another potentially stale GET.
	return diag.FromErr(setSAMLAuthFields(d, organization, confirmed))
}

// samlAuthRead retrieves the current SAML authentication configuration
func samlAuthRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)
	organization := d.Get("organization").(string)

	samlAuth, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationRead(samlAuthRequestContext(ctx, pc), organization).Execute()
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(handleSAMLAuthError(err, "reading SAML authentication"))
	}

	if err := setSAMLAuthFields(d, organization, samlAuth); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// samlAuthUpdate modifies an existing SAML authentication configuration
func samlAuthUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)
	organization := d.Get("organization").(string)

	samlAuth, err := buildSAMLAuthPatch(d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error building SAML auth request: %w", err))
	}

	req := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdate(samlAuthRequestContext(ctx, pc), organization).Data(*samlAuth)
	result, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdateExecute(req)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return diag.FromErr(handleSAMLAuthError(err, "updating SAML authentication"))
	}
	if result == nil {
		return diag.Errorf("empty SAML authentication response while updating")
	}
	// Wait for the backend to reflect enabled/metadata state
	confirmed, err := waitForSAMLAuthState(
		ctx,
		pc,
		organization,
		d.Get("saml_auth_enabled").(bool),
		d.Get("saml_auth_enforced").(bool),
		d.Get("saml_metadata_inline").(string),
		d.Get("saml_metadata_url").(string),
		30*time.Second,
	)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.FromErr(setSAMLAuthFields(d, organization, confirmed))
}

// samlAuthDelete disables SAML authentication for the organization
func samlAuthDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)
	organization := d.Get("organization").(string)

	samlAuth := cloudsmith.NewOrganizationSAMLAuthRequestPatch()
	samlAuth.SetSamlAuthEnabled(false)
	samlAuth.SetSamlAuthEnforced(false)
	samlAuth.SetSamlMetadataInline("")
	samlAuth.SetSamlMetadataUrl("")

	req := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdate(samlAuthRequestContext(ctx, pc), organization).Data(*samlAuth)
	result, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdateExecute(req)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return diag.FromErr(handleSAMLAuthError(err, "deleting SAML authentication"))
	}
	if result == nil {
		return diag.Errorf("empty SAML authentication response while deleting")
	}

	// Wait for the backend to reflect the disabled state
	if _, err := waitForSAMLAuthState(ctx, pc, organization, false, false, "", "", 30*time.Second); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

// samlAuthImport handles importing existing SAML authentication configurations
func samlAuthImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	pc := m.(*providerConfig)
	organization := d.Id()

	samlAuth, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationRead(samlAuthRequestContext(ctx, pc), organization).Execute()
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("SAML authentication not found for organization %s", organization)
		}
		return nil, handleSAMLAuthError(err, "importing SAML authentication")
	}

	if err := setSAMLAuthFields(d, organization, samlAuth); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

// buildSAMLAuthPatch creates a new SAML authentication patch request from the resource data
func buildSAMLAuthPatch(d *schema.ResourceData) (*cloudsmith.OrganizationSAMLAuthRequestPatch, error) {
	samlAuth := cloudsmith.NewOrganizationSAMLAuthRequestPatch()

	samlAuth.SetSamlAuthEnabled(d.Get("saml_auth_enabled").(bool))
	samlAuth.SetSamlAuthEnforced(d.Get("saml_auth_enforced").(bool))

	if v, ok := d.GetOk("saml_metadata_inline"); ok {
		samlAuth.SetSamlMetadataInline(v.(string))
		samlAuth.SetSamlMetadataUrl("")
	}

	if v, ok := d.GetOk("saml_metadata_url"); ok {
		samlAuth.SetSamlMetadataUrl(v.(string))
		samlAuth.SetSamlMetadataInline("")
	}

	return samlAuth, nil
}

// setSAMLAuthFields updates the resource data and identity from the same API snapshot.
func setSAMLAuthFields(d *schema.ResourceData, organization string, samlAuth *cloudsmith.OrganizationSAMLAuth) error {
	if samlAuth == nil {
		return fmt.Errorf("empty SAML authentication response")
	}
	// Helper function to reduce repetition and standardize error handling
	setField := func(key string, value interface{}) error {
		if err := d.Set(key, value); err != nil {
			return fmt.Errorf("error setting %s: %w", key, err)
		}
		return nil
	}

	// Set the basic fields first - these are always set regardless of their value
	if err := setField("organization", organization); err != nil {
		return err
	}
	if err := setField("saml_auth_enabled", samlAuth.GetSamlAuthEnabled()); err != nil {
		return err
	}
	if err := setField("saml_auth_enforced", samlAuth.GetSamlAuthEnforced()); err != nil {
		return err
	}

	inlineMetadata := samlAuth.GetSamlMetadataInline()
	url, hasURL := samlAuth.GetSamlMetadataUrlOk()

	if inlineMetadata != "" {
		if err := setField("saml_metadata_inline", inlineMetadata); err != nil {
			return err
		}
		if err := setField("saml_metadata_url", ""); err != nil {
			return err
		}
	} else if hasURL && url != nil && *url != "" {
		if err := setField("saml_metadata_url", url); err != nil {
			return err
		}
		if err := setField("saml_metadata_inline", ""); err != nil {
			return err
		}
	} else {
		// Neither present, clear both
		if err := setField("saml_metadata_inline", ""); err != nil {
			return err
		}
		if err := setField("saml_metadata_url", ""); err != nil {
			return err
		}
	}

	d.SetId(generateSAMLAuthID(organization, samlAuth))
	return nil
}

// generateSAMLAuthID creates a unique identifier for the SAML authentication resource
func generateSAMLAuthID(organization string, samlAuth *cloudsmith.OrganizationSAMLAuth) string {
	data := organization

	if samlAuth != nil {
		data += fmt.Sprintf("-%t", samlAuth.GetSamlAuthEnabled())
		data += fmt.Sprintf("-%t", samlAuth.GetSamlAuthEnforced())

		if url, hasURL := samlAuth.GetSamlMetadataUrlOk(); url != nil && hasURL {
			data += fmt.Sprintf("-%s", *url)
		}

		// Include inline metadata if present
		if metadata := samlAuth.GetSamlMetadataInline(); metadata != "" {
			data += fmt.Sprintf("-%s", metadata)
		}
	}

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// handleSAMLAuthError wraps an API error with the SAML action context while
// routing through formatAPIError so any typed ErrorDetail (detail/fields)
// from the API is surfaced to the caller.
func handleSAMLAuthError(err error, action string) error {
	return fmt.Errorf("error %s: %w", action, formatAPIError(err))
}

// resourceSAMLAuth returns a schema.Resource for managing SAML authentication configuration.
// This resource allows configuring SAML authentication settings for a Cloudsmith organization.
func resourceSAMLAuth() *schema.Resource {
	return &schema.Resource{
		CreateContext: samlAuthCreate,
		ReadContext:   samlAuthRead,
		UpdateContext: samlAuthUpdate,
		DeleteContext: samlAuthDelete,

		Importer: &schema.ResourceImporter{
			StateContext: samlAuthImport,
		},

		Schema: map[string]*schema.Schema{
			"organization": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Organization slug for SAML authentication",
			},
			"saml_auth_enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Enable SAML authentication for the organization",
			},
			"saml_auth_enforced": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Enforce SAML authentication for the organization",
			},
			"saml_metadata_inline": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Inline SAML metadata XML",
				ExactlyOneOf: []string{"saml_metadata_inline", "saml_metadata_url"},
				StateFunc: func(v interface{}) string {
					return strings.TrimSpace(v.(string))
				},
			},
			"saml_metadata_url": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "URL to fetch SAML metadata",
				ExactlyOneOf: []string{"saml_metadata_inline", "saml_metadata_url"},
			},
		},
	}
}
