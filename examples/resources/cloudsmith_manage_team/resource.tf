# Copyright Cloudsmith Ltd 2026
# SPDX-License-Identifier: MPL-2.0

resource "cloudsmith_manage_team" "example" {
  organization = "example_org"
  team_name    = "example_team"
  members {
    role = "Manager"
    user = "user1"
  }
  members {
    role = "Member"
    user = "user2"
  }
}
