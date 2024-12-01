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