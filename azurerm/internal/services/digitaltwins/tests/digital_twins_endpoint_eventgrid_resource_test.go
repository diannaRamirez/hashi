package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/digitaltwins/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func TestAccAzureRMDigitalTwinsEndpointEventGrid_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_digital_twins_endpoint_eventgrid", "test")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMDigitalTwinsEndpointEventGridDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMDigitalTwinsEndpointEventGrid_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDigitalTwinsEndpointEventGridExists(data.ResourceName),
				),
			},
			data.ImportStep("eventgrid_topic_endpoint", "eventgrid_topic_primary_access_key", "eventgrid_topic_secondary_access_key"),
		},
	})
}

func TestAccAzureRMDigitalTwinsEndpointEventGrid_requiresImport(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_digital_twins_endpoint_eventgrid", "test")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMDigitalTwinsEndpointEventGridDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMDigitalTwinsEndpointEventGrid_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDigitalTwinsEndpointEventGridExists(data.ResourceName),
				),
			},
			data.RequiresImportErrorStep(testAccAzureRMDigitalTwinsEndpointEventGrid_requiresImport),
		},
	})
}

func TestAccAzureRMDigitalTwinsEndpointEventGrid_updateEventGrid(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_digital_twins_endpoint_eventgrid", "test")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMDigitalTwinsEndpointEventGridDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMDigitalTwinsEndpointEventGrid_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDigitalTwinsEndpointEventGridExists(data.ResourceName),
				),
			},
			data.ImportStep("eventgrid_topic_endpoint", "eventgrid_topic_primary_access_key", "eventgrid_topic_secondary_access_key"),
			{
				Config: testAccAzureRMDigitalTwinsEndpointEventGrid_updateEventGrid(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDigitalTwinsEndpointEventGridExists(data.ResourceName),
				),
			},
			data.ImportStep("eventgrid_topic_endpoint", "eventgrid_topic_primary_access_key", "eventgrid_topic_secondary_access_key"),
			{
				Config: testAccAzureRMDigitalTwinsEndpointEventGrid_updateEventGridRestore(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDigitalTwinsEndpointEventGridExists(data.ResourceName),
				),
			},
			data.ImportStep("eventgrid_topic_endpoint", "eventgrid_topic_primary_access_key", "eventgrid_topic_secondary_access_key"),
		},
	})
}

func TestAccAzureRMDigitalTwinsEndpointEventGrid_updateDeadLetter(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_digital_twins_endpoint_eventgrid", "test")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMDigitalTwinsEndpointEventGridDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMDigitalTwinsEndpointEventGrid_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDigitalTwinsEndpointEventGridExists(data.ResourceName),
				),
			},
			data.ImportStep("eventgrid_topic_endpoint", "eventgrid_topic_primary_access_key", "eventgrid_topic_secondary_access_key"),
			{
				Config: testAccAzureRMDigitalTwinsEndpointEventGrid_updateDeadLetter(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDigitalTwinsEndpointEventGridExists(data.ResourceName),
				),
			},
			data.ImportStep("eventgrid_topic_endpoint", "eventgrid_topic_primary_access_key", "eventgrid_topic_secondary_access_key", "dead_letter_storage_secret"),
			{
				Config: testAccAzureRMDigitalTwinsEndpointEventGrid_updateDeadLetterRestore(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDigitalTwinsEndpointEventGridExists(data.ResourceName),
				),
			},
			data.ImportStep("eventgrid_topic_endpoint", "eventgrid_topic_primary_access_key", "eventgrid_topic_secondary_access_key", "dead_letter_storage_secret"),
		},
	})
}

func testCheckAzureRMDigitalTwinsEndpointEventGridExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.AzureProvider.Meta().(*clients.Client).Digitaltwins.EndpointClient
		ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("digitaltwins Endpoint not found: %s", resourceName)
		}
		id, err := parse.DigitaltwinsEndpointID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if resp, err := client.Get(ctx, id.ResourceGroup, id.ResourceName, id.Name); err != nil {
			if utils.ResponseWasNotFound(resp.Response) {
				return fmt.Errorf("bad: Digitaltwins Endpoint %q does not exist", id.Name)
			}
			return fmt.Errorf("bad: Get on Digitaltwins.EndpointClient: %+v", err)
		}
		return nil
	}
}

func testCheckAzureRMDigitalTwinsEndpointEventGridDestroy(s *terraform.State) error {
	client := acceptance.AzureProvider.Meta().(*clients.Client).Digitaltwins.EndpointClient
	ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_digital_twins_endpoint_eventgrid" {
			continue
		}
		id, err := parse.DigitaltwinsEndpointID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if resp, err := client.Get(ctx, id.ResourceGroup, id.ResourceName, id.Name); err != nil {
			if !utils.ResponseWasNotFound(resp.Response) {
				return fmt.Errorf("bad: Get on Digitaltwins.EndpointClient: %+v", err)
			}
		}
		return nil
	}
	return nil
}
func testAccAzureRMDigitalTwinsEndpointEventGrid_template(data acceptance.TestData) string {
	digitalTwins := testAccAzureRMDigitalTwins_basic(data)
	return fmt.Sprintf(`
%[1]s

resource "azurerm_eventgrid_topic" "test" {
  name                = "acctesteg-%[2]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}
`, digitalTwins, data.RandomInteger)
}

