# ID format: <workspace_slug>.<service_slug>
# The API does not return the service's API key, so the `key` attribute is not available after import.
terraform import cloudsmith_service.my_service my-workspace.my-service
