# tflint-ignore-file: terraform_required_version, terraform_required_providers

terraform {
  required_providers {
    cloudsmith = {
      source = "cloudsmith-io/cloudsmith"
    }
  }
}

provider "cloudsmith" {
  api_key = var.api_key
}
