package azurerm

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"net/http"
	"testing"
)

func TestPolicyDefinitionCreate(t *testing.T) {
	resourceName := "azurerm_policy_definition.test"

	ri := acctest.RandInt()
	config := testAzureRMPolicyDefinition(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkIfPolicyDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPolicyDefinitionExists(resourceName)),
			},
		},
	})
}

func testCheckAzureRMPolicyDefinitionExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}

		policyName := rs.Primary.Attributes["name"]

		client := testAccProvider.Meta().(*ArmClient).policyDefinitionsClient
		ctx := testAccProvider.Meta().(*ArmClient).StopContext

		resp, err := client.Get(ctx, policyName)
		if err != nil {
			return fmt.Errorf("Bad: Get on policyDefinitionsClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("policy does not exist: %s", name)
		}

		return nil
	}
}

func checkIfPolicyDestroyed(s *terraform.State) error {
	client := testAccProvider.Meta().(*ArmClient).policyDefinitionsClient
	ctx := testAccProvider.Meta().(*ArmClient).StopContext

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_policy_definition" {
			continue
		}

		name := rs.Primary.Attributes["name"]

		resp, err := client.Get(ctx, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("policy still exists:%s", *resp.Name)
		}
	}

	return nil
}

func testAzureRMPolicyDefinition(ri int) string {
	return fmt.Sprintf(`
resource "azurerm_policy_definition" "test" {
  name                = "acctestRG-%d"
  policy_type            = "Custom"
  mode                 = "All"
  display_name       = "acctestRG-%d"
  policy_rule =<<POLICY_RULE
	{
    "if": {
      "not": {
        "field": "location",
        "in": "[parameters('allowedLocations')]"
      }
    },
    "then": {
      "effect": "audit"
    }
  }
	POLICY_RULE

  parameters =<<PARAMETERS
	{
    "allowedLocations": {
      "type": "Array",
      "metadata": {
        "description": "The list of allowed locations for resources.",
        "displayName": "Allowed locations",
        "strongType": "location"
      }
    }
  }
	PARAMETERS
}`, ri, ri)
}
