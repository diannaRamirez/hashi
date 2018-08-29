package azurerm

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/response"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

/*
	---Testing for Success---
	Test a basic PostgreSQL virtual network rule configuration setup scenario.
*/
func TestAccAzureRMPostgreSQLVirtualNetworkRule_basic(t *testing.T) {
	resourceName := "azurerm_postgresql_virtual_network_rule.test"
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPostgreSQLVirtualNetworkRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMPostgreSQLVirtualNetworkRule_basic(ri, testLocation()),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPostgreSQLVirtualNetworkRuleExists(resourceName),
				),
			},
		},
	})
}

/*
	---Testing for Success---
	Test an update to the PostgreSQL Virtual Network Rule to connect to a different subnet, and
	validate that new subnet is set correctly.
*/
func TestAccAzureRMPostgreSQLVirtualNetworkRule_switchSubnets(t *testing.T) {
	resourceName := "azurerm_postgresql_virtual_network_rule.test"
	ri := acctest.RandInt()

	preConfig := testAccAzureRMPostgreSQLVirtualNetworkRule_subnetSwitchPre(ri, testLocation())
	postConfig := testAccAzureRMPostgreSQLVirtualNetworkRule_subnetSwitchPost(ri, testLocation())

	// Create regex strings that will ensure that one subnet name exists, but not the other
	preConfigRegex := regexp.MustCompile(fmt.Sprintf("(subnet1%d)$|(subnet[^2]%d)$", ri, ri))  //subnet 1 but not 2
	postConfigRegex := regexp.MustCompile(fmt.Sprintf("(subnet2%d)$|(subnet[^1]%d)$", ri, ri)) //subnet 2 but not 1

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPostgreSQLVirtualNetworkRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPostgreSQLVirtualNetworkRuleExists(resourceName),
					resource.TestMatchResourceAttr(resourceName, "subnet_id", preConfigRegex),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPostgreSQLVirtualNetworkRuleExists(resourceName),
					resource.TestMatchResourceAttr(resourceName, "subnet_id", postConfigRegex),
				),
			},
		},
	})
}

/*
	---Testing for Success---
*/
func TestAccAzureRMPostgreSQLVirtualNetworkRule_disappears(t *testing.T) {
	resourceName := "azurerm_postgresql_virtual_network_rule.test"
	ri := acctest.RandInt()
	config := testAccAzureRMPostgreSQLVirtualNetworkRule_basic(ri, testLocation())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPostgreSQLVirtualNetworkRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPostgreSQLVirtualNetworkRuleExists(resourceName),
					testCheckAzureRMPostgreSQLVirtualNetworkRuleDisappears(resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

/*
	--Testing for Success--
	Test if we are able to create multiple subnets and connect multiple subnets to the
	PostgreSQL server.
*/
func TestAccAzureRMPostgreSQLVirtualNetworkRule_multipleSubnets(t *testing.T) {
	resourceName1 := "azurerm_postgresql_virtual_network_rule.rule1"
	resourceName2 := "azurerm_postgresql_virtual_network_rule.rule2"
	resourceName3 := "azurerm_postgresql_virtual_network_rule.rule3"
	ri := acctest.RandInt()
	config := testAccAzureRMPostgreSQLVirtualNetworkRule_multipleSubnets(ri, testLocation())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPostgreSQLVirtualNetworkRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPostgreSQLVirtualNetworkRuleExists(resourceName1),
					testCheckAzureRMPostgreSQLVirtualNetworkRuleExists(resourceName2),
					testCheckAzureRMPostgreSQLVirtualNetworkRuleExists(resourceName3),
				),
			},
		},
	})
}

/*
	--Testing for Failure--
	Validation Function Tests - Invalid Name Validations
*/
func TestResourceAzureRMPostgreSQLVirtualNetworkRule_invalidNameValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		// Must only contain alphanumeric characters or hyphens (4 cases)
		{
			Value:    "test!Rule",
			ErrCount: 1,
		},
		{
			Value:    "test_Rule",
			ErrCount: 1,
		},
		{
			Value:    "test:Rule",
			ErrCount: 1,
		},
		{
			Value:    "test'Rule",
			ErrCount: 1,
		},
		// Cannot be more than 128 characters (1 case - ensure starts with a letter)
		{
			Value:    fmt.Sprintf("v%s", acctest.RandString(128)),
			ErrCount: 1,
		},
		// Cannot be empty (1 case)
		{
			Value:    "",
			ErrCount: 1,
		},
		// Cannot end in a hyphen (1 case)
		{
			Value:    "testRule-",
			ErrCount: 1,
		},
		// Cannot start with a number or hyphen (2 cases)
		{
			Value:    "7testRule",
			ErrCount: 1,
		},
		{
			Value:    "-testRule",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validatePostgreSQLVirtualNetworkRuleName(tc.Value, "azurerm_postgresql_virtual_network_rule")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Bad: Expected the Azure RM PostgreSQL Virtual Network Rule Name to trigger a validation error.")
		}
	}
}