func testAccAzureRMDigitalTwinsEndpointEventGrid_basic(data acceptance.TestData) string {
	template := testAccAzureRMDigitalTwinsEndpointEventGrid_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_digital_twins_endpoint_eventgrid" "test" {
  name                                 = "acctest-EG-%d"
  digital_twins_id                     = azurerm_digital_twins.test.id
  eventgrid_topic_endpoint             = azurerm_eventgrid_topic.test.endpoint
  eventgrid_topic_primary_access_key   = azurerm_eventgrid_topic.test.primary_access_key
  eventgrid_topic_secondary_access_key = azurerm_eventgrid_topic.test.secondary_access_key
}
`, template, data.RandomInteger)
}

func testAccAzureRMDigitalTwinsEndpointEventGrid_requiresImport(data acceptance.TestData) string {
	config := testAccAzureRMDigitalTwinsEndpointEventGrid_basic(data)
	return fmt.Sprintf(`
%s

resource "azurerm_digital_twins_endpoint_eventgrid" "import" {
  name                                 = azurerm_digital_twins_endpoint_eventgrid.test.name
  digital_twins_id                     = azurerm_digital_twins_endpoint_eventgrid.test.digital_twins_id
  eventgrid_topic_endpoint             = azurerm_digital_twins_endpoint_eventgrid.test.eventgrid_topic_endpoint
  eventgrid_topic_primary_access_key   = azurerm_digital_twins_endpoint_eventgrid.test.eventgrid_topic_primary_access_key
  eventgrid_topic_secondary_access_key = azurerm_digital_twins_endpoint_eventgrid.test.eventgrid_topic_secondary_access_key
}
`, config)
}

func testAccAzureRMDigitalTwinsEndpointEventGrid_updateEventGrid(data acceptance.TestData) string {
	template := testAccAzureRMDigitalTwinsEndpointEventGrid_template(data)
	return fmt.Sprintf(`
%[1]s

resource "azurerm_eventgrid_topic" "test_alt" {
  name                = "acctesteg-alt-%[2]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_digital_twins_endpoint_eventgrid" "test" {
  name                                 = "acctest-EG-%[2]d"
  digital_twins_id                     = azurerm_digital_twins.test.id
  eventgrid_topic_endpoint             = azurerm_eventgrid_topic.test_alt.endpoint
  eventgrid_topic_primary_access_key   = azurerm_eventgrid_topic.test_alt.primary_access_key
  eventgrid_topic_secondary_access_key = azurerm_eventgrid_topic.test_alt.secondary_access_key
}
`, template, data.RandomInteger)
}

func testAccAzureRMDigitalTwinsEndpointEventGrid_updateEventGridRestore(data acceptance.TestData) string {
	template := testAccAzureRMDigitalTwinsEndpointEventGrid_template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_eventgrid_topic" "test_alt" {
  name                = "acctesteg-alt-%[2]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_digital_twins_endpoint_eventgrid" "test" {
  name                                 = "acctest-EG-%[2]d"
  digital_twins_id                     = azurerm_digital_twins.test.id
  eventgrid_topic_endpoint             = azurerm_eventgrid_topic.test.endpoint
  eventgrid_topic_primary_access_key   = azurerm_eventgrid_topic.test.primary_access_key
  eventgrid_topic_secondary_access_key = azurerm_eventgrid_topic.test.secondary_access_key
}
`, template, data.RandomInteger)
}

func testAccAzureRMDigitalTwinsEndpointEventGrid_updateDeadLetter(data acceptance.TestData) string {
	template := testAccAzureRMDigitalTwinsEndpointEventGrid_template(data)
	return fmt.Sprintf(`
%[1]s

resource "azurerm_storage_account" "test" {
  name                     = "acctestacc%[2]s"
  resource_group_name      = azurerm_resource_group.test.name
  location                 = azurerm_resource_group.test.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

resource "azurerm_storage_container" "test" {
  name                  = "vhds"
  storage_account_name  = azurerm_storage_account.test.name
  container_access_type = "private"
}

resource "azurerm_digital_twins_endpoint_eventgrid" "test" {
  name                                 = "acctest-EG-%[3]d"
  digital_twins_id                     = azurerm_digital_twins.test.id
  eventgrid_topic_endpoint             = azurerm_eventgrid_topic.test.endpoint
  eventgrid_topic_primary_access_key   = azurerm_eventgrid_topic.test.primary_access_key
  eventgrid_topic_secondary_access_key = azurerm_eventgrid_topic.test.secondary_access_key
  dead_letter_storage_secret           = "${azurerm_storage_container.test.id}?${azurerm_storage_account.test.primary_access_key}"

}
`, template, data.RandomString, data.RandomInteger)
}

func testAccAzureRMDigitalTwinsEndpointEventGrid_updateDeadLetterRestore(data acceptance.TestData) string {
	template := testAccAzureRMDigitalTwinsEndpointEventGrid_template(data)
	return fmt.Sprintf(`
%[1]s

resource "azurerm_storage_account" "test" {
  name                     = "acctestacc%[2]s"
  resource_group_name      = azurerm_resource_group.test.name
  location                 = azurerm_resource_group.test.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

resource "azurerm_storage_container" "test" {
  name                  = "vhds"
  storage_account_name  = azurerm_storage_account.test.name
  container_access_type = "private"
}

resource "azurerm_digital_twins_endpoint_eventgrid" "test" {
  name                                 = "acctest-EG-%[3]d"
  digital_twins_id                     = azurerm_digital_twins.test.id
  eventgrid_topic_endpoint             = azurerm_eventgrid_topic.test.endpoint
  eventgrid_topic_primary_access_key   = azurerm_eventgrid_topic.test.primary_access_key
  eventgrid_topic_secondary_access_key = azurerm_eventgrid_topic.test.secondary_access_key

}
`, template, data.RandomString, data.RandomInteger)
}
