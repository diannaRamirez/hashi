package tests

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/apimanagement/mgmt/2019-12-01/apimanagement"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/apimanagement/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func TestAccAzureRMApiManagementIdentityProviderFacebook_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_api_management_identity_provider_facebook", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMApiManagementIdentityProviderFacebookDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMApiManagementIdentityProviderFacebook_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMApiManagementIdentityProviderFacebookExists(data.ResourceName),
				),
			},
			data.ImportStep("app_secret"),
		},
	})
}

func TestAccAzureRMApiManagementIdentityProviderFacebook_basicDeprecated(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_api_management_identity_provider_facebook", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMApiManagementIdentityProviderFacebookDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMApiManagementIdentityProviderFacebook_basicDeprecated(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMApiManagementIdentityProviderFacebookExists(data.ResourceName),
				),
			},
			data.ImportStep("app_secret", "api_management_name", "resource_group_name"),
		},
	})
}

func TestAccAzureRMApiManagementIdentityProviderFacebook_update(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_api_management_identity_provider_facebook", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMApiManagementIdentityProviderFacebookDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMApiManagementIdentityProviderFacebook_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMApiManagementIdentityProviderFacebookExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "app_id", "00000000000000000000000000000000"),
				),
			},
			data.ImportStep("app_secret"),
			{
				Config: testAccAzureRMApiManagementIdentityProviderFacebook_update(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMApiManagementIdentityProviderFacebookExists(data.ResourceName),
					resource.TestCheckResourceAttr(data.ResourceName, "app_id", "11111111111111111111111111111111"),
				),
			},
			data.ImportStep("app_secret"),
		},
	})
}

func TestAccAzureRMApiManagementIdentityProviderFacebook_requiresImport(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_api_management_identity_provider_facebook", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMApiManagementIdentityProviderFacebookDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMApiManagementIdentityProviderFacebook_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMApiManagementIdentityProviderFacebookExists(data.ResourceName),
				),
			},
			data.RequiresImportErrorStep(testAccAzureRMApiManagementIdentityProviderFacebook_requiresImport),
		},
	})
}

func testCheckAzureRMApiManagementIdentityProviderFacebookDestroy(s *terraform.State) error {
	client := acceptance.AzureProvider.Meta().(*clients.Client).ApiManagement.IdentityProviderClient
	ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_api_management_identity_provider_facebook" {
			continue
		}

		apiManagementId := rs.Primary.Attributes["api_management_id"]
		id, err := parse.ApiManagementID(apiManagementId)
		if err != nil {
			return fmt.Errorf("Error parsing API Management ID %q: %+v", apiManagementId, err)
		}

		resp, err := client.Get(ctx, id.ResourceGroup, id.ServiceName, apimanagement.Facebook)

		if err != nil {
			if !utils.ResponseWasNotFound(resp.Response) {
				return err
			}
		}

		return nil
	}
	return nil
}

func testCheckAzureRMApiManagementIdentityProviderFacebookExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.AzureProvider.Meta().(*clients.Client).ApiManagement.IdentityProviderClient
		ctx := acceptance.AzureProvider.Meta().(*clients.Client).StopContext

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		apiManagementId := rs.Primary.Attributes["api_management_id"]
		id, err := parse.ApiManagementID(apiManagementId)
		if err != nil {
			return fmt.Errorf("Error parsing API Management ID %q: %+v", apiManagementId, err)
		}

		resp, err := client.Get(ctx, id.ResourceGroup, id.ServiceName, apimanagement.Facebook)
		if err != nil {
			if utils.ResponseWasNotFound(resp.Response) {
				return fmt.Errorf("Bad: API Management Identity Provider %q (Resource Group %q / API Management Service %q) does not exist", apimanagement.Facebook, id.ResourceGroup, id.ServiceName)
			}
			return fmt.Errorf("Bad: Get on apiManagementIdentityProviderClient: %+v", err)
		}

		return nil
	}
}

func testAccAzureRMApiManagementIdentityProviderFacebook_basic(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-api-%[1]d"
  location = "%[2]s"
}

resource "azurerm_api_management" "test" {
  name                = "acctestAM-%[1]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  publisher_name      = "pub1"
  publisher_email     = "pub1@email.com"
  sku_name            = "Developer_1"
}

resource "azurerm_api_management_identity_provider_facebook" "test" {
  api_management_id = azurerm_api_management.test.id
  app_id            = "00000000000000000000000000000000"
  app_secret        = "00000000000000000000000000000000"
}
`, data.RandomInteger, data.Locations.Primary)
}

func testAccAzureRMApiManagementIdentityProviderFacebook_basicDeprecated(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-api-%[1]d"
  location = "%[2]s"
}

resource "azurerm_api_management" "test" {
  name                = "acctestAM-%[1]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  publisher_name      = "pub1"
  publisher_email     = "pub1@email.com"
  sku_name            = "Developer_1"
}

resource "azurerm_api_management_identity_provider_facebook" "test" {
  resource_group_name = azurerm_resource_group.test.name
  api_management_name = azurerm_api_management.test.name
  app_id              = "00000000000000000000000000000000"
  app_secret          = "00000000000000000000000000000000"
}
`, data.RandomInteger, data.Locations.Primary)
}

func testAccAzureRMApiManagementIdentityProviderFacebook_update(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-api-%[1]d"
  location = "%[2]s"
}

resource "azurerm_api_management" "test" {
  name                = "acctestAM-%[1]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  publisher_name      = "pub1"
  publisher_email     = "pub1@email.com"
  sku_name            = "Developer_1"
}

resource "azurerm_api_management_identity_provider_facebook" "test" {
  api_management_id = azurerm_api_management.test.id
  app_id            = "11111111111111111111111111111111"
  app_secret        = "11111111111111111111111111111111"
}
`, data.RandomInteger, data.Locations.Primary)
}

func testAccAzureRMApiManagementIdentityProviderFacebook_requiresImport(data acceptance.TestData) string {
	template := testAccAzureRMApiManagementIdentityProviderFacebook_basic(data)
	return fmt.Sprintf(`
%s

resource "azurerm_api_management_identity_provider_facebook" "import" {
  api_management_id = azurerm_api_management_identity_provider_facebook.test.api_management_id
  app_id            = azurerm_api_management_identity_provider_facebook.test.app_id
  app_secret        = azurerm_api_management_identity_provider_facebook.test.app_secret
}
`, template)
}
