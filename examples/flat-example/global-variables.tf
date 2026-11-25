
data "cloudsmith_organization" "org-demo" {
  slug = "YOUR-ORG-NAME"
}

variable "api_key" {
  type    = string
  default = "YOUR-API-KEY"
}

variable "default_storage_region" {
  type    = string
  default = "us-ohio"
}

variable "main_entitlement_token" {
  type    = string
  default = "Main Entitlement"
}

variable "main_entitlement_token_limit_num_downloads" {
  type    = string
  default = 1000
}

variable "geopip_allow_countries" {
  type    = list(string)
  default = ["US", "GB", "DE"]
}
