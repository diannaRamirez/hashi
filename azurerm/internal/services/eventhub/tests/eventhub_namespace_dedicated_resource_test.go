package tests

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func TestAccAzureRMEventHubNamespaceDedicated_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_requiresImport(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
				),
			},
			{
				Config:      testAccAzureRMEventHubNamespaceDedicated_requiresImport(data),
				ExpectError: acceptance.RequiresImportError("azurerm_eventhub_namespace_dedicated"),
			},
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_standard(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_standard(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_networkrule_iprule(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_networkrule_iprule(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_networkrule_vnet(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_networkrule_vnet(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_networkruleVnetIpRule(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_networkruleVnetIpRule(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "network_rulesets.0.virtual_network_rule.#", "2"),
					resource.TestCheckResourceAttr(data.ResourceName, "network_rulesets.0.ip_rule.#", "2"),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_readDefaultKeys(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
					resource.TestMatchResourceAttr(data.ResourceName, "default_primary_connection_string", regexp.MustCompile("Endpoint=.+")),
					resource.TestMatchResourceAttr(data.ResourceName, "default_secondary_connection_string", regexp.MustCompile("Endpoint=.+")),
					resource.TestMatchResourceAttr(data.ResourceName, "default_primary_key", regexp.MustCompile(".+")),
					resource.TestMatchResourceAttr(data.ResourceName, "default_secondary_key", regexp.MustCompile(".+")),
				),
			},
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_withAliasConnectionString(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				// `default_primary_connection_string_alias` and `default_secondary_connection_string_alias` are still `nil` in `azurerm_eventhub_namespace_dedicated` after created `azurerm_eventhub_namespace_dedicated` successfully since `azurerm_eventhub_namespace_dedicated_disaster_recovery_config` hasn't been created.
				// So these two properties should be checked in the second run.
				Config: testAccAzureRMEventHubNamespaceDedicated_withAliasConnectionString(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
				),
			},
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_withAliasConnectionString(data),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(data.ResourceName, "default_primary_connection_string_alias", regexp.MustCompile("Endpoint=.+")),
					resource.TestMatchResourceAttr(data.ResourceName, "default_secondary_connection_string_alias", regexp.MustCompile("Endpoint=.+")),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_maximumThroughputUnits(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_maximumThroughputUnits(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
				),
			},
			data.ImportStep(),
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_NonStandardCasing(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicatedNonStandardCasing(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists("azurerm_eventhub_namespace_dedicated.test"),
				),
			},
			{
				Config:             testAccAzureRMEventHubNamespaceDedicatedNonStandardCasing(data),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_BasicWithTagsUpdate(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
				),
			},
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_basicWithTagsUpdate(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "tags.%", "1"),
				),
			},
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_BasicWithCapacity(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_capacity(data, 20),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "capacity", "20"),
				),
			},
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_BasicWithCapacityUpdate(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_capacity(data, 20),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "capacity", "20"),
				),
			},
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_capacity(data, 2),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "capacity", "2"),
				),
			},
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_BasicWithSkuUpdate(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "sku", "Basic"),
				),
			},
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_standard(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "sku", "Standard"),
					resource.TestCheckResourceAttr(data.ResourceName, "capacity", "2"),
				),
			},
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_maximumThroughputUnitsUpdate(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_maximumThroughputUnits(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "sku", "Standard"),
					resource.TestCheckResourceAttr(data.ResourceName, "capacity", "2"),
					resource.TestCheckResourceAttr(data.ResourceName, "maximum_throughput_units", "20"),
				),
			},
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_maximumThroughputUnitsUpdate(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "sku", "Standard"),
					resource.TestCheckResourceAttr(data.ResourceName, "capacity", "1"),
					resource.TestCheckResourceAttr(data.ResourceName, "maximum_throughput_units", "1"),
				),
			},
		},
	})
}

func TestAccAzureRMEventHubNamespaceDedicated_autoInfalteDisabledWithAutoInflateUnits(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_eventhub_namespace_dedicated", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMEventHubNamespaceDedicatedDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMEventHubNamespaceDedicated_autoInfalteDisabledWithAutoInflateUnits(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMEventHubNamespaceDedicatedExists(data.ResourceName),
				),
			},
		},
	})
}