/*
	--Testing for Success--
	Validation Function Tests - (Barely) Valid Name Validations
*/
func TestResourceAzureRMPostgreSQLVirtualNetworkRule_validNameValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		// Test all lowercase
		{
			Value:    "thisisarule",
			ErrCount: 0,
		},
		// Test all uppercase
		{
			Value:    "THISISARULE",
			ErrCount: 0,
		},
		// Test alternating cases
		{
			Value:    "tHiSiSaRuLe",
			ErrCount: 0,
		},
		// Test hyphens
		{
			Value:    "this-is-a-rule",
			ErrCount: 0,
		},
		// Test multiple hyphens in a row
		{
			Value:    "this----1s----a----ru1e",
			ErrCount: 0,
		},
		// Test numbers (except for first character)
		{
			Value:    "v1108501298509850810258091285091820-5",
			ErrCount: 0,
		},
		// Test a lot of hyphens and numbers
		{
			Value:    "x-5-4-1-2-5-2-6-1-5-2-5-1-2-5-6-2-2",
			ErrCount: 0,
		},
		// Test exactly 128 characters
		{
			Value:    fmt.Sprintf("v%s", acctest.RandString(127)),
			ErrCount: 0,
		},
		// Test short, 1-letter name
		{
			Value:    "V",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validatePostgreSQLVirtualNetworkRuleName(tc.Value, "azurerm_postgresql_virtual_network_rule")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Bad: Expected the Azure RM PostgreSQL Virtual Network Rule Name pass name validation successfully but triggered a validation error.")
		}
	}
}

/*
	Test Check function to assert if a rule exists or not.
*/
func testCheckAzureRMPostgreSQLVirtualNetworkRuleExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		resourceGroup := rs.Primary.Attributes["resource_group_name"]
		serverName := rs.Primary.Attributes["server_name"]
		ruleName := rs.Primary.Attributes["name"]

		client := testAccProvider.Meta().(*ArmClient).postgresqlVirtualNetworkRulesClient
		ctx := testAccProvider.Meta().(*ArmClient).StopContext

		resp, err := client.Get(ctx, resourceGroup, serverName, ruleName)
		if err != nil {
			if utils.ResponseWasNotFound(resp.Response) {
				return fmt.Errorf("Bad: PostgreSQL Firewall Rule %q (server %q / resource group %q) was not found", ruleName, serverName, resourceGroup)
			}

			return err
		}

		return nil
	}
}

/*
	Test Check function to delete a rule.
*/
func testCheckAzureRMPostgreSQLVirtualNetworkRuleDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_postgresql_virtual_network_rule" {
			continue
		}

		resourceGroup := rs.Primary.Attributes["resource_group_name"]
		serverName := rs.Primary.Attributes["server_name"]
		ruleName := rs.Primary.Attributes["name"]

		client := testAccProvider.Meta().(*ArmClient).postgresqlVirtualNetworkRulesClient
		ctx := testAccProvider.Meta().(*ArmClient).StopContext

		resp, err := client.Get(ctx, resourceGroup, serverName, ruleName)
		if err != nil {
			if utils.ResponseWasNotFound(resp.Response) {
				return nil
			}

			return err
		}

		return fmt.Errorf("Bad: PostgreSQL Firewall Rule %q (server %q / resource group %q) still exists: %+v", ruleName, serverName, resourceGroup, resp)
	}

	return nil
}

