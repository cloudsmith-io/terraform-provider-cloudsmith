schema_version = 1

project {
  license        = "MPL-2.0"
  copyright_year = 2026
  copyright_holder = "Cloudsmith Ltd"

  header_ignore = [
    # Import examples render verbatim into the registry documentation.
    "examples/**/import.sh",
    "examples/**/import-by-string-id.tf",
  ]
}
