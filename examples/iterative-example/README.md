# Terraform iterative structure example

This terraform example project will setup 3 repositories using a loop-based approach for instances where each repository
shares many attributes with the others, but changes slightly based on their individual needs.

* Creates a Development, Staging and Production repository
  * Disables User entitlements. The default entitlement token can then be disabled so that authentication can only be done via API keys!
* Creates an individual CI service account for each repository which gets write access
  * Configures Github OIDC for authenticating as the service accounts (see our documentation [here](https://docs.cloudsmith.com/access-control/setup-cloudsmith-to-authenticate-with-oidc-in-github-actions) for how to configure that on the Github side)
* Creates a Developers team which gets write permission on the Development repository
* Configures DockerHub and Chainguard upstreams on each repository
* Configures a Vulnerablity policy that blocks high severity vulnerabilities.
* Configures a license policy that blocks AGPL licensed packages.

## Usage

To get started, supply your API key and org name in `global-variables.tf` file.
Run `terraform init` and then run `terraform apply` to execute the plan.

Configuration can be done in the `terraform.tfvars` file.
