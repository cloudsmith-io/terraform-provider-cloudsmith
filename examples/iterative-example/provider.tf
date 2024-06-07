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