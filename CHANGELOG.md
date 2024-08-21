## 4.0.0 (Unreleased)

NOTES:

* **Major Version**: Version 4.0 of the Azure Provider is a major version - some behaviours have changed and some deprecated fields/resources have been removed - please refer to [the 4.0 upgrade guide for more information](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/4.0-upgrade-guide).
* When upgrading to v4.0 of the AzureRM Provider, we recommend upgrading to the latest version of Terraform Core ([which can be found here](https://www.terraform.io/downloads)).

ENHANCEMENTS:

* Data Source: `azurerm_shared_image` - add support for the `trusted_launch_supported`, `trusted_launch_enabled`, `confidential_vm_supported`, `confidential_vm_enabled`, `accelerated_network_support_enabled` and `hibernation_enabled` properties [GH-26975]
* dependencies: updating `hashicorp/go-azure-sdk` to `v0.20240819.1075239` [GH-27107]
* `applicationgateways` - updating to use `2023-11-01` [GH-26776]
* `containerregistry` - updating to use `2023-06-01-preview` [GH-23393]
* `containerservice` - updating to `2024-05-01` [GH-27105]
* `mssql` - updating to use `hashicorp/go-azure-sdk` and `023-08-01-preview` [GH-27073]
* `mssqlmanagedinstance` - updating to use `hashicorp/go-azure-sdk` and `2023-08-01-preview` [GH-26872]
* `azurerm_image` - add support for the `disk_encryption_set_id` property to the `data_disk` block [GH-27015]
* `azurerm_log_analytics_workspace_table` - add support for more `total_retention_in_days` and `retention_in_days` values [GH-27053]
* `azurerm_mssql_elasticpool` - add support for the `HS_MOPRMS` and `MOPRMS` skus [GH-27085]
* `azurerm_netapp_pool` - allow `1` as a valid value for `size_in_tb` [GH-27095]
* `azurerm_notification_hub` - add support for the `browser_credential` property [GH-27058]
* `azurerm_redis_cache` - add support for the `access_keys_authentication_enabled` property [GH-27039]
* `azurerm_role_assignment` - add support for the `/`, `/providers/Microsoft.Capacity` and `/providers/Microsoft.BillingBenefits` scopes [GH-26663]
* `azurerm_shared_image` - add support for the `hibernation_enabled` property [GH-26975]
* `azurerm_storage_account` - support `queue_encryption_key_type` and `table_encryption_key_type` for more storage account kinds [GH-27112]
* `azurerm_web_application_firewall_policy` - add support for the `request_body_enforcement` property [GH-27094]

BUG FIXES:

* `azurerm_ip_group_cidr` - fixed the position of the CIDR check to correctly refresh the resource when it's no longer present [GH-27103]
* `azurerm_monitor_diagnostic_setting` - add further polling to work around an eventual consistency issue when creating the resource [GH-27088]
* `azurerm_storage_account` - prevent API error by populating `infrastructure_encryption_enabled` when updating `customer_managed_key` [GH-26971]
* `azurerm_virtual_network_dns_servers` - moved locks to prevent the creation of subnets with stale data [GH-27036]
* `azurerm_virtual_network_gateway_connection` - allow `0` as a valid value for `ipsec_policy.sa_datasize` [GH-27056]

---

For information on changes between the v3.116.0 and v3.0.0 releases, please see [the previous v3.x changelog entries](https://github.com/hashicorp/terraform-provider-azurerm/blob/main/CHANGELOG-v3.md).

For information on changes between the v2.99.0 and v2.0.0 releases, please see [the previous v2.x changelog entries](https://github.com/hashicorp/terraform-provider-azurerm/blob/main/CHANGELOG-v2.md).

For information on changes between the v1.44.0 and v1.0.0 releases, please see [the previous v1.x changelog entries](https://github.com/hashicorp/terraform-provider-azurerm/blob/main/CHANGELOG-v1.md).

For information on changes prior to the v1.0.0 release, please see [the v0.x changelog](https://github.com/hashicorp/terraform-provider-azurerm/blob/main/CHANGELOG-v0.md).