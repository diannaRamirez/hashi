---
subcategory: "CosmosDB (DocumentDB)"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_cosmosdb_sql_container"
description: |-
  Manages a SQL Container within a Cosmos DB Account.
---

# azurerm_cosmosdb_sql_container

Manages a SQL Container within a Cosmos DB Account.

## Example Usage

```hcl

resource "azurerm_cosmosdb_sql_container" "example" {
  name                = "example-container"
  resource_group_name = "${azurerm_cosmosdb_account.example.resource_group_name}"
  account_name        = "${azurerm_cosmosdb_account.example.name}"
  database_name       = "${azurerm_cosmosdb_sql_database.example.name}"
  partition_key_path  = "/definition/id"
  throughput          = 400

  unique_key {
    paths = ["/definition/idlong", "/definition/idshort"]
  }

  indexing_policy {
    indexing_mode = "consistent"
    automatic = true
    included_paths = [
      {
        path = "/test/?"
        indexes = []
      }
    ]
    excluded_paths = [
      { path = "/*" },
		  { path = "/\"_etag\"/?"}
    ]
  }

}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the Cosmos DB SQL Database. Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which the Cosmos DB SQL Database is created. Changing this forces a new resource to be created.

* `account_name` - (Required) The name of the Cosmos DB Account to create the container within. Changing this forces a new resource to be created.

* `database_name` - (Required) The name of the Cosmos DB SQL Database to create the container within. Changing this forces a new resource to be created.

* `partition_key_path` - (Optional) Define a partition key. Changing this forces a new resource to be created.

* `unique_key` - (Optional) One or more `unique_key` blocks as defined below. Changing this forces a new resource to be created.

* `throughput` - (Optional) The throughput of SQL container (RU/s). Must be set in increments of `100`. The minimum value is `400`. This must be set upon database creation otherwise it cannot be updated without a manual terraform destroy-apply.

* `default_ttl` - (Optional) The default time to live of SQL container. If missing, items are not expired automatically. If present and the value is set to `-1`, it is equal to infinity, and items don’t expire by default. If present and the value is set to some number `n` – items will expire `n` seconds after their last modified time.

* `indexing_policy` - (Optional) The indexing policy block as defined below.

---
A `unique_key` block supports the following:

* `paths` - (Required) A list of paths to use for this unique key.

---
A `indexing_policy` block supports the following: 

* `indexing_mode` - (Optional) Indexing Mode. Default is set to `consistent`. Can be set to `None`
* `automatic` - (Optional) Boolean. Default value is true. This allow Azure CosmosDB to automatically index documents as they are written.
* `included_path` - (Optional) Block as defined below
* `path`: path to include. If `/*` is set in included path, it can't be set in excluded path
* `indexes` : block as defined below. Can be empty. 
* `data_type` : can be either `String` or `Number`
* `precision` : Is a number defined at the index level for included paths. A value of `-1` indicates maximum precision. Recommanded to always use `-1`
* `kind` : can be either `range` or `hash` (default: range)
* `excluded_path` - (Optional) Bloak as defined below :
* `path`: path to exclude. If `/*` is set in excluded path, it can't be set in included path
---


## Attributes Reference

The following attributes are exported:

* `id` - the Cosmos DB SQL Database ID.

## Import

Cosmos SQL Database can be imported using the `resource id`, e.g.

```shell
terraform import azurerm_cosmosdb_sql_container.example /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg1/providers/Microsoft.DocumentDB/databaseAccounts/account1/apis/sql/databases/database1/containers/example
```