func testCheckAzureRMEventHubNamespaceDedicatedDestroy(s *terraform.State) error {
	conn := acceptance.AzureProvider.Meta().(*clients.Client).Eventhub.NamespacesClient
	ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_eventhub_namespace_dedicated" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(ctx, resourceGroup, name)

		if err != nil {
			if !utils.ResponseWasNotFound(resp.Response) {
				return err
			}
		}
	}

	return nil
}

func testCheckAzureRMEventHubNamespaceDedicatedExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acceptance.AzureProvider.Meta().(*clients.Client).Eventhub.NamespacesClient
		ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		namespaceName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Event Hub Namespace: %s", namespaceName)
		}

		resp, err := conn.Get(ctx, resourceGroup, namespaceName)
		if err != nil {
			if utils.ResponseWasNotFound(resp.Response) {
				return fmt.Errorf("Bad: Event Hub Namespace %q (resource group: %q) does not exist", namespaceName, resourceGroup)
			}

			return fmt.Errorf("Bad: Get on EventHubNamespaceDedicatedsClient: %+v", err)
		}

		return nil
	}
}

func testAccAzureRMEventHubNamespaceDedicated_basic(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                = "acctestEventHubNamespaceDedicated-%d"
  cluster_id          = azurerm_eventhub_cluster.test.id
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  sku                 = "Basic"
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func testAccAzureRMEventHubNamespaceDedicated_withAliasConnectionString(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-ehn-%[1]d"
  location = "%[2]s"
}

resource "azurerm_resource_group" "test2" {
  name     = "acctestRG2-ehn-%[1]d"
  location = "%[3]s"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                = "acctestEventHubNamespaceDedicated-%[1]d"
  cluster_id          = azurerm_eventhub_cluster.test.id
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name

  sku = "Standard"
}

resource "azurerm_eventhub_namespace_dedicated" "test2" {
  name                = "acctestEventHubNamespaceDedicated2-%[1]d"
  cluster_id          = azurerm_eventhub_cluster.test.id
  location            = azurerm_resource_group.test2.location
  resource_group_name = azurerm_resource_group.test2.name

  sku = "Standard"
}

resource "azurerm_eventhub_namespace_dedicated_disaster_recovery_config" "test" {
  name                 = "acctest-EHN-DRC-%[1]d"
  resource_group_name  = azurerm_resource_group.test.name
  cluster_id           = azurerm_eventhub_cluster.test.id
  namespace_name       = azurerm_eventhub_namespace_dedicated.test.name
  partner_namespace_id = azurerm_eventhub_namespace_dedicated.test2.id
}
`, data.RandomInteger, data.Locations.Primary, data.Locations.Secondary, data.RandomInteger)
}

func testAccAzureRMEventHubNamespaceDedicated_requiresImport(data acceptance.TestData) string {
	template := testAccAzureRMEventHubNamespaceDedicated_basic(data)
	return fmt.Sprintf(`
%s

resource "azurerm_eventhub_namespace_dedicated" "import" {
  name                = azurerm_eventhub_namespace_dedicated.test.name
  location            = azurerm_eventhub_namespace_dedicated.test.location
  resource_group_name = azurerm_eventhub_namespace_dedicated.test.resource_group_name
  sku                 = azurerm_eventhub_namespace_dedicated.test.sku
  cluster_id          = azurerm_eventhub_namespace_dedicated.test.cluster_id
}
`, template)
}

func testAccAzureRMEventHubNamespaceDedicated_standard(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                = "acctestEventHubNamespaceDedicated-%d"
  cluster_id          = azurerm_eventhub_cluster.test.id
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  sku                 = "Standard"
  capacity            = "2"
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func testAccAzureRMEventHubNamespaceDedicated_networkrule_iprule(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                = "acctestEventHubNamespaceDedicated-%d"
  cluster_id          = azurerm_eventhub_cluster.test.id
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  sku                 = "Standard"
  capacity            = "2"

  network_rulesets {
    default_action = "Deny"
    ip_rule {
      ip_mask = "10.0.0.0/16"
    }
  }
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func testAccAzureRMEventHubNamespaceDedicated_networkrule_vnet(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%[1]d"
  location = "%[2]s"
}

resource "azurerm_virtual_network" "test" {
  name                = "acctvn-%[1]d"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_subnet" "test" {
  name                 = "acctsub-%[1]d"
  resource_group_name  = azurerm_resource_group.test.name
  virtual_network_name = azurerm_virtual_network.test.name
  address_prefix       = "10.0.2.0/24"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%[1]d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                = "acctestEventHubNamespaceDedicated-%[1]d"
  cluster_id          = azurerm_eventhub_cluster.test.id
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  sku                 = "Standard"
  capacity            = "2"

  network_rulesets {
    default_action = "Deny"
    virtual_network_rule {
      subnet_id = azurerm_subnet.test.id

      ignore_missing_virtual_network_service_endpoint = true
    }
  }
}
`, data.RandomInteger, data.Locations.Primary)
}