/*
	Test Check function to assert if that a rule gets deleted.
*/
func testCheckAzureRMPostgreSQLVirtualNetworkRuleDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		resourceGroup := rs.Primary.Attributes["resource_group_name"]
		serverName := rs.Primary.Attributes["server_name"]
		ruleName := rs.Primary.Attributes["name"]

		client := testAccProvider.Meta().(*ArmClient).postgresqlVirtualNetworkRulesClient
		ctx := testAccProvider.Meta().(*ArmClient).StopContext

		future, err := client.Delete(ctx, resourceGroup, serverName, ruleName)
		if err != nil {
			//If the error is that the resource we want to delete does not exist in the first
			//place (404), then just return with no error.
			if response.WasNotFound(future.Response()) {
				return nil
			}

			return fmt.Errorf("Error deleting PostgreSQL Virtual Network Rule: %+v", err)
		}

		err = future.WaitForCompletionRef(ctx, client.Client)
		if err != nil {
			//Same deal as before. Just in case.
			if response.WasNotFound(future.Response()) {
				return nil
			}

			return fmt.Errorf("Error deleting PostgreSQL Virtual Network Rule: %+v", err)
		}

		return nil
	}
}

/*
	(This test configuration is intended to succeed.)
	Basic Provisioning Configuration
*/
func testAccAzureRMPostgreSQLVirtualNetworkRule_basic(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name = "acctestRG-%d"
  location = "%s"
}
resource "azurerm_virtual_network" "test" {
  name                = "acctestvnet%d"
  address_space       = ["10.7.29.0/29"]
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
}
resource "azurerm_subnet" "test" {
  name = "acctestsubnet%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix = "10.7.29.0/29"
  service_endpoints = ["Microsoft.Sql"]
}
resource "azurerm_postgresql_server" "test" {
  name                = "acctestpostgresqlsvr-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku {
    name     = "GP_Gen5_2"
    capacity = 2
    tier     = "GeneralPurpose"
    family   = "Gen5"
  }
  storage_profile {
    storage_mb            = 51200
    backup_retention_days = 7
    geo_redundant_backup  = "Disabled"
  }
  administrator_login          = "acctestun"
  administrator_login_password = "H@Sh1CoR3!"
  version                      = "9.5"
  ssl_enforcement              = "Enabled"
}
resource "azurerm_postgresql_virtual_network_rule" "test" {
  name = "acctestpostgresqlvnetrule%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  server_name = "${azurerm_postgresql_server.test.name}"
  subnet_id = "${azurerm_subnet.test.id}"
}
`, rInt, location, rInt, rInt, rInt, rInt)
}

/*
	(This test configuration is intended to succeed.)
	This test is designed to set up a scenario where a user would want to update the subnet
	on a given PostgreSQL virtual network rule. This configuration sets up the resources initially.
*/
func testAccAzureRMPostgreSQLVirtualNetworkRule_subnetSwitchPre(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name = "acctestRG-%d"
  location = "%s"
}
resource "azurerm_virtual_network" "test" {
  name                = "acctestvnet%d"
  address_space       = ["10.7.29.0/24"]
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
}
resource "azurerm_subnet" "test1" {
  name = "subnet1%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix = "10.7.29.0/25"
  service_endpoints = ["Microsoft.Sql"]
}
resource "azurerm_subnet" "test2" {
  name = "subnet2%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix = "10.7.29.128/25"
  service_endpoints = ["Microsoft.Sql"]
}
resource "azurerm_postgresql_server" "test" {
  name                = "acctestpostgresqlsvr-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku {
    name     = "GP_Gen5_2"
    capacity = 2
    tier     = "GeneralPurpose"
    family   = "Gen5"
  }
  storage_profile {
    storage_mb            = 51200
    backup_retention_days = 7
    geo_redundant_backup  = "Disabled"
  }
  administrator_login          = "acctestun"
  administrator_login_password = "H@Sh1CoR3!"
  version                      = "9.5"
  ssl_enforcement              = "Enabled"
}
resource "azurerm_postgresql_virtual_network_rule" "test" {
  name = "acctestpostgresqlvnetrule%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  server_name = "${azurerm_postgresql_server.test.name}"
  subnet_id = "${azurerm_subnet.test1.id}"
}
`, rInt, location, rInt, rInt, rInt, rInt, rInt)
}

