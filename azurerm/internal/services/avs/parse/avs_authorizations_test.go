package parse

import (
	"testing"
)

func TestAvsAuthorizationID(t *testing.T) {
	testData := []struct {
		Name     string
		Input    string
		Expected *AvsAuthorizationId
	}{
		{
			Name:     "Empty",
			Input:    "",
			Expected: nil,
		},
		{
			Name:     "No Resource Groups Segment",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000",
			Expected: nil,
		},
		{
			Name:     "No Resource Groups Value",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/",
			Expected: nil,
		},
		{
			Name:     "Resource Group ID",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/foo/",
			Expected: nil,
		},
		{
			Name:     "Missing Authorization Value",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourceGroup1/providers/Microsoft.AVS/privateClouds/privateCloud1/authorizations",
			Expected: nil,
		},
		{
			Name:  "avs Authorization ID",
			Input: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourceGroup1/providers/Microsoft.AVS/privateClouds/privateCloud1/authorizations/authorization1",
			Expected: &AvsAuthorizationId{
				ResourceGroup:    "resourceGroup1",
				PrivateCloudName: "privateCloud1",
				Name:             "authorization1",
			},
		},
		{
			Name:     "Wrong Casing",
			Input:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourceGroup1/providers/Microsoft.AVS/privateClouds/privateCloud1/Authorizations/authorization1",
			Expected: nil,
		},
	}

	for _, v := range testData {
		t.Logf("[DEBUG] Testing %q..", v.Name)

		actual, err := AvsAuthorizationID(v.Input)
		if err != nil {
			if v.Expected == nil {
				continue
			}
			t.Fatalf("Expected a value but got an error: %s", err)
		}

		if actual.ResourceGroup != v.Expected.ResourceGroup {
			t.Fatalf("Expected %q but got %q for ResourceGroup", v.Expected.ResourceGroup, actual.ResourceGroup)
		}

		if actual.PrivateCloudName != v.Expected.PrivateCloudName {
			t.Fatalf("Expected %q but got %q for PrivateCloudName", v.Expected.PrivateCloudName, actual.PrivateCloudName)
		}

		if actual.Name != v.Expected.Name {
			t.Fatalf("Expected %q but got %q for Name", v.Expected.Name, actual.Name)
		}
	}
}
