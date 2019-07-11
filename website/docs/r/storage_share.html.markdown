---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_storage_share"
sidebar_current: "docs-azurerm-resource-storage-share-x"
description: |-
  Manages a File Share within Azure Storage.
---

# azurerm_storage_share

Manages a File Share within Azure Storage.

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "azuretest"
  location = "West Europe"
}

resource "azurerm_storage_account" "test" {
  name                     = "azureteststorage"
  resource_group_name      = "${azurerm_resource_group.test.name}"
  location                 = "${azurerm_resource_group.test.location}"
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

resource "azurerm_storage_share" "testshare" {
  name                 = "sharename"
  storage_account_name = "${azurerm_storage_account.test.name}"
  quota                = 50
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the share. Must be unique within the storage account where the share is located.

* `storage_account_name` - (Required) Specifies the storage account in which to create the share.
 Changing this forces a new resource to be created.

* `resource_group_name` - (Optional / **Deprecated**) The name of the resource group in which to
    create the share. Changing this forces a new resource to be created.

* `quota` - (Optional) The maximum size of the share, in gigabytes. Must be greater than 0, and less than or equal to 5 TB (5120 GB). Default is 5120.


## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `id` - The ID of the File Share.
* `url` - The URL of the File Share

## Import

Storage Shares can be imported using the `resource id`, e.g.

```shell
terraform import azurerm_storage_share.testShare https://account1.file.core.windows.net/share1
```
