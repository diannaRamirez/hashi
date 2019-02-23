package azurerm

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmStorageAccountEncryptionSettings() *schema.Resource {
	return &schema.Resource{
		Read:   resourceArmStorageAccountEncryptionSettingsRead,
		Create: resourceArmStorageAccountEncryptionSettingsCreateUpdate,
		Update: resourceArmStorageAccountEncryptionSettingsCreateUpdate,
		Delete: resourceArmStorageAccountEncryptionSettingsDelete,

		Importer: &schema.ResourceImporter{
			State: resourceArmStorageAccountEncryptionSettingsImportState,
		},
		SchemaVersion: 2,

		Schema: map[string]*schema.Schema{
			"storage_account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: azure.ValidateResourceID,
			},
			"enable_blob_encryption": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"enable_file_encryption": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"key_vault": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// This attribute is not used, it was only added
						// to  create a dependency between this resource
						// and the key vault policy
						"key_vault_policy_id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: azure.ValidateResourceID,
						},

						"key_vault_id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: azure.ValidateResourceID,
						},

						"key_name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validate.NoEmptyStrings,
						},
						"key_version": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validate.NoEmptyStrings,
						},
						"key_vault_uri": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceArmStorageAccountEncryptionSettingsCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	ctx := meta.(*ArmClient).StopContext
	vaultClient := meta.(*ArmClient).keyVaultClient
	client := meta.(*ArmClient).storageServiceClient

	storageAccountId := d.Get("storage_account_id").(string)

	id, err := azure.ParseAzureResourceID(storageAccountId)
	if err != nil {
		return err
	}

	storageAccountName := id.Path["storageAccounts"]
	resourceGroupName := id.ResourceGroup

	// set default values for the attributes
	enableBlobEncryption := true
	enableFileEncryption := true

	// create the update object with the default values
	opts := storage.AccountUpdateParameters{
		AccountPropertiesUpdateParameters: &storage.AccountPropertiesUpdateParameters{
			Encryption: &storage.Encryption{
				Services: &storage.EncryptionServices{
					Blob: &storage.EncryptionService{
						Enabled: utils.Bool(enableBlobEncryption),
					},
					File: &storage.EncryptionService{
						Enabled: utils.Bool(enableFileEncryption),
					}},
				KeySource:          storage.MicrosoftStorage,
				KeyVaultProperties: &storage.KeyVaultProperties{},
			},
		},
	}

	if d.HasChange("enable_blob_encryption") || d.HasChange("enable_file_encryption") {
		opts.Encryption.Services.Blob.Enabled = utils.Bool(d.Get("enable_blob_encryption").(bool))
		opts.Encryption.Services.File.Enabled = utils.Bool(d.Get("enable_file_encryption").(bool))
	}

	if keyVaultProperties := expandAzureRmStorageAccountKeyVaultProperties(d); keyVaultProperties.KeyName != utils.String("") {
		if v, ok := d.GetOk("key_vault.0.key_vault_id"); ok {
			// Get the key vault base URL from the key vault
			keyVaultId := v.(string)
			pKeyVaultBaseUrl, err := azure.GetKeyVaultBaseUrlFromID(ctx, vaultClient, keyVaultId)

			if err != nil {
				return fmt.Errorf("Error looking up Key Vault URI from id %q: %+v", keyVaultId, err)
			}

			keyVaultProperties.KeyVaultURI = utils.String(pKeyVaultBaseUrl)
			opts.Encryption.KeyVaultProperties = keyVaultProperties
			opts.Encryption.KeySource = storage.MicrosoftKeyvault
		}
	}

	_, err = client.Update(ctx, resourceGroupName, storageAccountName, opts)
	if err != nil {
		return fmt.Errorf("Error updating Azure Storage Account Encryption %q: %+v", storageAccountName, err)
	}

	resourceId := fmt.Sprintf("%s/encryptionSettings", storageAccountId)
	d.SetId(resourceId)

	return resourceArmStorageAccountEncryptionSettingsRead(d, meta)
}

func resourceArmStorageAccountEncryptionSettingsRead(d *schema.ResourceData, meta interface{}) error {
	ctx := meta.(*ArmClient).StopContext
	client := meta.(*ArmClient).storageServiceClient

	storageAccountId := d.Get("storage_account_id").(string)

	id, err := parseAzureResourceID(storageAccountId)
	if err != nil {
		return err
	}
	name := id.Path["storageAccounts"]
	resGroup := id.ResourceGroup

	resp, err := client.GetProperties(ctx, resGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading the state of AzureRM Storage Account %q: %+v", name, err)
	}

	if props := resp.AccountProperties; props != nil {
		if encryption := props.Encryption; encryption != nil {
			if services := encryption.Services; services != nil {
				if blob := services.Blob; blob != nil {
					d.Set("enable_blob_encryption", blob.Enabled)
				}
				if file := services.File; file != nil {
					d.Set("enable_file_encryption", file.Enabled)
				}
			}

			if keyVaultProperties := encryption.KeyVaultProperties; keyVaultProperties != nil {
				keyVaultId := d.Get("key_vault.0.key_vault_id").(string)
				keyVaultPolicyId := d.Get("key_vault.0.key_vault_policy_id").(string)

				if err := d.Set("key_vault", flattenAzureRmStorageAccountKeyVaultProperties(keyVaultProperties, keyVaultId, keyVaultPolicyId)); err != nil {
					return fmt.Errorf("Error flattening `key_vault_properties`: %+v", err)
				}
			}
		}
	}

	return nil
}

