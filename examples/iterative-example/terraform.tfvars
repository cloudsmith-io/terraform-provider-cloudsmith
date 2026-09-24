# Copyright Cloudsmith 2026
# SPDX-License-Identifier: MPL-2.0

repositories = {
  "development" : {
    "add_developers" : true
  },
  "staging" : {
    "add_developers" : false
  },
  "production" : {
    "add_developers" : false
    "oidc_claims" : {
      "repository" = "Owner/ProductionGithubRepoName"
    }
  }
}
