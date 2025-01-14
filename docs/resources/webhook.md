# Webhook Resource

The webhook resource allows the creation and management of webhooks for a given Cloudsmith repository. Webhooks allow integration with external systems by emitting events via HTTP POST request.

See [help.cloudsmith.io](https://help.cloudsmith.io/docs/webhooks) for full webhook documentation.

## Example Usage

```hcl
provider "cloudsmith" {
    api_key = "my-api-key"
}

data "cloudsmith_organization" "my_organization" {
    slug = "my-organization"
}

resource "cloudsmith_repository" "my_repository" {
    description = "A certifiably-awesome private package repository"
    name        = "My Repository"
    namespace   = "${data.cloudsmith_organization.my_organization.slug_perm}"
    slug        = "my-repository"
}

resource "cloudsmith_webhook" "my_webhook" {
    namespace  = cloudsmith_repository.my_repository.namespace
    repository = cloudsmith_repository.my_repository.slug_perm

 events              = ["package.created", "package.deleted"]
 request_body_format = "Handlebars Template"
 target_url          = "https://example.com"

 template {
  event = "package.created"
  template = "created: {{data.name}}: {{data.version}}"
 }

 template {
  event = "package.deleted"
  template = "deleted: {{data.name}}: {{data.version}}"
 }
}
```

## Argument Reference

* `events` - (Required) List of events for which this webhook will be fired. Supported events include:
	+ `*` - Catch-all for all events.
	+ `package.created` - Fired when a package is created.
	+ `package.deleted` - Fired when a package is deleted.
	+ `package.downloaded` - Fired when a package is downloaded.
	+ `package.failed` - Fired when a package fails.
	+ `package.security_scanned` - Fired when a package security scan is completed.
	+ `package.synced` - Fired when a package is synced.
	+ `package.syncing` - Fired when a package is syncing.
	+ `package.tags_updated` - Fired when package tags are updated.
	+ `package.released` - Fired when a package is released.
	+ `package.restored` - Fired when a package is restored.
	+ `package.quarantined` - Fired when a package is quarantined.
* `is_active` - (Optional) If enabled, the webhook will trigger on subscribed events and send payloads to the configured target URL.
* `namespace` - (Required) Namespace (or organization) to which this webhook belongs.
* `package_query` - (Optional) The package-based search query for webhooks to fire. This uses the same syntax as the standard search used for repositories, and also supports boolean logic operators such as OR/AND/NOT and parentheses for grouping. If a package does not match, the webhook will not fire.
* `repository` - (Required) Repository to which this webhook belongs.
* `request_body_format` - (Optional) The format of the payloads for webhook requests.
* `request_body_template_format` - (Optional) The format of the payloads for webhook requests.
* `request_content_type` - (Optional) The value that will be sent for the 'Content Type' header.
* `secret_header` - (Optional) The header to send the predefined secret in. This must be unique from existing headers or it won't be sent. You can use this as a form of authentication on the endpoint side.
* `secret_value` - (Optional) The value for the predefined secret (note: this is treated as a passphrase and is encrypted when we store it). You can use this as a form of authentication on the endpoint side.
* `signature_key` - (Optional) The value for the signature key - This is used to generate an HMAC-based hex digest of the request body, which we send as the X-Cloudsmith-Signature header so that you can ensure that the request wasn't modified by a malicious party (note: this is treated as a passphrase and is encrypted when we store it).
* `target_url` - (Required) The destination URL that webhook payloads will be POST'ed to.
* `template` - (Optional) Variable number of blocks containing templates used to render webhook content before sending.
  * `event` - (Required) The event for which this template will be applied.
  * `template` - (Required) The contents of the template to be rendered.
* `is_active` - (Optional) If enabled, SSL certificates is verified when webhooks are sent. It's recommended to leave this enabled as not verifying the integrity of SSL certificates leaves you susceptible to Man-in-the-Middle (MITM) attacks.

## Attribute Reference

* `created_at` - ISO 8601 timestamp at which the webhook was created.
* `created_by` - The user/account that created the webhook.
* `disable_reason` - Why this webhook has been disabled.
* `slug_perm` - The slug_perm immutably identifies the webhook. It will never change once a webhook has been created.
* `updated_at` - ISO 8601 timestamp at which the webhook was updated.
* `updated_by` - The user/account that updated the webhook.

## Import

This resource can be imported using the organization slug, the repository slug, and the webhook slug:

```shell
terraform import cloudsmith_webhook.my_webhook my-organization.my-repository.w3bh0okS1uG
```
