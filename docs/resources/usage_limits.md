# Usage Limits Resource

The usage limits resource manages the on-demand package delivery and artifact data limits for a Cloudsmith organization, including whether on-demand open source usage is allowed.

**Note: Cloudsmith does not provide an API operation to delete or reset usage limits. Destroying this resource removes it from Terraform state and leaves the organization's current settings unchanged.**

## Example Usage

```hcl
provider "cloudsmith" {
  api_key = "my-api-key"
}

resource "cloudsmith_usage_limits" "example" {
  organization              = "my-organization"
  bandwidth_overage_limit   = 100
  storage_overage_limit     = 50
  allow_open_source_overage = true
}
```

## Argument Reference

The following arguments are supported:

* `organization` - (Required) The organization slug. Changing this value creates a new resource.
* `bandwidth_overage_limit` - (Required) Package Delivery On-Demand Limit, in GB. Must be zero or greater, no higher than `bandwidth_maximum`, and no lower than current package delivery overage usage.
* `storage_overage_limit` - (Required) Artifact Data On-Demand Limit, in GB. Must be zero or greater, no higher than `storage_maximum`, and no lower than current artifact data overage usage.
* `allow_open_source_overage` - (Required) Whether to allow on-demand open source usage.

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

* `bandwidth_maximum` - The maximum package delivery overage allowed by the organization's plan, in GB.
* `storage_maximum` - The maximum artifact data overage allowed by the organization's plan, in GB.

## Import

Usage limits can be imported using the organization slug:

```shell
terraform import cloudsmith_usage_limits.example my-organization
```

To inspect an organization's usage limits without managing them, use the [`cloudsmith_usage_limits` data source](../data-sources/usage_limits.md).
