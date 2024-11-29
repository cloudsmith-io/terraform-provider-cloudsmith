// Package cloudsmith provides Terraform provider functionality for managing Cloudsmith resources.
package cloudsmith

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudsmith-io/cloudsmith-api-go"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// samlAuthCreate handles the creation of a new SAML authentication configuration
func samlAuthCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)
	organization := d.Get("organization").(string)

	samlAuth, err := buildSAMLAuthPatch(d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error building SAML auth request: %w", err))
	}

	req := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdate(pc.Auth, organization).Data(*samlAuth)
	result, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdateExecute(req)
	if err != nil {
		return diag.FromErr(handleSAMLAuthError(err, resp, "creating SAML authentication"))
	}

	d.SetId(generateSAMLAuthID(organization, result))
	return samlAuthRead(ctx, d, m)
}

// samlAuthRead retrieves the current SAML authentication configuration
func samlAuthRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	pc := m.(*providerConfig)
	organization := d.Get("organization").(string)

	samlAuth, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationRead(pc.Auth, organization).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(handleSAMLAuthError(err, resp, "reading SAML authentication"))
	}

	d.Set("organization", organization)
	d.SetId(generateSAMLAuthID(organization, samlAuth))

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

	req := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdate(pc.Auth, organization).Data(*samlAuth)
	_, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdateExecute(req)
	if err != nil {
		return diag.FromErr(handleSAMLAuthError(err, resp, "updating SAML authentication"))
	}

	return samlAuthRead(ctx, d, m)
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

	req := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdate(pc.Auth, organization).Data(*samlAuth)
	_, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationPartialUpdateExecute(req)
	if err != nil {
		return diag.FromErr(handleSAMLAuthError(err, resp, "deleting SAML authentication"))
	}

	d.SetId("")
	return nil
}

// samlAuthImport handles importing existing SAML authentication configurations
func samlAuthImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	pc := m.(*providerConfig)
	organization := d.Id()

	samlAuth, resp, err := pc.APIClient.OrgsApi.OrgsSamlAuthenticationRead(pc.Auth, organization).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("SAML authentication not found for organization %s", organization)
		}
		return nil, handleSAMLAuthError(err, resp, "importing SAML authentication")
	}

	d.Set("organization", organization)
	d.SetId(generateSAMLAuthID(organization, samlAuth))

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

// setSAMLAuthFields updates the resource data with values from the API response
func setSAMLAuthFields(d *schema.ResourceData, organization string, samlAuth *cloudsmith.OrganizationSAMLAuth) error {
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

	// Handle inline metadata - only set if non-empty
	if inlineMetadata := samlAuth.GetSamlMetadataInline(); inlineMetadata != "" {
		if err := setField("saml_metadata_inline", inlineMetadata); err != nil {
			return err
		}
	}

	// Handle URL metadata with null handling
	url, hasURL := samlAuth.GetSamlMetadataUrlOk()
	if !hasURL || *url == "" {
		return setField("saml_metadata_url", nil)
	}
	return setField("saml_metadata_url", url)
}

// generateSAMLAuthID creates a unique identifier for the SAML authentication resource
func generateSAMLAuthID(organization string, samlAuth *cloudsmith.OrganizationSAMLAuth) string {
	data := organization

	if samlAuth != nil {
		data += fmt.Sprintf("-%t", samlAuth.GetSamlAuthEnabled())
		data += fmt.Sprintf("-%t", samlAuth.GetSamlAuthEnforced())

		if url, hasURL := samlAuth.GetSamlMetadataUrlOk(); hasURL {
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

// handleSAMLAuthError creates formatted error messages for API operations
func handleSAMLAuthError(err error, resp *http.Response, action string) error {
	if resp != nil {
		return fmt.Errorf("error %s: %v (status: %d)", action, err, resp.StatusCode)
	}
	return fmt.Errorf("error %s: %v", action, err)
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
