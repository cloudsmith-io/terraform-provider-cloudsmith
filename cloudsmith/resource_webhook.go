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
	eventTypes = []string{
		"*",
		"package.created",
		"package.deleted",
		"package.downloaded",
		"package.failed",
		"package.security_scanned",
		"package.synced",
		"package.syncing",
		"package.tags_updated",
	}
	requestBodyFormatMap = map[int]string{
		0: "JSON Object",
		1: "JSON Array",
		2: "Form Encoded JSON Object",
		3: "Handlebars Template",
	}
	requestBodyTemplateFormatMap = map[int]string{
		0: "Generic (user-defined)",
		1: "JSON (application/json)",
		2: "XML (application/xml)",
	}
)

// expandEvents extracts "events" from TF state as a *schema.Set and converts to
// a slice of strings we can use when interacting with the Cloudsmith API.
func expandEvents(d *schema.ResourceData) []string {
	set := d.Get("events").(*schema.Set)
	return lo.Map(set.List(), func(item interface{}, _ int) string {
		return item.(string)
	})
}

// flattenEvents takes a slice of strings as returned by the Cloudsmith API and
// converts to a *schema.Set that can be stored in TF state.
func flattenEvents(events []string) *schema.Set {
	set := schema.NewSet(schema.HashString, []interface{}{})
	for _, event := range events {
		set.Add(event)
	}
	return set
}

// expandRequestBodyFormat extracts the request body format from TF state as a
// human-readable string (if set) and converts it to an int64 that can be used
// to interact with the Cloudsmith API.
func expandRequestBodyFormat(d *schema.ResourceData) *int64 {
	value := optionalString(d, "request_body_format")
	if value == nil {
		return nil
	}

	m := lo.Invert(requestBodyFormatMap)
	v, ok := m[*value]

	if !ok {
		return nil
	}

	return cloudsmith.PtrInt64(int64(v))
}

// flattenRequestBodyFormat takes an int64 as returned by the Cloudsmith API and
// converts it to a human readable string that can be stored/used in TF state.
func flattenRequestBodyFormat(fmt int64) string {
	return requestBodyFormatMap[int(fmt)]
}

// expandRequestBodyTemplateFormat extracts the request body template format
// from TF state as a human-readable string (if set) and converts it to an int64
// that can be used to interact with the Cloudsmith API.
func expandRequestBodyTemplateFormat(d *schema.ResourceData) *int64 {
	value := optionalString(d, "request_body_template_format")
	if value == nil {
		return nil
	}

	m := lo.Invert(requestBodyTemplateFormatMap)
	v, ok := m[*value]

	if !ok {
		return nil
	}

	return cloudsmith.PtrInt64(int64(v))
}

// flattenRequestBodyTemplateFormat takes an int64 as returned by the Cloudsmith API and
// converts it to a human readable string that can be stored/used in TF state.
func flattenRequestBodyTemplateFormat(fmt int64) string {
	return requestBodyTemplateFormatMap[int(fmt)]
}

// expandEvents extracts "events" from TF state as a *schema.Set and converts to
// a slice of strings we can use when interacting with the Cloudsmith API.
func expandTemplates(d *schema.ResourceData) []cloudsmith.WebhookTemplate {
	set := d.Get("template").(*schema.Set)

	return lo.Map(set.List(), func(x interface{}, index int) cloudsmith.WebhookTemplate {
		m := x.(map[string]interface{})
		t := cloudsmith.WebhookTemplate{}
		t.SetEvent(m["event"].(string))
		t.SetTemplate(m["template"].(string))

		return t
	})
}

// flattenTemplates takes a slice of cloudsmith.WebhookTemplate as returned by
// the Cloudsmith API and converts to a *schema.Set that can be stored in TF
// state.
func flattenTemplates(templates []cloudsmith.WebhookTemplate) *schema.Set {
	templateSchema := resourceWebhook().Schema["template"].Elem.(*schema.Resource)
	set := schema.NewSet(schema.HashResource(templateSchema), []interface{}{})
	for _, template := range templates {
		set.Add(map[string]interface{}{
			"event":    template.GetEvent(),
			"template": template.GetTemplate(),
		})
	}
	return set
}

