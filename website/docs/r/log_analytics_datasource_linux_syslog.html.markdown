---
subcategory: "Log Analytics"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_log_analytics_datasource_linux_syslog"
description: |-
  Manages a Log Analytics Linux Syslog DataSource.
---

# azurerm_log_analytics_datasource_linux_syslog

Manages a Log Analytics Linux Syslog DataSource.

## Example Usage

```hcl
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "West Europe"
}

resource "azurerm_log_analytics_workspace" "example" {
  name                = "example-law"
  location            = azurerm_resource_group.example.location
  resource_group_name = azurerm_resource_group.example.name
  sku                 = "PerGB2018"
}

resource "azurerm_log_analytics_datasource_linux_syslog" "example" {
  name                = "example-lad-ls"
  resource_group_name = azurerm_resource_group.example.name
  workspace_name      = azurerm_log_analytics_workspace.example.name
  severities          = ["emerg"]
  syslog_name         = "kern"
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) The name which should be used for this Log Analytics Linux Syslog DataSource. Changing this forces a new Log Analytics Linux Syslog DataSource to be created.

* `resource_group_name` - (Required) The name of the Resource Group where the Log Analytics Linux Syslog DataSource should exist. Changing this forces a new Log Analytics Linux Syslog DataSource to be created.

* `workspace_name` - (Required) The name of the Log Analytics Workspace where the Log Analytics Linux Syslog DataSource should exist. Changing this forces a new Log Analytics Linux Syslog DataSource to be created.

* `severities` - (Required) A list of severity level of Linux Syslog. Possible values are `emerg`, `alert`, `crit`, `err`, `warning`, `notice`, `info` and `debug`.

* `syslog_name` - (Required) The name of program which is logging the Linux Syslog. Possible values are `auth`, `authpriv`, `cron`, `daemon`, `ftp`, `kern`, `local0`, `local1`, `local2`, `local3`, `local4`, `local5`, `local6`, `local7`, `lpr`, `mail`, `news`, `syslog`, `user`, `uucp`.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported: 

* `id` - The ID of the Log Analytics Linux Syslog DataSource.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the Log Analytics Linux Syslog DataSource.
* `read` - (Defaults to 5 minutes) Used when retrieving the Log Analytics Linux Syslog DataSource.
* `update` - (Defaults to 30 minutes) Used when updating the Log Analytics Linux Syslog DataSource.
* `delete` - (Defaults to 30 minutes) Used when deleting the Log Analytics Linux Syslog DataSource.

## Import

Log Analytics Linux Syslog DataSources can be imported using the `resource id`, e.g.

```shell
terraform import azurerm_log_analytics_datasource_linux_syslog.example /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.OperationalInsights/workspaces/workspace1/datasources/datasource1
```
