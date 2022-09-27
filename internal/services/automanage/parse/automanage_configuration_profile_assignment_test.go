package parse

// NOTE: this file is generated via 'go:generate' - manual changes will be overwritten

import (
	"testing"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/resourceids"
)

var _ resourceids.Id = AutomanageConfigurationProfileAssignmentId{}

func TestAutomanageConfigurationProfileAssignmentIDFormatter(t *testing.T) {
	actual := NewAutomanageConfigurationProfileAssignmentID("12345678-1234-9876-4563-123456789012", "resourceGroup1", "vm1", "configurationProfileAssignment1").ID()
	expected := "/subscriptions/12345678-1234-9876-4563-123456789012/resourceGroups/resourceGroup1/providers/Microsoft.Compute/virtualMachines/vm1/providers/Microsoft.Automanage/configurationProfileAssignments/configurationProfileAssignment1"
	if actual != expected {
		t.Fatalf("Expected %q but got %q", expected, actual)
	}
}

func TestAutomanageConfigurationProfileAssignmentID(t *testing.T) {
	testData := []struct {
		Input    string
		Error    bool
		Expected *AutomanageConfigurationProfileAssignmentId
	}{

		{
			// empty
			Input: "",
			Error: true,
		},

		{
			// missing SubscriptionId
			Input: "/",
			Error: true,
		},

		{
			// missing value for SubscriptionId
			Input: "/subscriptions/",
			Error: true,
		},

		{
			// missing ResourceGroup
			Input: "/subscriptions/12345678-1234-9876-4563-123456789012/",
			Error: true,
		},

		{
			// missing value for ResourceGroup
			Input: "/subscriptions/12345678-1234-9876-4563-123456789012/resourceGroups/",
			Error: true,
		},

		{
			// missing VirtualMachineName
			Input: "/subscriptions/12345678-1234-9876-4563-123456789012/resourceGroups/resourceGroup1/providers/Microsoft.Compute/",
			Error: true,
		},

		{
			// missing value for VirtualMachineName
			Input: "/subscriptions/12345678-1234-9876-4563-123456789012/resourceGroups/resourceGroup1/providers/Microsoft.Compute/virtualMachines/",
			Error: true,
		},

		{
			// missing ConfigurationProfileAssignmentName
			Input: "/subscriptions/12345678-1234-9876-4563-123456789012/resourceGroups/resourceGroup1/providers/Microsoft.Compute/virtualMachines/vm1/providers/Microsoft.Automanage/",
			Error: true,
		},

		{
			// missing value for ConfigurationProfileAssignmentName
			Input: "/subscriptions/12345678-1234-9876-4563-123456789012/resourceGroups/resourceGroup1/providers/Microsoft.Compute/virtualMachines/vm1/providers/Microsoft.Automanage/configurationProfileAssignments/",
			Error: true,
		},

		{
			// valid
			Input: "/subscriptions/12345678-1234-9876-4563-123456789012/resourceGroups/resourceGroup1/providers/Microsoft.Compute/virtualMachines/vm1/providers/Microsoft.Automanage/configurationProfileAssignments/configurationProfileAssignment1",
			Expected: &AutomanageConfigurationProfileAssignmentId{
				SubscriptionId:                     "12345678-1234-9876-4563-123456789012",
				ResourceGroup:                      "resourceGroup1",
				VirtualMachineName:                 "vm1",
				ConfigurationProfileAssignmentName: "configurationProfileAssignment1",
			},
		},

		{
			// upper-cased
			Input: "/SUBSCRIPTIONS/12345678-1234-9876-4563-123456789012/RESOURCEGROUPS/RESOURCEGROUP1/PROVIDERS/MICROSOFT.COMPUTE/VIRTUALMACHINES/VM1/PROVIDERS/MICROSOFT.AUTOMANAGE/CONFIGURATIONPROFILEASSIGNMENTS/CONFIGURATIONPROFILEASSIGNMENT1",
			Error: true,
		},
	}

	for _, v := range testData {
		t.Logf("[DEBUG] Testing %q", v.Input)

		actual, err := AutomanageConfigurationProfileAssignmentID(v.Input)
		if err != nil {
			if v.Error {
				continue
			}

			t.Fatalf("Expect a value but got an error: %s", err)
		}
		if v.Error {
			t.Fatal("Expect an error but didn't get one")
		}

		if actual.SubscriptionId != v.Expected.SubscriptionId {
			t.Fatalf("Expected %q but got %q for SubscriptionId", v.Expected.SubscriptionId, actual.SubscriptionId)
		}
		if actual.ResourceGroup != v.Expected.ResourceGroup {
			t.Fatalf("Expected %q but got %q for ResourceGroup", v.Expected.ResourceGroup, actual.ResourceGroup)
		}
		if actual.VirtualMachineName != v.Expected.VirtualMachineName {
			t.Fatalf("Expected %q but got %q for VirtualMachineName", v.Expected.VirtualMachineName, actual.VirtualMachineName)
		}
		if actual.ConfigurationProfileAssignmentName != v.Expected.ConfigurationProfileAssignmentName {
			t.Fatalf("Expected %q but got %q for ConfigurationProfileAssignmentName", v.Expected.ConfigurationProfileAssignmentName, actual.ConfigurationProfileAssignmentName)
		}
	}
}
