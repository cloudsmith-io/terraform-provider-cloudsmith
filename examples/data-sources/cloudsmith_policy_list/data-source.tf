data "cloudsmith_policy_list" "named" {
  workspace = "my-workspace"
  query     = "name:quarantine*"
  sort      = "-created_at"
}
