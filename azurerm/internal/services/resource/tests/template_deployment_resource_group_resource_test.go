package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
)

func TestAccResourceGroupTemplateDeployment_empty(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_emptyConfig(data, "Complete"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				// set some tags
				Config: resourceGroupTemplateDeployment_emptyWithTagsConfig(data, "Complete"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccResourceGroupTemplateDeployment_incremental(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_emptyConfig(data, "Incremental"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				// set some tags
				Config: resourceGroupTemplateDeployment_emptyWithTagsConfig(data, "Incremental"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccResourceGroupTemplateDeployment_singleItemUpdatingParams(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_singleItemWithParameterConfig(data, "first"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				Config: resourceGroupTemplateDeployment_singleItemWithParameterConfig(data, "second"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccResourceGroupTemplateDeployment_singleItemUpdatingTemplate(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_singleItemWithPublicIPConfig(data, "first"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				Config: resourceGroupTemplateDeployment_singleItemWithPublicIPConfig(data, "second"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccResourceGroupTemplateDeployment_withOutputs(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_withOutputsConfig(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "output_content", "{\"testOutput\":{\"type\":\"String\",\"value\":\"some-value\"}}"),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccResourceGroupTemplateDeployment_multipleItems(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_multipleItemsConfig(data, "first"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				Config: resourceGroupTemplateDeployment_multipleItemsConfig(data, "second"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccResourceGroupTemplateDeployment_multipleNestedItems(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_multipleNestedItemsConfig(data, "first"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				Config: resourceGroupTemplateDeployment_multipleNestedItemsConfig(data, "second"),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccResourceGroupTemplateDeployment_childItems(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_childItemsConfig(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				Config: resourceGroupTemplateDeployment_childItemsConfig(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccResourceGroupTemplateDeployment_updateOnErrorDeployment(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_withLastSuccessfulDeploymentConfig(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				Config: resourceGroupTemplateDeployment_withSpecificDeploymentConfig(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccResourceGroupTemplateDeployment_switchTemplateDeploymentBetweenLinkAndContent(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_withTemplateLinkAndParametersLinkConfig(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				Config: resourceGroupTemplateDeployment_withDeploymentContents(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				Config: resourceGroupTemplateDeployment_withTemplateLinkAndParametersLinkConfig(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccResourceGroupTemplateDeployment_updateTemplateLinkAndParametersLink(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group_template_deployment", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckResourceGroupTemplateDeploymentDestroyed,
		Steps: []resource.TestStep{
			{
				Config: resourceGroupTemplateDeployment_withTemplateLinkAndParametersLinkConfig(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
			{
				Config: resourceGroupTemplateDeployment_updateTemplateLinkAndParametersLinkConfig(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGroupTemplateDeploymentExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func testCheckResourceGroupTemplateDeploymentExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.AzureProvider.Meta().(*clients.Client).Resource.DeploymentsClient
		ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]
		resp, err := client.Get(ctx, resourceGroup, name)
		if err != nil {
			return fmt.Errorf("bad: Get on deploymentsClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("bad: Resource Group Template Deployment %q does not exist", name)
		}

		return nil
	}
}

func testCheckResourceGroupTemplateDeploymentDestroyed(s *terraform.State) error {
	client := acceptance.AzureProvider.Meta().(*clients.Client).Resource.DeploymentsClient
	ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_resource_group_template_deployment" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := client.Get(ctx, resourceGroup, name)
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Resource Group Template Deployment still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

func resourceGroupTemplateDeployment_emptyConfig(data acceptance.TestData, deploymentMode string) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = %q
}

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = %q

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": []
}
TEMPLATE
}
`, data.RandomInteger, data.Locations.Primary, deploymentMode)
}

func resourceGroupTemplateDeployment_emptyWithTagsConfig(data acceptance.TestData, deploymentMode string) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = %q
}

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = %q
  tags = {
    Hello = "World"
  }

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": []
}
TEMPLATE
}
`, data.RandomInteger, data.Locations.Primary, deploymentMode)
}

func resourceGroupTemplateDeployment_singleItemWithParameterConfig(data acceptance.TestData, value string) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = %q
}

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "someParam": {
      "type": "String",
      "allowedValues": [
        "first",
        "second",
        "third"
      ]
    }
  },
  "variables": {},
  "resources": []
}
TEMPLATE

  parameters_content = <<PARAM
{
  "someParam": {
   "value": %q
  }
}
PARAM
}
`, data.RandomInteger, data.Locations.Primary, value)
}

func resourceGroupTemplateDeployment_singleItemWithPublicIPConfig(data acceptance.TestData, tagValue string) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = %q
}

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": [
    {
      "type": "Microsoft.Network/publicIPAddresses",
      "apiVersion": "2015-06-15",
      "name": "acctestpip-%d",
      "location": "[resourceGroup().location]",
      "properties": {
        "publicIPAllocationMethod": "Dynamic"
      },
      "tags": {
        "Hello": %q
      }
    }
  ]
}
TEMPLATE
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, tagValue)
}

func resourceGroupTemplateDeployment_withOutputsConfig(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = %q
}

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": [],
  "outputs": {
    "testOutput": {
      "type": "String",
      "value": "some-value"
    }
  }
}
TEMPLATE
}
`, data.RandomInteger, data.Locations.Primary)
}

func resourceGroupTemplateDeployment_multipleItemsConfig(data acceptance.TestData, value string) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = %q
}

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": [
    {
      "type": "Microsoft.Network/publicIPAddresses",
      "apiVersion": "2015-06-15",
      "name": "acctestpip-1-%d",
      "location": "[resourceGroup().location]",
      "properties": {
        "publicIPAllocationMethod": "Dynamic"
      },
      "tags": {
        "Hello": %q
      }
    },
    {
      "type": "Microsoft.Network/publicIPAddresses",
      "apiVersion": "2015-06-15",
      "name": "acctestpip-2-%d",
      "location": "[resourceGroup().location]",
      "properties": {
        "publicIPAllocationMethod": "Dynamic"
      },
      "tags": {
        "Hello": %q
      }
    }
  ]
}
TEMPLATE
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, value, data.RandomInteger, value)
}

func resourceGroupTemplateDeployment_multipleNestedItemsConfig(data acceptance.TestData, value string) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = %q
}

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": [
    {
      "type": "Microsoft.Network/virtualNetworks",
      "apiVersion": "2020-05-01",
      "name": "parent-network",
      "location": "[resourceGroup().location]",
      "properties": {
        "addressSpace": {
          "addressPrefixes": [
            "10.0.0.0/16"
          ]
        }
      },
      "tags": {
        "Hello": %q
      },
      "resources": [
        {
          "type": "subnets",
          "apiVersion": "2020-05-01",
          "location": "[resourceGroup().location]",
          "name": "first",
          "dependsOn": [
            "parent-network"
          ],
          "properties": {
            "addressPrefix": "10.0.1.0/24"
          }
        },
        {
          "type": "subnets",
          "apiVersion": "2020-05-01",
          "location": "[resourceGroup().location]",
          "name": "second",
          "dependsOn": [
            "parent-network",
            "first"
          ],
          "properties": {
            "addressPrefix": "10.0.2.0/24"
          }
        }
      ]
    }
  ]
}
TEMPLATE
}
`, data.RandomInteger, data.Locations.Primary, value)
}

func resourceGroupTemplateDeployment_childItemsConfig(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestrg-%d"
  location = %q
}

resource "azurerm_route_table" "test" {
  name                = "acctestrt%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Incremental"

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": [
    {
      "type": "Microsoft.Network/routeTables/routes",
      "apiVersion": "2020-06-01",
      "name": "${azurerm_route_table.test.name}/child-route",
      "location": "[resourceGroup().location]",
      "properties": {
        "addressPrefix": "10.2.0.0/16",
        "nextHopType": "none"
      }
    }
  ]
}
TEMPLATE
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger)
}

func resourceGroupTemplateDeployment_withTemplateLinkAndParametersLinkConfig(data acceptance.TestData) string {
	template := resourceGroupTemplateDeployment_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest-UpdatedDeployment-%d"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  template_link {
    uri             = "https://raw.githubusercontent.com/Azure/azure-quickstart-templates/master/101-storage-account-create/azuredeploy.json"
    content_version = "1.0.0.0"
  }

  parameters_link {
    uri             = "https://raw.githubusercontent.com/Azure/azure-quickstart-templates/master/101-storage-account-create/azuredeploy.parameters.json"
    content_version = "1.0.0.0"
  }
}
`, template, data.RandomInteger)
}

func resourceGroupTemplateDeployment_updateTemplateLinkAndParametersLinkConfig(data acceptance.TestData) string {
	template := resourceGroupTemplateDeployment_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest-UpdatedDeployment-%d"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  template_link {
    uri = "https://raw.githubusercontent.com/Azure/azure-quickstart-templates/master/101-vnet-two-subnets/azuredeploy.json"
  }

  parameters_link {
    uri = "https://raw.githubusercontent.com/Azure/azure-quickstart-templates/master/101-vnet-two-subnets/azuredeploy.parameters.json"
  }
}
`, template, data.RandomInteger)
}

func resourceGroupTemplateDeployment_withDeploymentContents(data acceptance.TestData) string {
	template := resourceGroupTemplateDeployment_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest-UpdatedDeployment-%d"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "someParam": {
      "type": "String",
      "allowedValues": [
        "first",
        "second",
        "third"
      ]
    }
  },
  "variables": {},
  "resources": []
}
TEMPLATE

  parameters_content = <<PARAM
{
  "someParam": {
   "value": "first"
  }
}
PARAM
}
`, template, data.RandomInteger)
}

func resourceGroupTemplateDeployment_withLastSuccessfulDeploymentConfig(data acceptance.TestData) string {
	template := resourceGroupTemplateDeployment_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_resource_group_template_deployment" "test2" {
  name                = "acctest-Deployment-%d"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": []
}
TEMPLATE
}

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest-UpdatedDeployment-%d"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  on_error_deployment {
    type = "LastSuccessful"
  }

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": []
}
TEMPLATE

  depends_on = [azurerm_resource_group_template_deployment.test2]
}
`, template, data.RandomInteger, data.RandomInteger)
}

func resourceGroupTemplateDeployment_withSpecificDeploymentConfig(data acceptance.TestData) string {
	template := resourceGroupTemplateDeployment_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_resource_group_template_deployment" "test2" {
  name                = "acctest-Deployment-%d"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": []
}
TEMPLATE
}

resource "azurerm_resource_group_template_deployment" "test" {
  name                = "acctest-UpdatedDeployment-%d"
  resource_group_name = azurerm_resource_group.test.name
  deployment_mode     = "Complete"

  on_error_deployment {
    deployment_name = azurerm_resource_group_template_deployment.test2.name
    type            = "SpecificDeployment"
  }

  template_content = <<TEMPLATE
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {},
  "variables": {},
  "resources": []
}
TEMPLATE
}
`, template, data.RandomInteger, data.RandomInteger)
}

func resourceGroupTemplateDeployment_template(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-deployment-%d"
  location = "%s"
}
`, data.RandomInteger, data.Locations.Primary)
}
