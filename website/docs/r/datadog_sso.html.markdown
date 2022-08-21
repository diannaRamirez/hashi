---
subcategory: "Datadog"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_datadog_monitor_sso_configuration"
description: |-
  Manages SingleSignOn on the datadog Monitor.
---

# azurerm_datadog_monitor_sso_configuration

Manages SingleSignOn on the datadog Monitor.

## Example Usage

### Enabling SSO on monitor
```hcl
resource "azurerm_resource_group" "example" {
  name     = "example-datadog"
  location = "West US 2"
}
resource "azurerm_datadog_monitor_sso_configuration" "test" {
    name = "example-monitor"
    resource_group_name = azurerm_resource_group.example.name
    singlesignon_state = "Enable"
    enterprise_application_id = "XXXX"
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) The name which should be used for this datadog Monitor.

* `resource_group_name` - (Required) The name of the Resource Group where the datadog Monitor should exist.

* `singlesignon_state` - (Required) The state of SingleSignOn configuration.

* `enterprise_application_id` - (Required) The application Id to perform SSO operation.

--- 

* `configuration_name` - (Optional) The name of the SingleSignOn configuration.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - The ID of the SingleSignOn on datadog monitor.

* `type` - The type of the monitor resource.

* `provisioning_state` - The state of Datadog monitor.

* `singlesignon_url` - The SingleSignOn URL to login to Datadog org.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the SingleSignOn on the datadog Monitor.
* `read` - (Defaults to 5 minutes) Used when retrieving the SingleSignOn on the datadog Monitor.
* `update` - (Defaults to 30 minutes) Used when updating the SingleSignOn on the datadog Monitor.
* `delete` - (Defaults to 30 minutes) Used when deleting the SingleSignOn on the datadog Monitor.