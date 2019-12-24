package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
)

func TestAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskCaching(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_windows_virtual_machine_scale_set", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMWindowsVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskCaching(data, "None"),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			), {
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskCaching(data, "ReadOnly"),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			), {
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskCaching(data, "ReadWrite"),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			),
		},
	})
}

func TestAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskCustomSize(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_windows_virtual_machine_scale_set", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMWindowsVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				// unset
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_authPassword(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			),
			{
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskCustomSize(data, 128),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			),
			{
				// resize a second time to confirm https://github.com/Azure/azure-rest-api-specs/issues/1906
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskCustomSize(data, 256),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			),
		},
	})
}

func TestAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskEphemeral(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_windows_virtual_machine_scale_set", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMWindowsVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskEphemeral(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			),
		},
	})
}

func TestAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskStorageAccountTypeStandardLRS(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_windows_virtual_machine_scale_set", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMWindowsVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskStorageAccountType(data, "Standard_LRS"),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			),
		},
	})
}

func TestAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskStorageAccountTypeStandardSSDLRS(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_windows_virtual_machine_scale_set", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMWindowsVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskStorageAccountType(data, "StandardSSD_LRS"),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			),
		},
	})
}

func TestAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskStorageAccountTypePremiumLRS(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_windows_virtual_machine_scale_set", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMWindowsVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskStorageAccountType(data, "Premium_LRS"),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			),
		},
	})
}

func TestAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskWriteAcceleratorEnabled(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_windows_virtual_machine_scale_set", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMWindowsVirtualMachineScaleSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskWriteAcceleratorEnabled(data, true),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMWindowsVirtualMachineScaleSetExists(data.ResourceName),
				),
			},
			data.ImportStep(
				"admin_password",
				"terraform_should_roll_instances_when_required",
			),
		},
	})
}

func testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskCaching(data acceptance.TestData, caching string) string {
	template := testAccAzureRMWindowsVirtualMachineScaleSet_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_windows_virtual_machine_scale_set" "test" {
  name                = local.vm_name
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku                 = "Standard_F2"
  instances           = 1
  admin_username      = "adminuser"
  admin_password      = "P@ssword1234!"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type = "Standard_LRS"
    caching              = "%s"
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurerm_subnet.test.id
    }
  }
}
`, template, caching)
}

func testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskCustomSize(data acceptance.TestData, diskSize int) string {
	template := testAccAzureRMWindowsVirtualMachineScaleSet_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_windows_virtual_machine_scale_set" "test" {
  name                = local.vm_name
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku                 = "Standard_F2"
  instances           = 1
  admin_username      = "adminuser"
  admin_password      = "P@ssword1234!"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type = "Standard_LRS"
    caching              = "ReadWrite"
    disk_size_gb         = %d
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurerm_subnet.test.id
    }
  }
}
`, template, diskSize)
}

func testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskEphemeral(data acceptance.TestData) string {
	template := testAccAzureRMWindowsVirtualMachineScaleSet_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_windows_virtual_machine_scale_set" "test" {
  name                = local.vm_name
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku                 = "Standard_F8s_v2" # has to be this large for ephemeral disks on Windows
  instances           = 1
  admin_username      = "adminuser"
  admin_password      = "P@ssword1234!"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type = "Standard_LRS"
    caching              = "ReadOnly"

    diff_disk_settings {
      option = "Local"
    }
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurerm_subnet.test.id
    }
  }
}
`, template)
}

func testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskStorageAccountType(data acceptance.TestData, storageAccountType string) string {
	template := testAccAzureRMWindowsVirtualMachineScaleSet_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_windows_virtual_machine_scale_set" "test" {
  name                = local.vm_name
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku                 = "Standard_F2s_v2"
  instances           = 1
  admin_username      = "adminuser"
  admin_password      = "P@ssword1234!"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type = %q
    caching              = "ReadWrite"
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurerm_subnet.test.id
    }
  }
}
`, template, storageAccountType)
}

func testAccAzureRMWindowsVirtualMachineScaleSet_disksOSDiskWriteAcceleratorEnabled(data acceptance.TestData, enabled bool) string {
	template := testAccAzureRMWindowsVirtualMachineScaleSet_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_windows_virtual_machine_scale_set" "test" {
  name                = local.vm_name
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku                 = "Standard_M8ms"
  instances           = 1
  admin_username      = "adminuser"
  admin_password      = "P@ssword1234!"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type      = "Premium_LRS"
    caching                   = "None"
    write_accelerator_enabled = %t
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurerm_subnet.test.id
    }
  }
}
`, template, enabled)
}
