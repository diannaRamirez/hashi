package monitor_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	workbook "github.com/hashicorp/terraform-provider-azurerm/internal/services/monitor/sdk/2022-04-01/insights"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

type MonitorWorkbookResource struct{}

func TestAccMonitorWorkbook_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_monitor_workbook", "test")
	r := MonitorWorkbookResource{}
	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func TestAccMonitorWorkbook_requiresImport(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_monitor_workbook", "test")
	r := MonitorWorkbookResource{}
	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.RequiresImportErrorStep(r.requiresImport),
	})
}

func TestAccMonitorWorkbook_complete(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_monitor_workbook", "test")
	r := MonitorWorkbookResource{}
	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.complete(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func TestAccMonitorWorkbook_update(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_monitor_workbook", "test")
	r := MonitorWorkbookResource{}
	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.complete(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
		{
			Config: r.update(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func TestAccMonitorWorkbook_hiddenTitleInTags(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_monitor_workbook", "test")
	r := MonitorWorkbookResource{}
	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config:      r.hiddenTitleInTags(data),
			ExpectError: regexp.MustCompile("a tag with the key `hidden-title` should not be used to set the display name. Please Use `display_name` instead"),
		},
	})
}

func (r MonitorWorkbookResource) Exists(ctx context.Context, clients *clients.Client, state *pluginsdk.InstanceState) (*bool, error) {
	id, err := workbook.ParseWorkbookID(state.ID)
	if err != nil {
		return nil, err
	}

	client := clients.Monitor.WorkbookClient
	resp, err := client.WorkbooksGet(ctx, *id, workbook.WorkbooksGetOperationOptions{CanFetchContent: utils.Bool(true)})
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			return utils.Bool(false), nil
		}
		return nil, fmt.Errorf("retrieving %s: %+v", id, err)
	}

	return utils.Bool(resp.Model != nil), nil
}

func (r MonitorWorkbookResource) template(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctest-rg-%d"
  location = "%s"
}
`, data.RandomInteger, data.Locations.Primary)
}

func (r MonitorWorkbookResource) basic(data acceptance.TestData) string {
	template := r.template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_monitor_workbook" "test" {
  name                = "bE1ad266-d329-4454-b693-8287e4d3b35d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  display_name        = "acctest-amw-%d"
  source_id           = azurerm_resource_group.test.id
  serialized_data = jsonencode({
    "version" = "Notebook/1.0",
    "items" = [
      {
        "type" = 1,
        "content" = {
          "json" = "Test2022"
        },
        "name" = "text - 0"
      }
    ],
    "isLocked" = false,
    "fallbackResourceIds" = [
      "Azure Monitor"
    ]
  })
}
`, template, data.RandomInteger)
}

func (r MonitorWorkbookResource) hiddenTitleInTags(data acceptance.TestData) string {
	template := r.template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_monitor_workbook" "test" {
  name                = "bE1ad266-d329-4454-b693-8287e4d3b35d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  display_name        = "acctest-amw-%d"
  source_id           = azurerm_resource_group.test.id
  serialized_data = jsonencode({
    "version" = "Notebook/1.0",
    "items" = [
      {
        "type" = 1,
        "content" = {
          "json" = "Test2022"
        },
        "name" = "text - 0"
      }
    ],
    "isLocked" = false,
    "fallbackResourceIds" = [
      "Azure Monitor"
    ]
  })
  tags = {
    hidden-title = "Test Display Name"
  }
}
`, template, data.RandomInteger)
}

func (r MonitorWorkbookResource) requiresImport(data acceptance.TestData) string {
	config := r.basic(data)
	return fmt.Sprintf(`
			%s

resource "azurerm_monitor_workbook" "import" {
  name                = azurerm_monitor_workbook.test.name
  resource_group_name = azurerm_monitor_workbook.test.resource_group_name
  location            = azurerm_monitor_workbook.test.location
  category            = azurerm_monitor_workbook.test.category
  display_name        = azurerm_monitor_workbook.test.display_name
  source_id           = azurerm_monitor_workbook.test.source_id
  serialized_data     = azurerm_monitor_workbook.test.serialized_data
}
`, config)
}

func (r MonitorWorkbookResource) complete(data acceptance.TestData) string {
	template := r.template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_user_assigned_identity" "test" {
  name                = "acctestUAI-%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_storage_account" "test" {
  name                     = "acctestsads%s"
  resource_group_name      = azurerm_resource_group.test.name
  location                 = azurerm_resource_group.test.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

resource "azurerm_storage_container" "test" {
  name                  = "test"
  storage_account_name  = azurerm_storage_account.test.name
  container_access_type = "private"
}

resource "azurerm_role_assignment" "test" {
  scope                = azurerm_storage_account.test.id
  role_definition_name = "Storage Blob Data Owner"
  principal_id         = azurerm_user_assigned_identity.test.principal_id
}

resource "azurerm_monitor_workbook" "test" {
  name                = "0f498fab-2989-4395-b084-fc092d83a6b1"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  display_name        = "acctest-amw-1"
  source_id           = azurerm_resource_group.test.id
  category            = "workbook1"
  description         = "description1"
  storage_uri         = azurerm_storage_container.test.resource_manager_id

  identity {
    type = "UserAssigned"
    identity_ids = [
      azurerm_user_assigned_identity.test.id
    ]
  }

  serialized_data = jsonencode({
    "version" = "Notebook/1.0",
    "items" = [
      {
        "type" = 1,
        "content" = {
          "json" = "Test2021"
        },
        "name" = "text - 0"
      }
    ],
    "isLocked" = false,
    "fallbackResourceIds" = [
      "Azure Monitor"
    ]
  })
  tags = {
    env = "test"
  }

  depends_on = [
    azurerm_role_assignment.test,
  ]
}
`, template, data.RandomInteger, data.RandomString)
}

func (r MonitorWorkbookResource) update(data acceptance.TestData) string {
	template := r.template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_user_assigned_identity" "test" {
  name                = "acctestUAI-%d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_storage_account" "test" {
  name                     = "acctestsads%s"
  resource_group_name      = azurerm_resource_group.test.name
  location                 = azurerm_resource_group.test.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

resource "azurerm_storage_container" "test" {
  name                  = "test"
  storage_account_name  = azurerm_storage_account.test.name
  container_access_type = "private"
}

resource "azurerm_role_assignment" "test" {
  scope                = azurerm_storage_account.test.id
  role_definition_name = "Storage Blob Data Owner"
  principal_id         = azurerm_user_assigned_identity.test.principal_id
}

resource "azurerm_monitor_workbook" "test" {
  name                = "0f498fab-2989-4395-b084-fc092d83a6b1"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  display_name        = "acctest-amw-2"
  source_id           = azurerm_resource_group.test.id
  category            = "workbook2"
  description         = "description2"
  storage_uri         = azurerm_storage_container.test.resource_manager_id

  identity {
    type = "UserAssigned"
    identity_ids = [
      azurerm_user_assigned_identity.test.id
    ]
  }

  serialized_data = jsonencode({
    "version" = "Notebook/1.0",
    "items" = [
      {
        "type" = 1,
        "content" = {
          "json" = "Test2022"
        },
        "name" = "text - 0"
      }
    ],
    "isLocked" = false,
    "fallbackResourceIds" = [
      "Azure Monitor"
    ]
  })
  tags = {
    env = "test2"
  }

  depends_on = [
    azurerm_role_assignment.test,
  ]
}
`, template, data.RandomInteger, data.RandomString)
}
