---
subcategory: "API Management"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_api_management_custom_domain"
description: |-
  Manages a API Management Custom Domain.
---

# azurerm_api_management_custom_domain

Manages a API Management Custom Domain.

## Disclaimers

~> **Note:** It's possible to define Custom Domains both within [the `azurerm_api_management` resource](api_management.html) via the `hostname_configurations` block and by using [the `azurerm_api_management_custom_domain` resource](api_management_custom_domain.html). However it's not possible to use both methods to manage Custom Domains within an API Management Service, since there'll be conflicts.

## Example Usage

```hcl
resource "azurerm_key_vault" "example" {
  // TODO
}

resource "azurerm_key_vault_certificate" "example" {
  key_vault_id = azurerm_key_vault.example.id
  // TODO
}

resource "azurerm_api_management_custom_domain" "example" {
  resource_group_name = "example"
  api_management_name = "example"
  proxy {
    host_name    = "api.example.com"
    key_vault_id = azurerm_key_vault_certificate.example.secret_id
  }
}
```

## Arguments Reference

The following arguments are supported:

* `api_management_name` - (Required) TODO. Changing this forces a new API Management Custom Domain to be created.

* `resource_group_name` - (Required) The name of the Resource Group where the API Management Custom Domain should exist. Changing this forces a new API Management Custom Domain to be created.

---

* `developer_portal` - (Optional) One or more `developer_portal` blocks as defined below.

* `management` - (Optional) One or more `management` blocks as defined below.

* `portal` - (Optional) One or more `portal` blocks as defined below.

* `proxy` - (Optional) One or more `proxy` blocks as defined below.

* `scm` - (Optional) One or more `scm` blocks as defined below.

---

A `developer_portal` block supports the following:

* `host_name` - (Required) The Hostname to use for the Developer Portal.

* `certificate` - (Optional) The Base64 Encoded Certificate. (Mutually exlusive with `key_vault_id`.)

* `certificate_password` - (Optional) The password associated with the certificate provided above.

* `key_vault_id` - (Optional) The ID of the Key Vault Secret containing the SSL Certificate, which must be should be of the type application/x-pkcs12.

* `negotiate_client_certificate` - (Optional) Should Client Certificate Negotiation be enabled for this Hostname? Defaults to false.

---

A `management` block supports the following:

* `host_name` - (Required) The Hostname to use for the Management API.

* `certificate` - (Optional) The Base64 Encoded Certificate. (Mutually exlusive with `key_vault_id`.)

* `certificate_password` - (Optional) The password associated with the certificate provided above.

* `key_vault_id` - (Optional) The ID of the Key Vault Secret containing the SSL Certificate, which must be should be of the type application/x-pkcs12.

* `negotiate_client_certificate` - (Optional) Should Client Certificate Negotiation be enabled for this Hostname? Defaults to false.

---

A `portal` block supports the following:

* `host_name` - (Required) The Hostname to use for the legacy Developer Portal.

* `certificate` - (Optional) The Base64 Encoded Certificate. (Mutually exlusive with `key_vault_id`.)

* `certificate_password` - (Optional) The password associated with the certificate provided above.

* `key_vault_id` - (Optional) The ID of the Key Vault Secret containing the SSL Certificate, which must be should be of the type application/x-pkcs12.

* `negotiate_client_certificate` - (Optional) Should Client Certificate Negotiation be enabled for this Hostname? Defaults to false.

---

A `proxy` block supports the following:

* `host_name` - (Required) The Hostname to use for the legacy Developer Portal.

* `certificate` - (Optional) The Base64 Encoded Certificate. (Mutually exlusive with `key_vault_id`.)

* `certificate_password` - (Optional) The password associated with the certificate provided above.

* `default_ssl_binding` - (Optional) Is the certificate associated with this Hostname the Default SSL Certificate? This is used when an SNI header isn't specified by a client. Defaults to false.

* `key_vault_id` - (Optional) The ID of the Key Vault Secret containing the SSL Certificate, which must be should be of the type application/x-pkcs12.

* `negotiate_client_certificate` - (Optional) Should Client Certificate Negotiation be enabled for this Hostname? Defaults to false.

---

A `scm` block supports the following:

* `host_name` - (Required) The Hostname to use for the SCM domain.

* `certificate` - (Optional) The Base64 Encoded Certificate. (Mutually exlusive with `key_vault_id`.)

* `certificate_password` - (Optional) The password associated with the certificate provided above.

* `key_vault_id` - (Optional) The ID of the Key Vault Secret containing the SSL Certificate, which must be should be of the type application/x-pkcs12.

* `negotiate_client_certificate` - (Optional) Should Client Certificate Negotiation be enabled for this Hostname? Defaults to false.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported: 

* `id` - The ID of the API Management Custom Domain.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the API Management Custom Domain.
* `read` - (Defaults to 5 minutes) Used when retrieving the API Management Custom Domain.
* `update` - (Defaults to 30 minutes) Used when updating the API Management Custom Domain.
* `delete` - (Defaults to 30 minutes) Used when deleting the API Management Custom Domain.

## Import

API Management Custom Domains can be imported using the `resource id`, e.g.

```shell
terraform import azurerm_api_management_custom_domain.example /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/Microsoft.ApiManagement/service/instance1
```