func testAccAzureRMEventHubNamespaceDedicated_networkruleVnetIpRule(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%[1]d"
  location = "%[2]s"
}

resource "azurerm_virtual_network" "test" {
  name                = "acctvn1-%[1]d"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_subnet" "test" {
  name                 = "acctsub1-%[1]d"
  resource_group_name  = azurerm_resource_group.test.name
  virtual_network_name = azurerm_virtual_network.test.name
  address_prefix       = "10.0.1.0/24"
  service_endpoints    = ["Microsoft.EventHub"]
}

resource "azurerm_virtual_network" "test2" {
  name                = "acctvn2-%[1]d"
  address_space       = ["10.1.0.0/16"]
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_subnet" "test2" {
  name                 = "acctsub2-%[1]d"
  resource_group_name  = azurerm_resource_group.test.name
  virtual_network_name = azurerm_virtual_network.test2.name
  address_prefix       = "10.1.1.0/24"
  service_endpoints    = ["Microsoft.EventHub"]
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%[1]d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                = "acctestEventHubNamespaceDedicated-%[1]d"
  cluster_id          = azurerm_eventhub_cluster.test.id
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  sku                 = "Standard"
  capacity            = "2"

  network_rulesets {
    default_action = "Deny"

    virtual_network_rule {
      subnet_id = azurerm_subnet.test.id
    }

    virtual_network_rule {
      subnet_id = azurerm_subnet.test2.id
    }

    ip_rule {
      ip_mask = "10.0.1.0/24"
    }

    ip_rule {
      ip_mask = "10.1.1.0/24"
    }
  }
}
`, data.RandomInteger, data.Locations.Primary)
}

func testAccAzureRMEventHubNamespaceDedicatedNonStandardCasing(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                = "acctestEventHubNamespaceDedicated-%d"
  cluster_id          = azurerm_eventhub_cluster.test.id
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  sku                 = "basic"
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func testAccAzureRMEventHubNamespaceDedicated_maximumThroughputUnits(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                     = "acctestEventHubNamespaceDedicated-%d"
  cluster_id               = azurerm_eventhub_cluster.test.id
  location                 = azurerm_resource_group.test.location
  resource_group_name      = azurerm_resource_group.test.name
  sku                      = "Standard"
  capacity                 = "2"
  auto_inflate_enabled     = true
  maximum_throughput_units = 20
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func testAccAzureRMEventHubNamespaceDedicated_basicWithTagsUpdate(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                = "acctestEventHubNamespaceDedicated-%d"
  cluster_id          = azurerm_eventhub_cluster.test.id
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  sku                 = "Basic"

  tags = {
    environment = "Production"
  }
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func testAccAzureRMEventHubNamespaceDedicated_capacity(data acceptance.TestData, capacity int) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                = "acctestEventHubNamespaceDedicated-%d"
  cluster_id          = azurerm_eventhub_cluster.test.id
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  sku                 = "Basic"
  capacity            = %d
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger, capacity)
}

func testAccAzureRMEventHubNamespaceDedicated_maximumThroughputUnitsUpdate(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                     = "acctestEventHubNamespaceDedicated-%d"
  cluster_id               = azurerm_eventhub_cluster.test.id
  location                 = azurerm_resource_group.test.location
  resource_group_name      = azurerm_resource_group.test.name
  sku                      = "Standard"
  capacity                 = 1
  auto_inflate_enabled     = true
  maximum_throughput_units = 1
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func testAccAzureRMEventHubNamespaceDedicated_autoInfalteDisabledWithAutoInflateUnits(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_eventhub_cluster" "test" {
  name                = "acctesteventhubclusTER-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "Dedicated_1"
}

resource "azurerm_eventhub_namespace_dedicated" "test" {
  name                     = "acctestEventHubNamespaceDedicated-%d"
  cluster_id               = azurerm_eventhub_cluster.test.id
  location                 = azurerm_resource_group.test.location
  resource_group_name      = azurerm_resource_group.test.name
  sku                      = "Standard"
  capacity                 = 1
  auto_inflate_enabled     = false
  maximum_throughput_units = 0
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}
