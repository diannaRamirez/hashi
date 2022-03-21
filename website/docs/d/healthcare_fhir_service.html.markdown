---
subcategory: "Healthcare"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_healthcare_fhir_service"
description: |- Get information about an existing Healthcare Fhir Service
---

# Data Source: azurerm_healthcare_fhir_service

Use this data source to access information about an existing Healthcare Fhir Service

## Example Usage

```hcl
data "azurerm_healthcare_fhir_service" "example" {
  name                = "example-healthcare_fhir_service"
  resource_group_name = "example-resources"
  workspace_id        = "example-workspace"
}

output "healthcare_fhir_service_id" {
  value = data.azurerm_healthcare_fhir_service.example.id
}
```

## Argument Reference

* `name` - The name of the Healthcare Fhir Service.

* `resource_group_name` - The name of the Resource Group in which the Healthcare Fhir Service exists.

* `workspace_id` - The name of the Healthcare Workspace in which the Healthcare Fhir Service exists.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Healthcare Fhir Service.

* `location` - The Azure Region where the Healthcare Fhir Service is located.

* `authentication_configuration` - The `authentication_configuration` block as defined below.

* `kind` - The kind of the Healthcare Fhir Service. 

* `identity` - The `identity` block as defined below.

* `access_policy_object_ids` The list of the access policies of the service instance.

* `cors_configuration` The `cors_configuration` block as defined below.

* `acr_login_servers` The list of azure container registry settings used for convert data operation of the service instance.

* `authentication_configuration` The `authentication_configuration` block as defined below.

* `export_storage_account_name` The name of the export storage account which accepts the operation configuration information.

* `public_network_access_enabled` The public networks when data plane traffic coming from public networks while private endpoint is enabled.

* `tags` - A map of tags assigned to the Healthcare Fhir Service.

---
An `identity` block supports the following:

* `type` - (Required) The type of identity used for the Healthcare Fhir service. Possible values are `SystemAssigned` and `UserAssigned`. If `UserAssigned` is set, an `identity_ids` must be set as well.
* `identity_ids` - (Optional) The list of User Assigned Identity IDs which should be assigned to this Healthcare Fhir service.

---
A `cors_configuration` block supports the following:

* `allowed_origins` - (Required) The set of origins to be allowed via CORS.
* `allowed_headers` - (Required) The set of headers to be allowed via CORS.
* `allowed_methods` - (Required) The methods to be allowed via CORS.
* `max_age_in_seconds` - (Required) The max age to be allowed via CORS.
* `allow_credentials` - (Boolean) The credentials are allowed via CORS.

---
An `authentication_configuration` supports the following:

* `authority` The Azure Active Directory (tenant) that serves as the authentication authority to access the service. The default authority is the Directory defined in the authentication scheme in use when running Terraform.
  Authority must be registered to Azure AD and in the following format: https://{Azure-AD-endpoint}/{tenant-id}.
* `audience` The intended audience to receive authentication tokens for the service. The default value is https://<name>.fhir.azurehealthcareapis.com
* `smart_proxy_enabled` The 'SMART on FHIR' option for mobile and web implementations.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `read` - (Defaults to 5 minutes) Used when retrieving the Healthcare Fhir Service.

