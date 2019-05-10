package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
)

func TestAccAzureRMAutomationBoolVariable_basic(t *testing.T) {
	resourceName := "azurerm_automation_variable_bool.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMAutomationBoolVariableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMAutomationBoolVariable_basic(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMAutomationBoolVariableExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "value", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAzureRMAutomationBoolVariable_complete(t *testing.T) {
	resourceName := "azurerm_automation_variable_bool.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMAutomationBoolVariableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMAutomationBoolVariable_complete(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMAutomationBoolVariableExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "This variable is created by Terraform acceptance test."),
					resource.TestCheckResourceAttr(resourceName, "value", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAzureRMAutomationBoolVariable_basicCompleteUpdate(t *testing.T) {
	resourceName := "azurerm_automation_variable_bool.test"
	ri := tf.AccRandTimeInt()
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMAutomationBoolVariableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMAutomationBoolVariable_basic(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMAutomationBoolVariableExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "value", "false"),
				),
			},
			{
				Config: testAccAzureRMAutomationBoolVariable_complete(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMAutomationBoolVariableExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "This variable is created by Terraform acceptance test."),
					resource.TestCheckResourceAttr(resourceName, "value", "true"),
				),
			},
			{
				Config: testAccAzureRMAutomationBoolVariable_basic(ri, location),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMAutomationBoolVariableExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "value", "false"),
				),
			},
		},
	})
}

func testCheckAzureRMAutomationBoolVariableExists(resourceName string) resource.TestCheckFunc {
	return testCheckAzureRMAutomationVariableExists(resourceName, "Bool")
}

func testCheckAzureRMAutomationBoolVariableDestroy(s *terraform.State) error {
	return testCheckAzureRMAutomationVariableDestroy(s, "Bool")
}

func testAccAzureRMAutomationBoolVariable_basic(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_automation_account" "test" {
  name                = "acctestAutoAcct-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"

  sku {
    name = "Basic"
  }
}

resource "azurerm_automation_variable_bool" "test" {
  name                    = "acctestAutoVar-%d"
  resource_group_name     = "${azurerm_resource_group.test.name}"
  automation_account_name = "${azurerm_automation_account.test.name}"
  value                   = false
}
`, rInt, location, rInt, rInt)
}

func testAccAzureRMAutomationBoolVariable_complete(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurerm_automation_account" "test" {
  name                = "acctestAutoAcct-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"

  sku {
    name = "Basic"
  }
}

resource "azurerm_automation_variable_bool" "test" {
  name                    = "acctestAutoVar-%d"
  resource_group_name     = "${azurerm_resource_group.test.name}"
  automation_account_name = "${azurerm_automation_account.test.name}"
  description             = "This variable is created by Terraform acceptance test."
  value                   = true
}
`, rInt, location, rInt, rInt)
}
