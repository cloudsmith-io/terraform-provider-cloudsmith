provider "cloudsmith" {
  api_key = "my-api-key"
}

data "cloudsmith_namespace" "my_namespace" {
  slug = "my-namespace"
}
