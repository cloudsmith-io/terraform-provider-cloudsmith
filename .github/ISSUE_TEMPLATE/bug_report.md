---
name: Bug Report
about: Report a bug or unexpected behavior in the Cloudsmith Terraform Provider
title: "[BUG] "
labels: bug
---

## Description

<!-- A clear and concise description of what the bug is -->

## Terraform Version

<!-- Run `terraform version`. If you are not running the latest Terraform version, please upgrade first because your issue may already have been fixed. -->

## Provider Version

<!-- Run `terraform providers` or check `.terraform.lock.hcl` to show the Cloudsmith provider version. If you are not running the latest provider version, please upgrade first because your issue may already have been fixed. -->

## Affected Resources/Data Sources

<!-- List the resources or data sources involved, for example `cloudsmith_repository`. If this affects multiple resources or data sources, please mention that because it may point to shared provider logic or Terraform core behavior. -->

## Terraform Configuration

<!-- Share the smallest Terraform configuration that reproduces the problem. Remove any secrets before posting. For large configurations, use a GitHub Gist or similar link. -->

```hcl
# paste configuration here
```

## Steps to Reproduce

<!-- Please list the exact steps required to reproduce the issue, for example `terraform apply`. -->

1.
2.
3.

## Expected Behavior

<!-- What you expected to happen -->

## Actual Behavior

<!-- What actually happened -->

## Debug Output

<!-- Please provide a link to a GitHub Gist containing the complete debug output. You can capture this by running Terraform with `TF_LOG=DEBUG`. Please do not paste the full debug output directly in the issue. -->

## Panic Output

<!-- If Terraform produced a panic, please provide a link to a GitHub Gist containing the contents of `crash.log`. -->

## Environment

- **OS**: <!-- e.g. macOS 15, Ubuntu 24.04, Windows 11 -->
- **Execution Context**: <!-- e.g. local machine, CI, Terraform Cloud -->

## Important Factoids

<!-- Are there any atypical details about your Cloudsmith account, Terraform setup, or environment that we should know? -->

## References

<!-- Are there any related GitHub issues or pull requests that should be linked here? For example `GH-1234`. -->

## Additional Context

<!-- Add any other context about the problem here -->
