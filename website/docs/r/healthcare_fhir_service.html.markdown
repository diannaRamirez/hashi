---
subcategory: "Healthcare"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_healthcare_fhir_service"
description: |-
  Manages a Healthcare Fhir Service.
---

# azurerm_healthcare_fhir_service

Manages a Healthcare Fhir Service

## Example Usage

```hcl
data "azurerm_client_config" "current" {
}
resource "azurerm_healthcare_fhir_service" "test" {
  name                = "tfexfhir"
  location            = "east us"
  resource_group_name = "tfex-resource_group"
  workspace_id        = "tfex-workspace_id"
  kind                = "fhir-R4"
  authentication_configuration {
    authority = "https://login.microsoftonline.com/tenantId"
    audience  = "https://tfexfhir.fhir.azurehealthcareapis.com"
  }
  access_policy_object_ids = [
    data.azurerm_client_config.current.object_id,
  ]

  identity {
    type = "SystemAssigned"
  }

  acr_login_servers = ["tfex-container_registry_login_server"]

  cors_configuration {
    allowed_origins    = ["https://tfex.com:123", "https://tfex1.com:3389"]
    allowed_headers    = ["*"]
    allowed_methods    = ["GET", "DELETE", "PUT"]
    max_age_in_seconds = 3600
    allow_credentials  = true
  }
  export_storage_account_name = "storage_account_name"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the Healthcare Fhir Service. Changing this forces a new Healthcare Fhir Service to be created.

* `workspace_id`  - (Required) Specifies the name of the Healthcare Workspace where the Healthcare Fhir Service should exist. Changing this forces a new Healthcare Fhir Service to be created.

* `location` - (Required) Specifies the Azure Region where the Healthcare Fhir Service should be created. Changing this forces a new Healthcare Fhir Service to be created.

* `kind` - (Required) Specifies the kind of the Healthcare Fhir Service. Possible values are: `fhir-Stu3` and `fhir-R4`. Defaults to `fhir-R4`. Changing this forces a new Healthcare Fhir Service to be created.

* `identity` - (Optional) An `identity` block as defined below.

* `access_policy_object_ids` - (Optional) A list of the access policies of the service instance.

* `cors_configuration` - (Optional) A `cors_configuration` block as defined below.

* `acr_login_servers` - - (Optional) A list of azure container registry settings used for convert data operation of the service instance.

* `authentication_configuration` - (Required) An `authentication_configuration` block as defined below.

* `export_storage_account_name` - (Optional) specifies the name of the export storage account which accepts the operation configuration information

* `public_network_access_enabled` - (Optional) Whether to enabled public networks when data plane traffic coming from public networks while private endpoint is enabled.

---
An `identity` block supports the following:

* `type` - (Required) The type of identity used for the Healthcare Fhir service. Possible values are `SystemAssigned` and `UserAssigned`. If `UserAssigned` is set, an `identity_ids` must be set as well.
* `identity_ids` - (Optional) A list of User Assigned Identity IDs which should be assigned to this Healthcare Fhir service.

---
A `cors_configuration` block supports the following:

* `allowed_origins` - (Required) A set of origins to be allowed via CORS.
* `allowed_headers` - (Required) A set of headers to be allowed via CORS.
* `allowed_methods` - (Required) The methods to be allowed via CORS.
* `max_age_in_seconds` - (Required) The max age to be allowed via CORS.
* `allow_credentials` - (Boolean) If credentials are allowed via CORS.

---
An `authentication_configuration` supports the following:

* `authority` - (Optional) The Azure Active Directory (tenant) that serves as the authentication authority to access the service. The default authority is the Directory defined in the authentication scheme in use when running Terraform.
  Authority must be registered to Azure AD and in the following format: https://{Azure-AD-endpoint}/{tenant-id}.
* `audience` - (Optional) The intended audience to receive authentication tokens for the service. The default value is https://<name>.fhir.azurehealthcareapis.com
* `smart_proxy_enabled` - (Boolean) Enables the 'SMART on FHIR' option for mobile and web implementations.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Healthcare Fhir Service.

## Timeouts
The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the Healthcare Fhir Service.
* `update` - (Defaults to 30 minutes) Used when updating the Healthcare Fhir Service.
* `read` - (Defaults to 5 minutes) Used when retrieving the Healthcare Fhir Service.
* `delete` - (Defaults to 30 minutes) Used when deleting the Healthcare Fhir Service.

## Import

Healthcare Fhir Service can be imported using the resource`id`, e.g.

```shell
terraform import azurerm_healthcare_fhir_service.example /subscriptions/12345678-1234-9876-4563-123456789012/resourceGroups/group1/providers/Microsoft.HealthcareApis/workspaces/workspace1/fhirservices/service1
```
