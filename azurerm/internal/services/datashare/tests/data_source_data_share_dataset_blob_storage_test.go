package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
)

func TestAccDataSourceAzureRMDataShareDatasetBlobStorage_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "data.azurerm_data_share_dataset_blob_storage", "test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acceptance.PreCheck(t) },
		Providers:    acceptance.SupportedProviders,
		CheckDestroy: testCheckAzureRMDataShareDataSetDestroy("azurerm_data_share_dataset_blob_storage"),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceDataShareDatasetBlobStorage_basic(data),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDataShareDataSetExists(data.ResourceName),
					resource.TestCheckResourceAttrSet(data.ResourceName, "container_name"),
					resource.TestCheckResourceAttrSet(data.ResourceName, "storage_account_name"),
					resource.TestCheckResourceAttrSet(data.ResourceName, "storage_account_resource_group_name"),
					resource.TestCheckResourceAttrSet(data.ResourceName, "storage_account_subscription_id"),
					resource.TestCheckResourceAttrSet(data.ResourceName, "file_path"),
					resource.TestCheckResourceAttrSet(data.ResourceName, "display_name"),
				),
			},
		},
	})
}

func testAccDataSourceDataShareDatasetBlobStorage_basic(data acceptance.TestData) string {
	config := testAccAzureRMDataShareDataSetBlobStorageFile_basic(data)
	return fmt.Sprintf(`
%s

data "azurerm_data_share_dataset_blob_storage" "test" {
  name     = azurerm_data_share_dataset_blob_storage.test.name
  share_id = azurerm_data_share_dataset_blob_storage.test.share_id
}
`, config)
}
