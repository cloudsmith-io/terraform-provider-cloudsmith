# Usage Limits Data Source

The usage limits data source reads the current on-demand usage settings and plan-specific maximum values for a Cloudsmith organization without managing them.

## Example Usage

```hcl
provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_usage_limits" "example" {
  organization = "my-organization"
}

output "package_delivery_limit" {
  value = data.cloudsmith_usage_limits.example.bandwidth_overage_limit
}
```

## Argument Reference

* `organization` - (Required) The organization slug.

## Attribute Reference

* `allow_open_source_overage` - Whether on-demand open source usage is allowed.
* `bandwidth_overage_limit` - The Package Delivery On-Demand Limit, in GB.
* `storage_overage_limit` - The Artifact Data On-Demand Limit, in GB.
* `bandwidth_maximum` - The maximum package delivery overage allowed by the organization's plan, in GB.
* `storage_maximum` - The maximum artifact data overage allowed by the organization's plan, in GB.

To manage these settings, use the [`cloudsmith_usage_limits` resource](../resources/usage_limits.md).