/*
	(This test configuration is intended to succeed.)
	This test is designed to set up a scenario where a user would want to update the subnet
	on a given PostgreSQL virtual network rule. This configuration contains the update from
	azurerm_subnet.test1 to azurerm_subnet.test2.
*/
func testAccAzureRMPostgreSQLVirtualNetworkRule_subnetSwitchPost(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name = "acctestRG-%d"
  location = "%s"
}
resource "azurerm_virtual_network" "test" {
  name                = "acctestvnet%d"
  address_space       = ["10.7.29.0/24"]
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
}
resource "azurerm_subnet" "test1" {
  name = "subnet1%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix = "10.7.29.0/25"
  service_endpoints = ["Microsoft.Sql"]
}
resource "azurerm_subnet" "test2" {
  name = "subnet2%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.test.name}"
  address_prefix = "10.7.29.128/25"
  service_endpoints = ["Microsoft.Sql"]
}
resource "azurerm_postgresql_server" "test" {
  name                = "acctestpostgresqlsvr-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku {
    name     = "GP_Gen5_2"
    capacity = 2
    tier     = "GeneralPurpose"
    family   = "Gen5"
  }
  storage_profile {
    storage_mb            = 51200
    backup_retention_days = 7
    geo_redundant_backup  = "Disabled"
  }
  administrator_login          = "acctestun"
  administrator_login_password = "H@Sh1CoR3!"
  version                      = "9.5"
  ssl_enforcement              = "Enabled"
}
resource "azurerm_postgresql_virtual_network_rule" "test" {
  name = "acctestpostgresqlvnetrule%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  server_name = "${azurerm_postgresql_server.test.name}"
  subnet_id = "${azurerm_subnet.test2.id}"
}
`, rInt, location, rInt, rInt, rInt, rInt, rInt)
}

/*
	(This test configuration is intended to succeed.)
	This configuration sets up 3 subnets in 2 different virtual networks, and adds
	PostgreSQL virtual network rules for all 3 subnets to the SQL server.
*/
func testAccAzureRMPostgreSQLVirtualNetworkRule_multipleSubnets(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name = "acctestRG-%d"
  location = "%s"
}
resource "azurerm_virtual_network" "vnet1" {
  name                = "acctestvnet1%d"
  address_space       = ["10.7.29.0/24"]
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
}
resource "azurerm_virtual_network" "vnet2" {
  name                = "acctestvnet2%d"
  address_space       = ["10.1.29.0/29"]
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
}
resource "azurerm_subnet" "vnet1_subnet1" {
  name = "acctestsubnet1%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.vnet1.name}"
  address_prefix = "10.7.29.0/29"
  service_endpoints = ["Microsoft.Sql"]
}
resource "azurerm_subnet" "vnet1_subnet2" {
  name = "acctestsubnet2%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.vnet1.name}"
  address_prefix = "10.7.29.128/29"
  service_endpoints = ["Microsoft.Sql"]
}
resource "azurerm_subnet" "vnet2_subnet1" {
  name = "acctestsubnet3%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  virtual_network_name = "${azurerm_virtual_network.vnet2.name}"
  address_prefix = "10.1.29.0/29"
  service_endpoints = ["Microsoft.Sql"]
}
resource "azurerm_postgresql_server" "test" {
  name                = "acctestpostgresqlsvr-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  sku {
    name     = "GP_Gen5_2"
    capacity = 2
    tier     = "GeneralPurpose"
    family   = "Gen5"
  }
  storage_profile {
    storage_mb            = 51200
    backup_retention_days = 7
    geo_redundant_backup  = "Disabled"
  }
  administrator_login          = "acctestun"
  administrator_login_password = "H@Sh1CoR3!"
  version                      = "9.5"
  ssl_enforcement              = "Enabled"
}
resource "azurerm_postgresql_virtual_network_rule" "rule1" {
  name = "acctestpostgresqlvnetrule1%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  server_name = "${azurerm_postgresql_server.test.name}"
  subnet_id = "${azurerm_subnet.vnet1_subnet1.id}"
}
resource "azurerm_postgresql_virtual_network_rule" "rule2" {
  name = "acctestpostgresqlvnetrule2%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  server_name = "${azurerm_postgresql_server.test.name}"
  subnet_id = "${azurerm_subnet.vnet1_subnet2.id}"
}
resource "azurerm_postgresql_virtual_network_rule" "rule3" {
  name = "acctestpostgresqlvnetrule3%d"
  resource_group_name = "${azurerm_resource_group.test.name}"
  server_name = "${azurerm_postgresql_server.test.name}"
  subnet_id = "${azurerm_subnet.vnet2_subnet1.id}"
}
`, rInt, location, rInt, rInt, rInt, rInt, rInt, rInt, rInt, rInt, rInt)
}