func resourceWebhookCreate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	req := pc.APIClient.WebhooksApi.WebhooksCreate(pc.Auth, namespace, repository)
	req = req.Data(cloudsmith.RepositoryWebhookRequest{
		Events:                    expandEvents(d),
		IsActive:                  optionalBool(d, "is_active"),
		PackageQuery:              nullableString(d, "package_query"),
		RequestBodyFormat:         expandRequestBodyFormat(d),
		RequestBodyTemplateFormat: expandRequestBodyTemplateFormat(d),
		RequestContentType:        nullableString(d, "request_content_type"),
		SecretHeader:              nullableString(d, "secret_header"),
		SecretValue:               nullableString(d, "secret_value"),
		SignatureKey:              optionalString(d, "signature_key"),
		TargetUrl:                 requiredString(d, "target_url"),
		Templates:                 expandTemplates(d),
		VerifySsl:                 optionalBool(d, "verify_ssl"),
	})

	webhook, _, err := pc.APIClient.WebhooksApi.WebhooksCreateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(webhook.GetSlugPerm())

	checkerFunc := func() error {
		req := pc.APIClient.WebhooksApi.WebhooksRead(pc.Auth, namespace, repository, d.Id())
		if _, resp, err := pc.APIClient.WebhooksApi.WebhooksReadExecute(req); err != nil {
			if is404(resp) {
				return errKeepWaiting
			}
			return err
		}
		return nil
	}
	if err := waiter(checkerFunc, defaultCreationTimeout, defaultCreationInterval); err != nil {
		return fmt.Errorf("error waiting for webhook (%s) to be created: %w", d.Id(), err)
	}

	return resourceWebhookRead(d, m)
}

