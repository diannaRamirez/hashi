---
subcategory: "Database"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_postgresql_flexible_server"
description: |-
  Manages a PostgreSQL Flexible Server.
---

# azurerm_postgresql_flexible_server

Manages a PostgreSQL Flexible Server.

## Example Usage

```hcl
resource "azurerm_postgresql_flexible_server" "example" {
  name                         = "example-psqlflexibleserver"
  resource_group_name          = azurerm_resource_group.test.name
  location                     = azurerm_resource_group.test.location
  version                      = "12"

  administrator_login          = "psqladminun"
  administrator_login_password = "H@Sh1CoR3!"

  storage_mb = 32768

  sku {
    name = "Standard_D4s_v3"
    tier = "GeneralPurpose"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the PostgreSQL Flexible Server. Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to create the PostgreSQL Flexible Server. Changing this forces a new resource to be created.

* `location` - (Required) Specifies the supported Azure location where the resource exists. Changing this forces a new resource to be created.

* `administrator_login` - (Required) The Administrator Login for the PostgreSQL Flexible Server. Required when `create_mode` is `Default`. Changing this forces a new resource to be created.

* `administrator_login_password` - (Required) The Password associated with the `administrator_login` for the PostgreSQL Flexible Server.

* `sku` - (Required) A `sku` block as defined below.

* `version` - (Required) Specifies the version of PostgreSQL Flexible Server to use. Valid values are `11` or `12`. Changing this forces a new resource to be created.

* `backup_retention_days` - (Optional) Backup retention days for the server, supported values are between `7` and `35` days.

* `create_mode` - (Optional) The creation mode. Can be used to restore or replicate existing servers. Possible values are `Default` or `PointInTimeRestore`. Defaults to `Default`. Changing this forces a new resource to be created.

* `restore_point_in_time` - (Optional) When `create_mode` is `PointInTimeRestore` this designates the point in time to restore from `creation_source_server_id`. 

* `delegated_subnet_resource_id` - (Optional) Create a PostgreSQL Flexible Server using an already existing virtual network subnet. The provided subnet should not have any other resource deployed in it and this subnet will be delegated to the PostgreSQL Flexible Server, if not already delegated. Changing this forces a new resource to be created.

* `display_name` - (Optional) The display name of a server. Changing this forces a new resource to be created.

* `identity` - (Optional) An `identity` block as defined below. . Changing this forces a new resource to be created.

* `point_in_time_utc` - (Optional) Restore point creation time (ISO8601 format), specifying the time to restore from. Changing this forces a new resource to be created.

* `source_server_name` - (Optional) The source PostgreSQL Flexible Server name to restore from. Changing this forces a new resource to be created.

* `ha_enabled` - (Optional) Enable High availability for the PostgreSQL Flexible Server. If enalbed the server will provisions a physically separate primary and standby PostgreSQL Flexible Server in different zones. Possible values include `true` or `false`. Defaults to `false`.

* `maintenance_window` - (Optional) A `maintenance_window` block as defined below.

~> **NOTE:** A `maintenance_window` block **cannot** be included in the configuration file during the intial creation of the PostgreSQL Flexible Server. If you do include the `maintenance_window` block during the intial creation of the PostgreSQL Flexible Server it will cause an unclear `Resource Not Found` error messages.

* `backup_retention_days` - (Optional) Number of days to retain Backups for the server. Possible values are between `7` and `35` inclusive. Defaults to `7`.

* `storage_mb` - (Optional) Max storage allowed for a server. Possible values are `32768`, `65536`, `131072`, `262144`, `524288`, `1048576`, `2097152`, `4194304`, `8388608`, `16777216`, and `33554432`. Defaults to `32768`.

* `properties_tags` - (Optional) A mapping of tags to assign to the resource.

* `tags` - (Optional) A mapping of tags to assign to the resource.

---

A `identity` block supports the following:

* `type` - (Required) The Type of Identity which should be used for this PostgreSQL Flexible Server. At this time the only possible value is `SystemAssigned`.

---

A `sku` block supports the following:

* `name` - (Required) Specifies the SKU Name for the PostgreSQL Flexible Server. Possible values are `Standard_B1ms`, `Standard_B2s`, `Standard_D2s_v3`, `Standard_D4s_v3`, `Standard_D8s_v3`, `Standard_D16s_v3`, `Standard_D32s_v3`, `Standard_D48s_v3`, `Standard_D64s_v3`, `Standard_E2s_v3`, `Standard_E4s_v3`, `Standard_E8s_v3`, `Standard_E16s_v3`, `Standard_E32s_v3`, `Standard_E48s_v3`, `Standard_E64s_v3`.

* `tier` - (Required) Specifies the SKU tier for the PostgreSQL Flexible Server. Possible values are `Burstable`, `GeneralPurpose`, or `MemoryOptimized`.

---

A `maintenance_window` block supports the following:

* `enabled` - (Optional) indicates whether the maintenance window is `enabled` or `disabled`. Defaults to `false`.

* `day_of_week` - (Optional) day of week for maintenance window. Defaults to `0`.

* `start_hour` - (Optional) start hour for maintenance window. Defaults to `0`.

* `start_minute` - (Optional) start minute for maintenance window. Defaults to `0`.

---

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the PostgreSQL Flexible Server.

* `byok_enforcement` - The status showing whether the data encryption is enabled with a customer-managed key.

* `ha_state` - The state of the High Availability server. Possible values include: `NotEnabled`, `CreatingStandby`, `ReplicatingData`, `FailingOver`, `Healthy`, and `RemovingStandby`.

* `public_network_access` - Is public network access enabled?

* `standby_availability_zone` - The availability Zone information of the server.

* `fqdn` - The FQDN of the PostgreSQL Flexible Server.

* `identity` - An `identity` block as documented below.

---

A `identity` block exports the following:

* `principal_id` - The Client ID of the Service Principal assigned to this PostgreSQL Flexible Server.

* `tenant_id` - The ID of the Tenant the Service Principal is assigned in.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 60 minutes) Used when creating the PostgreSQL Flexible Server.
* `update` - (Defaults to 60 minutes) Used when updating the PostgreSQL Flexible Server.
* `read` - (Defaults to 5 minutes) Used when retrieving the PostgreSQL Flexible Server.
* `delete` - (Defaults to 60 minutes) Used when deleting the PostgreSQL Flexible Server.

## Import

PostgreSQL Flexible Server's can be imported using the `resource id`, e.g.

```shell
terraform import azurerm_postgresql_flexible_server.server1 /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.DBforPostgreSQL/flexibleServers/server1
```