func resourceArmStorageAccountEncryptionSettingsDelete(d *schema.ResourceData, meta interface{}) error {
	ctx := meta.(*ArmClient).StopContext
	client := meta.(*ArmClient).storageServiceClient

	storageAccountId := d.Get("storage_account_id").(string)

	id, err := azure.ParseAzureResourceID(storageAccountId)
	if err != nil {
		return err
	}

	storageAccountName := id.Path["storageAccounts"]
	resourceGroupName := id.ResourceGroup

	// Since this isn't a real object, just modifying an existing object
	// "Delete" doesn't really make sense it should really be a "Revert to Default"
	// So instead of the Delete func actually deleting the Storage Account I am
	// making it reset the Storage Account to it's default state
	opts := storage.AccountUpdateParameters{
		AccountPropertiesUpdateParameters: &storage.AccountPropertiesUpdateParameters{
			Encryption: &storage.Encryption{
				KeySource: storage.MicrosoftStorage,
			},
		},
	}

	_, err = client.Update(ctx, resourceGroupName, storageAccountName, opts)
	if err != nil {
		return fmt.Errorf("Error updating Azure Storage Account Encryption %q: %+v", storageAccountName, err)
	}

	return nil
}

func resourceArmStorageAccountEncryptionSettingsImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ctx := meta.(*ArmClient).StopContext
	client := meta.(*ArmClient).storageServiceClient

	id := d.Id()

	d.Set("storage_account_id", id)

	saId, err := parseAzureResourceID(id)
	if err != nil {
		return nil, err
	}
	name := saId.Path["storageAccounts"]
	resGroup := saId.ResourceGroup

	resp, err := client.GetProperties(ctx, resGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil, nil
		}
		return nil, fmt.Errorf("Error importing the state of AzureRM Storage Account %q: %+v", name, err)
	}

	if props := resp.AccountProperties; props != nil {
		if encryption := props.Encryption; encryption != nil {
			if services := encryption.Services; services != nil {
				if blob := services.Blob; blob != nil {
					d.Set("enable_blob_encryption", blob.Enabled)
				}
				if file := services.File; file != nil {
					d.Set("enable_file_encryption", file.Enabled)
				}
			}

			if keyVaultProperties := encryption.KeyVaultProperties; keyVaultProperties != nil {
				if err := d.Set("key_vault", flattenAzureRmStorageAccountKeyVaultProperties(keyVaultProperties, "", "")); err != nil {
					return nil, fmt.Errorf("Error flattening `key_vault_properties` on import: %+v", err)
				}
			}
		}
	}

	resourceId := fmt.Sprintf("%s/encryptionSettings", id)
	d.SetId(resourceId)

	results := make([]*schema.ResourceData, 1)

	results[0] = d
	return results, nil
}

func expandAzureRmStorageAccountKeyVaultProperties(d *schema.ResourceData) *storage.KeyVaultProperties {
	vs := d.Get("key_vault").([]interface{})
	if len(vs) == 0 {
		return &storage.KeyVaultProperties{}
	}

	v := vs[0].(map[string]interface{})
	keyName := v["key_name"].(string)
	keyVersion := v["key_version"].(string)

	return &storage.KeyVaultProperties{
		KeyName:    utils.String(keyName),
		KeyVersion: utils.String(keyVersion),
	}
}

func flattenAzureRmStorageAccountKeyVaultProperties(keyVaultProperties *storage.KeyVaultProperties, keyVaultId string, keyVaultPolicyId string) []interface{} {
	if keyVaultProperties == nil {
		return make([]interface{}, 0)
	}

	result := make(map[string]interface{})
	if keyVaultId != "" {
		result["key_vault_id"] = keyVaultId
	}

	if keyVaultPolicyId != "" {
		result["key_vault_policy_id"] = keyVaultPolicyId
	}

	if keyVaultProperties.KeyName != nil {
		result["key_name"] = *keyVaultProperties.KeyName
	}
	if keyVaultProperties.KeyVersion != nil {
		result["key_version"] = *keyVaultProperties.KeyVersion
	}
	if keyVaultProperties.KeyVaultURI != nil {
		result["key_vault_uri"] = *keyVaultProperties.KeyVaultURI
	}

	return []interface{}{result}
}