func resourceWebhookRead(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	req := pc.APIClient.WebhooksApi.WebhooksRead(pc.Auth, namespace, repository, d.Id())

	webhook, resp, err := pc.APIClient.WebhooksApi.WebhooksReadExecute(req)
	if err != nil {
		if is404(resp) {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("created_at", timeToString(webhook.GetCreatedAt()))
	d.Set("created_by", webhook.GetCreatedBy())
	d.Set("disable_reason", webhook.GetDisableReasonStr())
	d.Set("events", flattenEvents(webhook.GetEvents()))
	d.Set("is_active", webhook.GetIsActive())
	d.Set("package_query", webhook.GetPackageQuery())
	d.Set("request_body_format", flattenRequestBodyFormat(webhook.GetRequestBodyFormat()))
	d.Set("request_body_template_format", flattenRequestBodyTemplateFormat(webhook.GetRequestBodyTemplateFormat()))
	d.Set("request_content_type", webhook.GetRequestContentType())
	d.Set("secret_header", webhook.GetSecretHeader())
	d.Set("slug_perm", webhook.GetSlugPerm())
	d.Set("target_url", webhook.GetTargetUrl())
	d.Set("template", flattenTemplates(webhook.GetTemplates()))
	d.Set("updated_at", timeToString(webhook.GetUpdatedAt()))
	d.Set("updated_by", webhook.GetUpdatedBy())
	d.Set("verify_ssl", webhook.GetVerifySsl())

	// namespace and repository are not returned from the entitlement read
	// endpoint, so we can use the values stored in resource state. We rely on
	// ForceNew to ensure if either changes a new resource is created.
	d.Set("namespace", namespace)
	d.Set("repository", repository)

	return nil
}

func resourceWebhookUpdate(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	req := pc.APIClient.WebhooksApi.WebhooksPartialUpdate(pc.Auth, namespace, repository, d.Id())
	req = req.Data(cloudsmith.RepositoryWebhookRequestPatch{
		Events:                    expandEvents(d),
		IsActive:                  optionalBool(d, "is_active"),
		PackageQuery:              nullableString(d, "package_query"),
		RequestBodyFormat:         expandRequestBodyFormat(d),
		RequestBodyTemplateFormat: expandRequestBodyTemplateFormat(d),
		RequestContentType:        nullableString(d, "request_content_type"),
		SecretHeader:              nullableString(d, "secret_header"),
		SecretValue:               nullableString(d, "secret_value"),
		SignatureKey:              optionalString(d, "signature_key"),
		TargetUrl:                 optionalString(d, "target_url"),
		Templates:                 expandTemplates(d),
		VerifySsl:                 optionalBool(d, "verify_ssl"),
	})

	webhook, _, err := pc.APIClient.WebhooksApi.WebhooksPartialUpdateExecute(req)
	if err != nil {
		return err
	}

	d.SetId(webhook.GetSlugPerm())

	checkerFunc := func() error {
		// this is somewhat of a hack until we have a better way to poll for an
		// entitlement being updated (changes incoming on the API side)
		time.Sleep(time.Second * 5)
		return nil
	}
	if err := waiter(checkerFunc, defaultUpdateTimeout, defaultUpdateInterval); err != nil {
		return fmt.Errorf("error waiting for webhook (%s) to be updated: %w", d.Id(), err)
	}

	return resourceWebhookRead(d, m)
}

func resourceWebhookDelete(d *schema.ResourceData, m interface{}) error {
	pc := m.(*providerConfig)

	namespace := requiredString(d, "namespace")
	repository := requiredString(d, "repository")

	req := pc.APIClient.WebhooksApi.WebhooksDelete(pc.Auth, namespace, repository, d.Id())
	_, err := pc.APIClient.WebhooksApi.WebhooksDeleteExecute(req)
	if err != nil {
		return err
	}

	checkerFunc := func() error {
		req := pc.APIClient.WebhooksApi.WebhooksRead(pc.Auth, namespace, repository, d.Id())
		if _, resp, err := pc.APIClient.WebhooksApi.WebhooksReadExecute(req); err != nil {
			if is404(resp) {
				return nil
			}
			return err
		}
		return errKeepWaiting
	}
	if err := waiter(checkerFunc, defaultDeletionTimeout, defaultDeletionInterval); err != nil {
		return fmt.Errorf("error waiting for webhook (%s) to be deleted: %w", d.Id(), err)
	}

	return nil
}

//nolint:funlen
func resourceWebhook() *schema.Resource {
	return &schema.Resource{
		Create: resourceWebhookCreate,
		Read:   resourceWebhookRead,
		Update: resourceWebhookUpdate,
		Delete: resourceWebhookDelete,

		Schema: map[string]*schema.Schema{
			"created_at": {
				Type:        schema.TypeString,
				Description: "ISO 8601 timestamp at which the webhook was created.",
				Computed:    true,
			},
			"created_by": {
				Type:        schema.TypeString,
				Description: "The user/account that created the webhook.",
				Computed:    true,
			},
			"disable_reason": {
				Type:        schema.TypeString,
				Description: "Why this webhook has been disabled.",
				Computed:    true,
			},
			"events": {
				Type:        schema.TypeSet,
				Description: "List of events for which this webhook will be fired.",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(eventTypes, false),
				},
				MinItems: 1,
				Required: true,
			},
			"is_active": {
				Type:        schema.TypeBool,
				Description: "If enabled, the webhook will trigger on subscribed events and send payloads to the configured target URL.",
				Optional:    true,
				Computed:    true,
			},
			"namespace": {
				Type:         schema.TypeString,
				Description:  "Namespace to which this webhook belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"package_query": {
				Type: schema.TypeString,
				Description: "The package-based search query for webhooks to fire. This uses the same " +
					"syntax as the standard search used for repositories, and also supports boolean " +
					"logic operators such as OR/AND/NOT and parentheses for grouping. If a package does " +
					"not match, the webhook will not fire.",
				Optional: true,
			},
			"repository": {
				Type:         schema.TypeString,
				Description:  "Repository to which this webhook belongs.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"request_body_format": {
				Type:         schema.TypeString,
				Description:  "The format of the payloads for webhook requests.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(lo.Values(requestBodyFormatMap), false),
			},
			"request_body_template_format": {
				Type:         schema.TypeString,
				Description:  "The format of the payloads for webhook requests.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(lo.Values(requestBodyTemplateFormatMap), false),
			},
			"request_content_type": {
				Type:         schema.TypeString,
				Description:  "The value that will be sent for the 'Content Type' header.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"secret_header": {
				Type: schema.TypeString,
				Description: "The header to send the predefined secret in. This must be unique from existing " +
					"headers or it won't be sent. You can use this as a form of authentication on the endpoint side.",
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"secret_value": {
				Type: schema.TypeString,
				Description: "The value for the predefined secret (note: this is treated as a passphrase and is " +
					"encrypted when we store it). You can use this as a form of authentication on the endpoint side.",
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"signature_key": {
				Type: schema.TypeString,
				Description: "The value for the signature key - This is used to generate an HMAC-based hex digest of " +
					"the request body, which we send as the X-Cloudsmith-Signature header so that you can ensure that " +
					"the request wasn't modified by a malicious party (note: this is treated as a passphrase and is " +
					"encrypted when we store it).",
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"slug_perm": {
				Type: schema.TypeString,
				Description: "The slug_perm immutably identifies the webhook. " +
					"It will never change once a webhook has been created.",
				Computed: true,
			},
			"target_url": {
				Type:         schema.TypeString,
				Description:  "The destination URL that webhook payloads will be POST'ed to.",
				Required:     true,
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"template": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"event": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(eventTypes, false),
						},
						"template": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Optional: true,
			},
			"updated_at": {
				Type:        schema.TypeString,
				Description: "ISO 8601 timestamp at which the webhook was updated.",
				Computed:    true,
			},
			"updated_by": {
				Type:        schema.TypeString,
				Description: "The user/account that updated the webhook.",
				Computed:    true,
			},
			"verify_ssl": {
				Type: schema.TypeBool,
				Description: "If enabled, SSL certificates is verified when webhooks are sent. It's recommended to " +
					"leave this enabled as not verifying the integrity of SSL certificates leaves you susceptible " +
					"to Man-in-the-Middle (MITM) attacks.",
				Optional: true,
				Computed: true,
			},
		},

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}
