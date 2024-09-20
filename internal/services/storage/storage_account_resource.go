// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package storage

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/edgezones"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/identity"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/location"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/tags"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storage/2023-01-01/blobservice"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storage/2023-01-01/storageaccounts"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/features"
	"github.com/hashicorp/terraform-provider-azurerm/internal/locks"
	keyVaultClient "github.com/hashicorp/terraform-provider-azurerm/internal/services/keyvault/client"
	keyVaultParse "github.com/hashicorp/terraform-provider-azurerm/internal/services/keyvault/parse"
	keyVaultValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/keyvault/validate"
	managedHsmParse "github.com/hashicorp/terraform-provider-azurerm/internal/services/managedhsm/parse"
	managedHsmValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/managedhsm/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/network"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/storage/helpers"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/storage/migration"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/storage/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

const dataPlaneDependentPropertyError = `the '%[1]s' code block cannot be set on create for new storage accounts, this property has been deprecated and will become computed only in version 5.0 of the provider
New storage accounts must use the separate '%[2]s' resource to manage the '%[1]s' configuration values
Existing storage accounts can continue to use the exposed '%[1]s' code block to manage (e.g., update) the storage account resource in version 3.117.0 and version 4.0 of the provider`

var (
	storageAccountResourceName  = "azurerm_storage_account"
	storageKindsSupportsSkuTier = map[storageaccounts.Kind]struct{}{
		storageaccounts.KindBlobStorage: {},
		storageaccounts.KindFileStorage: {},
		storageaccounts.KindStorageVTwo: {},
	}
	storageKindsSupportHns = map[storageaccounts.Kind]struct{}{
		storageaccounts.KindBlobStorage:      {},
		storageaccounts.KindBlockBlobStorage: {},
		storageaccounts.KindStorageVTwo:      {},
	}
	storageKindsSupportLargeFileShares = map[storageaccounts.Kind]struct{}{
		storageaccounts.KindFileStorage: {},
		storageaccounts.KindStorageVTwo: {},
	}
	initialDelayDuration = 10 * time.Second
)

func resourceStorageAccount() *pluginsdk.Resource {
	resource := &pluginsdk.Resource{
		Create: resourceStorageAccountCreate,
		Read:   resourceStorageAccountRead,
		Update: resourceStorageAccountUpdate,
		Delete: resourceStorageAccountDelete,

		SchemaVersion: 4,
		StateUpgraders: pluginsdk.StateUpgrades(map[int]pluginsdk.StateUpgrade{
			0: migration.AccountV0ToV1{},
			1: migration.AccountV1ToV2{},
			2: migration.AccountV2ToV3{},
			3: migration.AccountV3ToV4{},
		}),

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := commonids.ParseStorageAccountID(id)
			return err
		}),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(60 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(60 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(60 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.StorageAccountName,
			},

			"resource_group_name": commonschema.ResourceGroupName(),

			"location": commonschema.Location(),

			"account_kind": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForKind(), false),
				Default:      string(storageaccounts.KindStorageVTwo),
			},

			"account_tier": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForSkuTier(), false),
			},

			"account_replication_type": {
				Type:     pluginsdk.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"LRS",
					"ZRS",
					"GRS",
					"RAGRS",
					"GZRS",
					"RAGZRS",
				}, false),
			},

			// Only valid for FileStorage, BlobStorage & StorageV2 accounts, defaults to "Hot" in create function
			"access_tier": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForAccessTier(), false), // TODO: docs for `Premium`
			},

			"data_plane_access_on_read_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  true,
			},

			"azure_files_authentication": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"directory_type": {
							Type:     pluginsdk.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(storageaccounts.DirectoryServiceOptionsAADDS),
								string(storageaccounts.DirectoryServiceOptionsAADKERB),
								string(storageaccounts.DirectoryServiceOptionsAD),
							}, false),
						},

						"active_directory": {
							Type:     pluginsdk.TypeList,
							Optional: true,
							Computed: true,
							MaxItems: 1,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"domain_guid": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validation.IsUUID,
									},

									"domain_name": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},

									"storage_sid": {
										Type:         pluginsdk.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},

									"domain_sid": {
										Type:         pluginsdk.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},

									"forest_name": {
										Type:         pluginsdk.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},

									"netbios_domain_name": {
										Type:         pluginsdk.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},
								},
							},
						},

						"default_share_level_permission": {
							Type:         pluginsdk.TypeString,
							Optional:     true,
							Default:      string(storageaccounts.DefaultSharePermissionNone),
							ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForDefaultSharePermission(), false),
						},
					},
				},
			},

			"cross_tenant_replication_enabled": func() *pluginsdk.Schema {
				s := &pluginsdk.Schema{
					Type:     pluginsdk.TypeBool,
					Optional: true,
					Default:  false,
				}
				if !features.FourPointOhBeta() {
					s.Default = true
				}
				return s
			}(),

			"custom_domain": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"name": {
							Type:     pluginsdk.TypeString,
							Required: true,
						},

						"use_subdomain": {
							Type:     pluginsdk.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},

			"customer_managed_key": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"key_vault_key_id": {
							Type:         pluginsdk.TypeString,
							Optional:     true,
							ValidateFunc: keyVaultValidate.NestedItemIdWithOptionalVersion,
							ExactlyOneOf: []string{"customer_managed_key.0.managed_hsm_key_id", "customer_managed_key.0.key_vault_key_id"},
						},

						"managed_hsm_key_id": {
							Type:         pluginsdk.TypeString,
							Optional:     true,
							ValidateFunc: validation.Any(managedHsmValidate.ManagedHSMDataPlaneVersionedKeyID, managedHsmValidate.ManagedHSMDataPlaneVersionlessKeyID),
							ExactlyOneOf: []string{"customer_managed_key.0.managed_hsm_key_id", "customer_managed_key.0.key_vault_key_id"},
						},

						"user_assigned_identity_id": {
							Type:         pluginsdk.TypeString,
							Required:     true,
							ValidateFunc: commonids.ValidateUserAssignedIdentityID,
						},
					},
				},
			},

			"edge_zone": commonschema.EdgeZoneOptionalForceNew(),

			"https_traffic_only_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  true,
			},

			"immutability_policy": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				MaxItems: 1,
				ForceNew: true,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"allow_protected_append_writes": {
							Type:     pluginsdk.TypeBool,
							Required: true,
						},
						"period_since_creation_in_days": {
							Type:     pluginsdk.TypeInt,
							Required: true,
						},
						"state": {
							Type:         pluginsdk.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForAccountImmutabilityPolicyState(), false),
						},
					},
				},
			},

			"min_tls_version": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				Default:      string(storageaccounts.MinimumTlsVersionTLSOneTwo),
				ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForMinimumTlsVersion(), false),
			},

			"is_hns_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"nfsv3_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"allow_nested_items_to_be_public": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  true,
			},

			"shared_access_key_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  true,
			},

			"public_network_access_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  true,
			},

			"dns_endpoint_type": {
				Type:     pluginsdk.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  string(storageaccounts.DnsEndpointTypeStandard),
				ValidateFunc: validation.StringInSlice([]string{
					string(storageaccounts.DnsEndpointTypeStandard),
					string(storageaccounts.DnsEndpointTypeAzureDnsZone),
				}, false),
			},

			"default_to_oauth_authentication": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  false,
			},

			"network_rules": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"bypass": {
							Type:     pluginsdk.TypeSet,
							Optional: true,
							Computed: true,
							Elem: &pluginsdk.Schema{
								Type:         pluginsdk.TypeString,
								ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForBypass(), false),
							},
							Set: pluginsdk.HashString,
						},

						"ip_rules": {
							Type:     pluginsdk.TypeSet,
							Optional: true,
							Computed: true,
							Elem: &pluginsdk.Schema{
								Type:         pluginsdk.TypeString,
								ValidateFunc: validate.StorageAccountIpRule,
							},
							Set: pluginsdk.HashString,
						},

						"virtual_network_subnet_ids": {
							Type:     pluginsdk.TypeSet,
							Optional: true,
							Computed: true,
							Elem: &pluginsdk.Schema{
								Type: pluginsdk.TypeString,
							},
							Set: pluginsdk.HashString,
						},

						"default_action": {
							Type:         pluginsdk.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForDefaultAction(), false),
						},

						"private_link_access": {
							Type:     pluginsdk.TypeList,
							Optional: true,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"endpoint_resource_id": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: azure.ValidateResourceID,
									},

									"endpoint_tenant_id": {
										Type:         pluginsdk.TypeString,
										Optional:     true,
										Computed:     true,
										ValidateFunc: validation.IsUUID,
									},
								},
							},
						},
					},
				},
			},

			"identity": commonschema.SystemAssignedUserAssignedIdentityOptional(),

			"blob_properties": {
				Type:     pluginsdk.TypeList,
				Computed: true,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"change_feed_enabled": {
							Type:     pluginsdk.TypeBool,
							Computed: true,
						},

						"change_feed_retention_in_days": {
							Type:     pluginsdk.TypeInt,
							Computed: true,
						},

						"container_delete_retention_policy": {
							Type:     pluginsdk.TypeList,
							Computed: true,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"days": {
										Type:     pluginsdk.TypeInt,
										Computed: true,
									},
								},
							},
						},

						"cors_rule": helpers.SchemaStorageAccountCorsRuleComputed(),

						"default_service_version": {
							Type:     pluginsdk.TypeString,
							Computed: true,
						},

						"delete_retention_policy": {
							Type:     pluginsdk.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"days": {
										Type:     pluginsdk.TypeInt,
										Computed: true,
									},

									"permanent_delete_enabled": {
										Type:     pluginsdk.TypeBool,
										Computed: true,
									},
								},
							},
						},

						"last_access_time_enabled": {
							Type:     pluginsdk.TypeBool,
							Computed: true,
						},

						"restore_policy": {
							Type:     pluginsdk.TypeList,
							Computed: true,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"days": {
										Type:     pluginsdk.TypeInt,
										Computed: true,
									},
								},
							},
						},

						"versioning_enabled": {
							Type:     pluginsdk.TypeBool,
							Computed: true,
						},
					},
				},
			},

			"queue_properties": {
				Type:     pluginsdk.TypeList,
				Computed: true,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"cors_rule": helpers.SchemaStorageAccountCorsRuleComputed(),

						"hour_metrics": {
							Type:     pluginsdk.TypeList,
							Computed: true,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"version": {
										Type:     pluginsdk.TypeString,
										Computed: true,
									},
									"enabled": {
										Type:     pluginsdk.TypeBool,
										Computed: true,
									},
									"include_apis": {
										Type:     pluginsdk.TypeBool,
										Computed: true,
									},
									"retention_policy_days": {
										Type:     pluginsdk.TypeInt,
										Computed: true,
									},
								},
							},
						},

						"logging": {
							Type:     pluginsdk.TypeList,
							Computed: true,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"version": {
										Type:     pluginsdk.TypeString,
										Computed: true,
									},

									"delete": {
										Type:     pluginsdk.TypeBool,
										Computed: true,
									},

									"read": {
										Type:     pluginsdk.TypeBool,
										Computed: true,
									},

									"write": {
										Type:     pluginsdk.TypeBool,
										Computed: true,
									},

									"retention_policy_days": {
										Type:     pluginsdk.TypeInt,
										Computed: true,
									},
								},
							},
						},

						"minute_metrics": {
							Type:     pluginsdk.TypeList,
							Optional: true,
							Computed: true,
							MaxItems: 1,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"version": {
										Type:     pluginsdk.TypeString,
										Computed: true,
									},

									"enabled": {
										Type:     pluginsdk.TypeBool,
										Computed: true,
									},

									"include_apis": {
										Type:     pluginsdk.TypeBool,
										Computed: true,
									},

									"retention_policy_days": {
										Type:     pluginsdk.TypeInt,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},

			"routing": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"publish_internet_endpoints": {
							Type:     pluginsdk.TypeBool,
							Optional: true,
							Default:  false,
						},

						"publish_microsoft_endpoints": {
							Type:     pluginsdk.TypeBool,
							Optional: true,
							Default:  false,
						},

						"choice": {
							Type:         pluginsdk.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForRoutingChoice(), false),
							Default:      string(storageaccounts.RoutingChoiceMicrosoftRouting),
						},
					},
				},
			},

			"share_properties": {
				Type:     pluginsdk.TypeList,
				Computed: true,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"cors_rule": helpers.SchemaStorageAccountCorsRuleComputed(),

						"retention_policy": {
							Type:     pluginsdk.TypeList,
							Computed: true,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"days": {
										Type:     pluginsdk.TypeInt,
										Computed: true,
									},
								},
							},
						},

						"smb": {
							Type:     pluginsdk.TypeList,
							Computed: true,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"authentication_types": {
										Type:     pluginsdk.TypeSet,
										Computed: true,
										Elem: &pluginsdk.Schema{
											Type: pluginsdk.TypeString,
										},
									},

									"channel_encryption_type": {
										Type:     pluginsdk.TypeSet,
										Computed: true,
										Elem: &pluginsdk.Schema{
											Type: pluginsdk.TypeString,
										},
									},

									"kerberos_ticket_encryption_type": {
										Type:     pluginsdk.TypeSet,
										Computed: true,
										Elem: &pluginsdk.Schema{
											Type: pluginsdk.TypeString,
										},
									},

									"multichannel_enabled": {
										Type:     pluginsdk.TypeBool,
										Computed: true,
									},

									"versions": {
										Type:     pluginsdk.TypeSet,
										Computed: true,
										Elem: &pluginsdk.Schema{
											Type: pluginsdk.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},

			"static_website": {
				Type:     pluginsdk.TypeList,
				Computed: true,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"error_404_document": {
							Type:     pluginsdk.TypeString,
							Computed: true,
						},
						"index_document": {
							Type:     pluginsdk.TypeString,
							Computed: true,
						},
					},
				},
			},

			"queue_encryption_key_type": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForKeyType(), false),
				Default:      string(storageaccounts.KeyTypeService),
			},

			"table_encryption_key_type": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForKeyType(), false),
				Default:      string(storageaccounts.KeyTypeService),
			},

			"infrastructure_encryption_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"sas_policy": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"expiration_action": {
							Type:     pluginsdk.TypeString,
							Optional: true,
							Default:  string(storageaccounts.ExpirationActionLog),
							ValidateFunc: validation.StringInSlice([]string{
								string(storageaccounts.ExpirationActionLog),
							}, false),
						},
						"expiration_period": {
							Type:         pluginsdk.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},
					},
				},
			},

			"allowed_copy_scope": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(storageaccounts.PossibleValuesForAllowedCopyScope(), false),
			},

			"sftp_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  false,
			},

			"large_file_share_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Computed: true,
			},

			"local_user_enabled": {
				Type:     pluginsdk.TypeBool,
				Optional: true,
				Default:  true,
			},

			"primary_location": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_location": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_blob_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_blob_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_blob_internet_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_blob_internet_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_blob_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_blob_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_blob_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_blob_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_blob_internet_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_blob_internet_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_blob_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_blob_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_queue_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_queue_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_queue_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_queue_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_queue_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_queue_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_queue_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_queue_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_table_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_table_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_table_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_table_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_table_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_table_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_table_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_table_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_web_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_web_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_web_internet_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_web_internet_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_web_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_web_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_web_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_web_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_web_internet_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_web_internet_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_web_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_web_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_dfs_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_dfs_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_dfs_internet_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_dfs_internet_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_dfs_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_dfs_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_dfs_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_dfs_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_dfs_internet_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_dfs_internet_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_dfs_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_dfs_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_file_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_file_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_file_internet_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_file_internet_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_file_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_file_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_file_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_file_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_file_internet_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_file_internet_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_file_microsoft_host": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"secondary_file_microsoft_endpoint": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"primary_access_key": {
				Type:      pluginsdk.TypeString,
				Sensitive: true,
				Computed:  true,
			},

			"secondary_access_key": {
				Type:      pluginsdk.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"primary_connection_string": {
				Type:      pluginsdk.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"secondary_connection_string": {
				Type:      pluginsdk.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"primary_blob_connection_string": {
				Type:      pluginsdk.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"secondary_blob_connection_string": {
				Type:      pluginsdk.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"tags": {
				// TODO: introduce/refactor this to use a `commonschema.TagsOptionalWith(a, b, c)` to enable us to handle this in one place
				Type:         pluginsdk.TypeMap,
				Optional:     true,
				ValidateFunc: validate.StorageAccountTags,
				Elem: &pluginsdk.Schema{
					Type: pluginsdk.TypeString,
				},
			},
		},
		CustomizeDiff: pluginsdk.CustomDiffWithAll(
			pluginsdk.CustomizeDiffShim(func(ctx context.Context, d *pluginsdk.ResourceDiff, v interface{}) error {
				if d.HasChange("account_kind") {
					accountKind, changedKind := d.GetChange("account_kind")

					if accountKind != string(storageaccounts.KindStorage) && changedKind != string(storageaccounts.KindStorageVTwo) {
						log.Printf("[DEBUG] recreate storage account, could not be migrated from %q to %q", accountKind, changedKind)
						d.ForceNew("account_kind")
						return nil
					} else {
						log.Printf("[DEBUG] storage account can be upgraded from %q to %q", accountKind, changedKind)
					}
				}

				if d.HasChange("large_file_share_enabled") {
					lfsEnabled, changedEnabled := d.GetChange("large_file_share_enabled")
					if lfsEnabled.(bool) && !changedEnabled.(bool) {
						d.ForceNew("large_file_share_enabled")
					}
				}

				if d.Get("access_tier") != "" {
					accountKind := storageaccounts.Kind(d.Get("account_kind").(string))
					if _, ok := storageKindsSupportsSkuTier[accountKind]; !ok {
						keys := sortedKeysFromSlice(storageKindsSupportsSkuTier)
						return fmt.Errorf("`access_tier` is only available for accounts where `kind` is set to one of: %+v", strings.Join(keys, " / "))
					}
				}

				return nil
			}),
			pluginsdk.ForceNewIfChange("account_replication_type", func(ctx context.Context, old, new, meta interface{}) bool {
				newAccRep := strings.ToUpper(new.(string))

				switch strings.ToUpper(old.(string)) {
				case "LRS", "GRS", "RAGRS":
					if newAccRep == "GZRS" || newAccRep == "RAGZRS" || newAccRep == "ZRS" {
						return true
					}
				case "ZRS", "GZRS", "RAGZRS":
					if newAccRep == "LRS" || newAccRep == "GRS" || newAccRep == "RAGRS" {
						return true
					}
				}
				return false
			}),
		),
	}

	if !features.FourPointOhBeta() {
		resource.Schema["blob_properties"] = &pluginsdk.Schema{
			Type:     pluginsdk.TypeList,
			Optional: true,
			Computed: true,
			MaxItems: 1,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"change_feed_enabled": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
						Default:  false,
					},

					"change_feed_retention_in_days": {
						Type:         pluginsdk.TypeInt,
						Optional:     true,
						ValidateFunc: validation.IntBetween(1, 146000),
					},

					"container_delete_retention_policy": {
						Type:     pluginsdk.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"days": {
									Type:         pluginsdk.TypeInt,
									Optional:     true,
									Default:      7,
									ValidateFunc: validation.IntBetween(1, 365),
								},
							},
						},
					},

					"cors_rule": helpers.SchemaStorageAccountCorsRule(true),

					"default_service_version": {
						Type:         pluginsdk.TypeString,
						Optional:     true,
						Computed:     true,
						ValidateFunc: validate.BlobPropertiesDefaultServiceVersion,
					},

					"delete_retention_policy": {
						Type:     pluginsdk.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"days": {
									Type:         pluginsdk.TypeInt,
									Optional:     true,
									Default:      7,
									ValidateFunc: validation.IntBetween(1, 365),
								},
								"permanent_delete_enabled": {
									Type:     pluginsdk.TypeBool,
									Optional: true,
									Default:  false,
								},
							},
						},
					},
					"last_access_time_enabled": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
						Default:  false,
					},

					"restore_policy": {
						Type:     pluginsdk.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"days": {
									Type:         pluginsdk.TypeInt,
									Required:     true,
									ValidateFunc: validation.IntBetween(1, 365),
								},
							},
						},
						RequiredWith: []string{"blob_properties.0.delete_retention_policy"},
					},

					"versioning_enabled": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
						Default:  false,
					},
				},
			},
		}

		resource.Schema["queue_properties"] = &pluginsdk.Schema{
			Type:       pluginsdk.TypeList,
			Optional:   true,
			Computed:   true,
			Deprecated: "the `queue_properties` code block requires reaching out to the dataplane, to better support private endpoints and storage accounts with public network access disabled, new storage accounts will be required to use the `azurerm_storage_account_queue_properties` resource instead of the exposed `queue_properties` code block in the storage account resource itself.",
			MaxItems:   1,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"cors_rule": helpers.SchemaStorageAccountCorsRule(false),

					"hour_metrics": {
						Type:     pluginsdk.TypeList,
						Optional: true,
						Computed: true,
						MaxItems: 1,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"version": {
									Type:         pluginsdk.TypeString,
									Required:     true,
									ValidateFunc: validation.StringIsNotEmpty,
								},

								// TODO 4.0: Remove this property and determine whether to enable based on existence of the out side block.
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},

								"include_apis": {
									Type:     pluginsdk.TypeBool,
									Optional: true,
								},

								"retention_policy_days": {
									Type:         pluginsdk.TypeInt,
									Optional:     true,
									ValidateFunc: validation.IntBetween(1, 365),
								},
							},
						},
					},

					"logging": {
						Type:     pluginsdk.TypeList,
						Optional: true,
						Computed: true,
						MaxItems: 1,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"version": {
									Type:         pluginsdk.TypeString,
									Required:     true,
									ValidateFunc: validation.StringIsNotEmpty,
								},

								"delete": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},

								"read": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},

								"write": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},

								"retention_policy_days": {
									Type:         pluginsdk.TypeInt,
									Optional:     true,
									ValidateFunc: validation.IntBetween(1, 365),
								},
							},
						},
					},

					"minute_metrics": {
						Type:     pluginsdk.TypeList,
						Optional: true,
						Computed: true,
						MaxItems: 1,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"version": {
									Type:         pluginsdk.TypeString,
									Required:     true,
									ValidateFunc: validation.StringIsNotEmpty,
								},

								// TODO 4.0: Remove this property and determine whether to enable based on existence of the out side block.
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},

								"include_apis": {
									Type:     pluginsdk.TypeBool,
									Optional: true,
								},

								"retention_policy_days": {
									Type:         pluginsdk.TypeInt,
									Optional:     true,
									ValidateFunc: validation.IntBetween(1, 365),
								},
							},
						},
					},
				},
			},
		}

		resource.Schema["share_properties"] = &pluginsdk.Schema{
			Type:     pluginsdk.TypeList,
			Optional: true,
			Computed: true,
			MaxItems: 1,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"cors_rule": helpers.SchemaStorageAccountCorsRule(true),

					"retention_policy": {
						Type:     pluginsdk.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"days": {
									Type:         pluginsdk.TypeInt,
									Optional:     true,
									Default:      7,
									ValidateFunc: validation.IntBetween(1, 365),
								},
							},
						},
					},

					"smb": {
						Type:     pluginsdk.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"authentication_types": {
									Type:     pluginsdk.TypeSet,
									Optional: true,
									Elem: &pluginsdk.Schema{
										Type: pluginsdk.TypeString,
										ValidateFunc: validation.StringInSlice([]string{
											"Kerberos",
											"NTLMv2",
										}, false),
									},
								},

								"channel_encryption_type": {
									Type:     pluginsdk.TypeSet,
									Optional: true,
									Elem: &pluginsdk.Schema{
										Type: pluginsdk.TypeString,
										ValidateFunc: validation.StringInSlice([]string{
											"AES-128-CCM",
											"AES-128-GCM",
											"AES-256-GCM",
										}, false),
									},
								},

								"kerberos_ticket_encryption_type": {
									Type:     pluginsdk.TypeSet,
									Optional: true,
									Elem: &pluginsdk.Schema{
										Type: pluginsdk.TypeString,
										ValidateFunc: validation.StringInSlice([]string{
											"AES-256",
											"RC4-HMAC",
										}, false),
									},
								},

								"multichannel_enabled": {
									Type:     pluginsdk.TypeBool,
									Optional: true,
									Default:  false,
								},

								"versions": {
									Type:     pluginsdk.TypeSet,
									Optional: true,
									Elem: &pluginsdk.Schema{
										Type: pluginsdk.TypeString,
										ValidateFunc: validation.StringInSlice([]string{
											"SMB2.1",
											"SMB3.0",
											"SMB3.1.1",
										}, false),
									},
								},
							},
						},
					},
				},
			},
		}

		// lintignore:XS003
		resource.Schema["static_website"] = &pluginsdk.Schema{
			Type:       pluginsdk.TypeList,
			Optional:   true,
			Computed:   true,
			Deprecated: "the `static_website` field requires reaching out to the dataplane, to better support private endpoints and storage accounts with public network access disabled, new storage accounts will be required to use the `azurerm_storage_account_static_website_properties` resource instead of the exposed `static_website` code block in the storage account resource itself.",
			MaxItems:   1,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"error_404_document": {
						Type:         pluginsdk.TypeString,
						Optional:     true,
						ValidateFunc: validation.StringIsNotEmpty,
					},
					"index_document": {
						Type:         pluginsdk.TypeString,
						Optional:     true,
						ValidateFunc: validation.StringIsNotEmpty,
					},
				},
			},
		}

		resource.Schema["https_traffic_only_enabled"].Computed = true
		resource.Schema["https_traffic_only_enabled"].Default = nil

		resource.Schema["enable_https_traffic_only"] = &pluginsdk.Schema{
			Type:          pluginsdk.TypeBool,
			Optional:      true,
			Computed:      true,
			ConflictsWith: []string{"https_traffic_only_enabled"},
			Deprecated:    "The property `enable_https_traffic_only` has been superseded by `https_traffic_only_enabled` and will be removed in v4.0 of the AzureRM Provider.",
		}
	}

	return resource
}

func resourceStorageAccountCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	tenantId := meta.(*clients.Client).Account.TenantId
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	storageClient := meta.(*clients.Client).Storage
	client := meta.(*clients.Client).Storage.ResourceManager.StorageAccounts
	keyVaultClient := meta.(*clients.Client).KeyVault
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	if !features.FourPointOh() {
		// NOTE: We want to block creation of all new storage accounts in v3.x that have 'blob_properties'
		// 'queue_properties', 'share_properties' and/or 'static_website' fields defined within the
		// configuration file...
		if _, ok := d.GetOk("blob_properties"); ok {
			return fmt.Errorf(dataPlaneDependentPropertyError, "blob_properties", storageAccountBlobPropertiesResourceName)
		}

		if _, ok := d.GetOk("queue_properties"); ok {
			return fmt.Errorf(dataPlaneDependentPropertyError, "queue_properties", storageAccountQueuePropertiesResourceName)
		}

		if _, ok := d.GetOk("share_properties"); ok {
			return fmt.Errorf(dataPlaneDependentPropertyError, "share_properties", storageAccountSharePropertiesResourceName)
		}

		if _, ok := d.GetOk("static_website"); ok {
			return fmt.Errorf(dataPlaneDependentPropertyError, "static_website", storageAccountStaticWebSitePropertiesResourceName)
		}
	}

	id := commonids.NewStorageAccountID(subscriptionId, d.Get("resource_group_name").(string), d.Get("name").(string))
	locks.ByName(id.StorageAccountName, storageAccountResourceName)
	defer locks.UnlockByName(id.StorageAccountName, storageAccountResourceName)

	existing, err := client.GetProperties(ctx, id, storageaccounts.DefaultGetPropertiesOperationOptions())
	if err != nil {
		if !response.WasNotFound(existing.HttpResponse) {
			return fmt.Errorf("checking for existing %s: %+v", id, err)
		}
	}
	if !response.WasNotFound(existing.HttpResponse) {
		return tf.ImportAsExistsError(storageAccountResourceName, id.ID())
	}

	accountKind := storageaccounts.Kind(d.Get("account_kind").(string))
	accountTier := storageaccounts.SkuTier(d.Get("account_tier").(string))
	replicationType := d.Get("account_replication_type").(string)

	publicNetworkAccess := storageaccounts.PublicNetworkAccessDisabled
	if d.Get("public_network_access_enabled").(bool) {
		publicNetworkAccess = storageaccounts.PublicNetworkAccessEnabled
	}
	expandedIdentity, err := identity.ExpandLegacySystemAndUserAssignedMap(d.Get("identity").([]interface{}))
	if err != nil {
		return fmt.Errorf("expanding `identity`: %+v", err)
	}

	httpsTrafficOnlyEnabled := true
	// nolint staticcheck
	if v, ok := d.GetOkExists("https_traffic_only_enabled"); ok {
		httpsTrafficOnlyEnabled = v.(bool)
	} else if !features.FourPointOhBeta() {
		// nolint staticcheck
		if v, ok := d.GetOkExists("enable_https_traffic_only"); ok {
			httpsTrafficOnlyEnabled = v.(bool)
		}
	}

	dnsEndpointType := d.Get("dns_endpoint_type").(string)
	isHnsEnabled := d.Get("is_hns_enabled").(bool)
	nfsV3Enabled := d.Get("nfsv3_enabled").(bool)
	payload := storageaccounts.StorageAccountCreateParameters{
		ExtendedLocation: expandEdgeZone(d.Get("edge_zone").(string)),
		Kind:             accountKind,
		Identity:         expandedIdentity,
		Location:         location.Normalize(d.Get("location").(string)),
		Properties: &storageaccounts.StorageAccountPropertiesCreateParameters{
			AllowBlobPublicAccess:        pointer.To(d.Get("allow_nested_items_to_be_public").(bool)),
			AllowCrossTenantReplication:  pointer.To(d.Get("cross_tenant_replication_enabled").(bool)),
			AllowSharedKeyAccess:         pointer.To(d.Get("shared_access_key_enabled").(bool)),
			DnsEndpointType:              pointer.To(storageaccounts.DnsEndpointType(dnsEndpointType)),
			DefaultToOAuthAuthentication: pointer.To(d.Get("default_to_oauth_authentication").(bool)),
			SupportsHTTPSTrafficOnly:     pointer.To(httpsTrafficOnlyEnabled),
			IsNfsV3Enabled:               pointer.To(nfsV3Enabled),
			IsHnsEnabled:                 pointer.To(isHnsEnabled),
			IsLocalUserEnabled:           pointer.To(d.Get("local_user_enabled").(bool)),
			IsSftpEnabled:                pointer.To(d.Get("sftp_enabled").(bool)),
			MinimumTlsVersion:            pointer.To(storageaccounts.MinimumTlsVersion(d.Get("min_tls_version").(string))),
			NetworkAcls:                  expandAccountNetworkRules(d.Get("network_rules").([]interface{}), tenantId),
			PublicNetworkAccess:          pointer.To(publicNetworkAccess),
			SasPolicy:                    expandAccountSASPolicy(d.Get("sas_policy").([]interface{})),
		},
		Sku: storageaccounts.Sku{
			Name: storageaccounts.SkuName(fmt.Sprintf("%s_%s", string(accountTier), replicationType)),
			Tier: pointer.To(accountTier),
		},
		Tags: tags.Expand(d.Get("tags").(map[string]interface{})),
	}

	if v := d.Get("allowed_copy_scope").(string); v != "" {
		payload.Properties.AllowedCopyScope = pointer.To(storageaccounts.AllowedCopyScope(v))
	}
	if v, ok := d.GetOk("azure_files_authentication"); ok {
		expandAADFilesAuthentication, err := expandAccountAzureFilesAuthentication(v.([]interface{}))
		if err != nil {
			return fmt.Errorf("parsing `azure_files_authentication`: %v", err)
		}
		payload.Properties.AzureFilesIdentityBasedAuthentication = expandAADFilesAuthentication
	}
	if _, ok := d.GetOk("custom_domain"); ok {
		payload.Properties.CustomDomain = expandAccountCustomDomain(d.Get("custom_domain").([]interface{}))
	}
	if v, ok := d.GetOk("immutability_policy"); ok {
		payload.Properties.ImmutableStorageWithVersioning = expandAccountImmutabilityPolicy(v.([]interface{}))
	}

	// BlobStorage does not support ZRS
	if accountKind == storageaccounts.KindBlobStorage && string(payload.Sku.Name) == string(storageaccounts.SkuNameStandardZRS) {
		return fmt.Errorf("`account_replication_type` of `ZRS` isn't supported for Blob Storage accounts")
	}

	accessTier, accessTierSetInConfig := d.GetOk("access_tier")
	_, skuTierSupported := storageKindsSupportsSkuTier[accountKind]
	if !skuTierSupported && accessTierSetInConfig {
		keys := sortedKeysFromSlice(storageKindsSupportsSkuTier)
		return fmt.Errorf("`access_tier` is only available for accounts of kind set to one of: %+v", strings.Join(keys, " / "))
	}

	if skuTierSupported {
		if !accessTierSetInConfig {
			// default to "Hot"
			accessTier = string(storageaccounts.AccessTierHot)
		}
		payload.Properties.AccessTier = pointer.To(storageaccounts.AccessTier(accessTier.(string)))
	}

	if _, supportsHns := storageKindsSupportHns[accountKind]; !supportsHns && isHnsEnabled {
		keys := sortedKeysFromSlice(storageKindsSupportHns)
		return fmt.Errorf("`is_hns_enabled` can only be used for accounts with `kind` set to one of: %+v", strings.Join(keys, " / "))
	}

	// NFSv3 is supported for standard general-purpose v2 storage accounts and for premium block blob storage accounts.
	// (https://docs.microsoft.com/en-us/azure/storage/blobs/network-file-system-protocol-support-how-to#step-5-create-and-configure-a-storage-account)
	if nfsV3Enabled {
		if !isHnsEnabled {
			return fmt.Errorf("`nfsv3_enabled` can only be used when `is_hns_enabled` is `true`")
		}

		isPremiumTierAndBlockBlobStorageKind := accountTier == storageaccounts.SkuTierPremium && accountKind == storageaccounts.KindBlockBlobStorage
		isStandardTierAndStorageV2Kind := accountTier == storageaccounts.SkuTierStandard && accountKind == storageaccounts.KindStorageVTwo
		if !isPremiumTierAndBlockBlobStorageKind && !isStandardTierAndStorageV2Kind {
			return fmt.Errorf("`nfsv3_enabled` can only be used with account tier `Standard` and account kind `StorageV2`, or account tier `Premium` and account kind `BlockBlobStorage`")
		}
	}

	// AccountTier must be Premium for FileStorage
	if accountKind == storageaccounts.KindFileStorage && accountTier != storageaccounts.SkuTierPremium {
		return fmt.Errorf("`account_tier` must be `Premium` for File Storage accounts")
	}

	// nolint staticcheck
	if v, ok := d.GetOkExists("large_file_share_enabled"); ok {
		// @tombuildsstuff: we cannot set this to `false` because the API returns:
		//
		// performing Create: unexpected status 400 (400 Bad Request) with error: InvalidRequestPropertyValue: The
		// value 'Disabled' is not allowed for property largeFileSharesState. For more information, see -
		// https://aka.ms/storageaccountlargefilesharestate
		if v.(bool) {
			if _, ok := storageKindsSupportLargeFileShares[accountKind]; !ok {
				keys := sortedKeysFromSlice(storageKindsSupportLargeFileShares)
				return fmt.Errorf("`large_file_shares_enabled` can only be set to `true` with `account_kind` set to one of: %+v", strings.Join(keys, " / "))
			}
			payload.Properties.LargeFileSharesState = pointer.To(storageaccounts.LargeFileSharesStateEnabled)
		}
	}

	if v, ok := d.GetOk("routing"); ok {
		payload.Properties.RoutingPreference = expandAccountRoutingPreference(v.([]interface{}))
	}

	// TODO 4.0: look into standardizing this across resources that support CMK and at the very least look at improving the UX
	// for encryption of blob, file, table and queue
	//
	// By default (by leaving empty), the table and queue encryption key type is set to "Service". While users can change it to "Account" so that
	// they can further use CMK to encrypt table/queue data. Only the StorageV2 account kind supports the Account key type.
	// Also noted that the blob and file are always using the "Account" key type.
	// See: https://docs.microsoft.com/en-gb/azure/storage/common/account-encryption-key-create?tabs=portal
	queueEncryptionKeyType := storageaccounts.KeyType(d.Get("queue_encryption_key_type").(string))
	tableEncryptionKeyType := storageaccounts.KeyType(d.Get("table_encryption_key_type").(string))
	encryptionRaw := d.Get("customer_managed_key").([]interface{})
	encryption, err := expandAccountCustomerManagedKey(ctx, keyVaultClient, id.SubscriptionId, encryptionRaw, accountTier, accountKind, *expandedIdentity, queueEncryptionKeyType, tableEncryptionKeyType)
	if err != nil {
		return fmt.Errorf("expanding `customer_managed_key`: %+v", err)
	}

	infrastructureEncryption := d.Get("infrastructure_encryption_enabled").(bool)

	if infrastructureEncryption {
		validPremiumConfiguration := accountTier == storageaccounts.SkuTierPremium && (accountKind == storageaccounts.KindBlockBlobStorage) || accountKind == storageaccounts.KindFileStorage
		validV2Configuration := accountKind == storageaccounts.KindStorageVTwo
		if !(validPremiumConfiguration || validV2Configuration) {
			return fmt.Errorf("`infrastructure_encryption_enabled` can only be used with account kind `StorageV2`, or account tier `Premium` and account kind is one of `BlockBlobStorage` or `FileStorage`")
		}
		encryption.RequireInfrastructureEncryption = &infrastructureEncryption
	}

	payload.Properties.Encryption = encryption

	if err := client.CreateThenPoll(ctx, id, payload); err != nil {
		return fmt.Errorf("creating %s: %+v", id, err)
	}
	d.SetId(id.ID())
	d.Set("data_plane_access_on_read_enabled", d.Get("data_plane_access_on_read_enabled").(bool))

	log.Printf("[DEBUG] [%s:CREATE] Calling 'client.GetProperties': %s", strings.ToUpper(storageAccountResourceName), id)
	account, err := client.GetProperties(ctx, id, storageaccounts.DefaultGetPropertiesOperationOptions())
	if err != nil {
		return fmt.Errorf("retrieving %s: %+v", id, err)
	}

	if account.Model == nil {
		return fmt.Errorf("retrieving %s: `model` was nil", id)
	}

	log.Printf("[DEBUG] [%s:CREATE] Calling 'storageClient.AddToCache': %s", strings.ToUpper(storageAccountResourceName), id)
	if err := storageClient.AddToCache(id, *account.Model); err != nil {
		return fmt.Errorf("populating cache for %s: %+v", id, err)
	}

	return resourceStorageAccountRead(d, meta)
}

func resourceStorageAccountUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	tenantId := meta.(*clients.Client).Account.TenantId
	storageClient := meta.(*clients.Client).Storage
	client := storageClient.ResourceManager.StorageAccounts
	keyVaultClient := meta.(*clients.Client).KeyVault
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := commonids.ParseStorageAccountID(d.Id())
	if err != nil {
		return err
	}

	locks.ByName(id.StorageAccountName, storageAccountResourceName)
	defer locks.UnlockByName(id.StorageAccountName, storageAccountResourceName)

	accountTier := storageaccounts.SkuTier(d.Get("account_tier").(string))
	replicationType := d.Get("account_replication_type").(string)
	storageType := fmt.Sprintf("%s_%s", accountTier, replicationType)
	accountKind := storageaccounts.Kind(d.Get("account_kind").(string))

	if accountKind == storageaccounts.KindBlobStorage || accountKind == storageaccounts.KindStorage {
		if storageType == string(storageaccounts.SkuNameStandardZRS) {
			return fmt.Errorf("an `account_replication_type` of `ZRS` isn't supported for Blob Storage accounts")
		}
	}

	existing, err := client.GetProperties(ctx, *id, storageaccounts.DefaultGetPropertiesOperationOptions())
	if err != nil {
		return fmt.Errorf("retrieving %s: %+v", id, err)
	}

	if existing.Model == nil {
		return fmt.Errorf("retrieving %s: `model` was nil", id)
	}
	if existing.Model.Kind == nil {
		return fmt.Errorf("retrieving %s: `model.Kind` was nil", id)
	}
	if existing.Model.Properties == nil {
		return fmt.Errorf("retrieving %s: `model.Properties` was nil", id)
	}
	if existing.Model.Sku == nil {
		return fmt.Errorf("retrieving %s: `model.Sku` was nil", id)
	}

	props := storageaccounts.StorageAccountPropertiesCreateParameters{
		AccessTier:                            existing.Model.Properties.AccessTier,
		AllowBlobPublicAccess:                 existing.Model.Properties.AllowBlobPublicAccess,
		AllowedCopyScope:                      existing.Model.Properties.AllowedCopyScope,
		AllowSharedKeyAccess:                  existing.Model.Properties.AllowSharedKeyAccess,
		AllowCrossTenantReplication:           existing.Model.Properties.AllowCrossTenantReplication,
		AzureFilesIdentityBasedAuthentication: existing.Model.Properties.AzureFilesIdentityBasedAuthentication,
		CustomDomain:                          existing.Model.Properties.CustomDomain,
		DefaultToOAuthAuthentication:          existing.Model.Properties.DefaultToOAuthAuthentication,
		DnsEndpointType:                       existing.Model.Properties.DnsEndpointType,
		Encryption:                            existing.Model.Properties.Encryption,
		KeyPolicy:                             existing.Model.Properties.KeyPolicy,
		ImmutableStorageWithVersioning:        existing.Model.Properties.ImmutableStorageWithVersioning,
		IsNfsV3Enabled:                        existing.Model.Properties.IsNfsV3Enabled,
		IsSftpEnabled:                         existing.Model.Properties.IsSftpEnabled,
		IsLocalUserEnabled:                    existing.Model.Properties.IsLocalUserEnabled,
		IsHnsEnabled:                          existing.Model.Properties.IsHnsEnabled,
		MinimumTlsVersion:                     existing.Model.Properties.MinimumTlsVersion,
		NetworkAcls:                           existing.Model.Properties.NetworkAcls,
		PublicNetworkAccess:                   existing.Model.Properties.PublicNetworkAccess,
		RoutingPreference:                     existing.Model.Properties.RoutingPreference,
		SasPolicy:                             existing.Model.Properties.SasPolicy,
		SupportsHTTPSTrafficOnly:              existing.Model.Properties.SupportsHTTPSTrafficOnly,
	}

	if existing.Model.Properties.LargeFileSharesState != nil && *existing.Model.Properties.LargeFileSharesState == storageaccounts.LargeFileSharesStateEnabled {
		// We can only set this if it's Enabled, else the API complains during Update that we're sending Disabled, even if it's always been off
		props.LargeFileSharesState = existing.Model.Properties.LargeFileSharesState
	}

	expandedIdentity := existing.Model.Identity
	if d.HasChange("identity") {
		expandedIdentity, err = identity.ExpandLegacySystemAndUserAssignedMap(d.Get("identity").([]interface{}))
		if err != nil {
			return fmt.Errorf("expanding `identity`: %+v", err)
		}
	}

	if d.HasChange("access_tier") {
		props.AccessTier = pointer.To(storageaccounts.AccessTier(d.Get("access_tier").(string)))
	}
	if d.HasChange("allowed_copy_scope") {
		props.AllowedCopyScope = pointer.To(storageaccounts.AllowedCopyScope(d.Get("allowed_copy_scope").(string)))
	}
	if d.HasChange("allow_nested_items_to_be_public") {
		props.AllowBlobPublicAccess = pointer.To(d.Get("allow_nested_items_to_be_public").(bool))
	}
	if d.HasChange("cross_tenant_replication_enabled") {
		props.AllowCrossTenantReplication = pointer.To(d.Get("cross_tenant_replication_enabled").(bool))
	}
	if d.HasChange("custom_domain") {
		props.CustomDomain = expandAccountCustomDomain(d.Get("custom_domain").([]interface{}))
	}
	if d.HasChange("customer_managed_key") {
		queueEncryptionKeyType := storageaccounts.KeyType(d.Get("queue_encryption_key_type").(string))
		tableEncryptionKeyType := storageaccounts.KeyType(d.Get("table_encryption_key_type").(string))
		encryptionRaw := d.Get("customer_managed_key").([]interface{})
		encryption, err := expandAccountCustomerManagedKey(ctx, keyVaultClient, id.SubscriptionId, encryptionRaw, accountTier, accountKind, *expandedIdentity, queueEncryptionKeyType, tableEncryptionKeyType)
		if err != nil {
			return fmt.Errorf("expanding `customer_managed_key`: %+v", err)
		}

		// When updating CMK the existing value for `RequireInfrastructureEncryption` gets overwritten which results in
		// an error from the API so we set this back into encryption after it's been overwritten by this update
		existingEnc := existing.Model.Properties.Encryption
		if existingEnc != nil && existingEnc.RequireInfrastructureEncryption != nil {
			encryption.RequireInfrastructureEncryption = existingEnc.RequireInfrastructureEncryption
		}

		props.Encryption = encryption
	}
	if d.HasChange("shared_access_key_enabled") {
		props.AllowSharedKeyAccess = pointer.To(d.Get("shared_access_key_enabled").(bool))
	} else {
		// If AllowSharedKeyAccess is nil that breaks the Portal UI as reported in https://github.com/hashicorp/terraform-provider-azurerm/issues/11689
		// currently the Portal UI reports nil as false, and per the ARM API documentation nil is true. This manifests itself in the Portal UI
		// when a storage account is created by terraform that the AllowSharedKeyAccess is Disabled when it is actually Enabled, thus confusing out customers
		// to fix this, I have added this code to explicitly to set the value to true if is nil to workaround the Portal UI bug for our customers.
		// this is designed as a passive change, meaning the change will only take effect when the existing storage account is modified in some way if the
		// account already exists. since I have also switched up the default behaviour for net new storage accounts to always set this value as true, this issue
		// should automatically correct itself over time with these changes.
		// TODO: Remove code when Portal UI team fixes their code
		if sharedKeyAccess := props.AllowSharedKeyAccess; sharedKeyAccess == nil {
			props.AllowSharedKeyAccess = pointer.To(true)
		}
	}
	if d.HasChange("default_to_oauth_authentication") {
		props.DefaultToOAuthAuthentication = pointer.To(d.Get("default_to_oauth_authentication").(bool))
	}

	if d.HasChange("https_traffic_only_enabled") {
		props.SupportsHTTPSTrafficOnly = pointer.To(d.Get("https_traffic_only_enabled").(bool))
	}

	if !features.FourPointOhBeta() {
		if d.HasChange("enable_https_traffic_only") {
			props.SupportsHTTPSTrafficOnly = pointer.To(d.Get("enable_https_traffic_only").(bool))
		}
	}

	if d.HasChange("large_file_share_enabled") {
		// largeFileSharesState can only be set to `Enabled` and not `Disabled`, even if it is currently `Disabled`
		if oldValue, newValue := d.GetChange("large_file_share_enabled"); oldValue.(bool) && !newValue.(bool) {
			return fmt.Errorf("`large_file_share_enabled` cannot be disabled once it's been enabled")
		}

		if _, ok := storageKindsSupportLargeFileShares[accountKind]; !ok {
			keys := sortedKeysFromSlice(storageKindsSupportLargeFileShares)
			return fmt.Errorf("`large_file_shares_enabled` can only be set to `true` with `account_kind` set to one of: %+v", strings.Join(keys, " / "))
		}
		props.LargeFileSharesState = pointer.To(storageaccounts.LargeFileSharesStateEnabled)
	}

	if d.HasChange("local_user_enabled") {
		props.IsLocalUserEnabled = pointer.To(d.Get("local_user_enabled").(bool))
	}

	if d.HasChange("min_tls_version") {
		props.MinimumTlsVersion = pointer.To(storageaccounts.MinimumTlsVersion(d.Get("min_tls_version").(string)))
	}

	if d.HasChange("network_rules") {
		props.NetworkAcls = expandAccountNetworkRules(d.Get("network_rules").([]interface{}), tenantId)
	}

	if d.HasChange("public_network_access_enabled") {
		publicNetworkAccess := storageaccounts.PublicNetworkAccessDisabled
		if d.Get("public_network_access_enabled").(bool) {
			publicNetworkAccess = storageaccounts.PublicNetworkAccessEnabled
		}
		props.PublicNetworkAccess = pointer.To(publicNetworkAccess)
	}

	if d.HasChange("routing") {
		props.RoutingPreference = expandAccountRoutingPreference(d.Get("routing").([]interface{}))
	}

	if d.HasChange("sas_policy") {
		// TODO: Currently, there is no way to represent a `null` value in the payload - instead it will be omitted, `sas_policy` can not be disabled once enabled.
		props.SasPolicy = expandAccountSASPolicy(d.Get("sas_policy").([]interface{}))
	}

	if d.HasChange("sftp_enabled") {
		props.IsSftpEnabled = pointer.To(d.Get("sftp_enabled").(bool))
	}

	payload := storageaccounts.StorageAccountCreateParameters{
		ExtendedLocation: existing.Model.ExtendedLocation,
		Kind:             *existing.Model.Kind,
		Location:         existing.Model.Location,
		Identity:         existing.Model.Identity,
		Properties:       &props,
		Sku:              *existing.Model.Sku,
		Tags:             existing.Model.Tags,
	}

	// ensure any top-level properties are updated
	if d.HasChange("account_kind") {
		payload.Kind = accountKind
	}

	if d.HasChange("account_replication_type") {
		// storageType is derived from "account_replication_type" and "account_tier" (force-new)
		payload.Sku = storageaccounts.Sku{
			Name: storageaccounts.SkuName(storageType),
		}
	}

	if d.HasChange("identity") {
		payload.Identity = expandedIdentity
	}

	if d.HasChange("tags") {
		payload.Tags = tags.Expand(d.Get("tags").(map[string]interface{}))
	}

	if err := client.CreateThenPoll(ctx, *id, payload); err != nil {
		return fmt.Errorf("updating %s: %+v", id, err)
	}

	// azure_files_authentication must be the last to be updated, cause it'll occupy the storage account for several minutes after receiving the response 200 OK. Issue: https://github.com/Azure/azure-rest-api-specs/issues/11272
	if d.HasChange("azure_files_authentication") {
		// due to service issue: https://github.com/Azure/azure-rest-api-specs/issues/12473, we need to update to None before changing its DirectoryServiceOptions
		old, new := d.GetChange("azure_files_authentication.0.directory_type")
		if old != new && new != string(storageaccounts.DirectoryServiceOptionsNone) {
			log.Print("[DEBUG] Disabling AzureFilesIdentityBasedAuthentication prior to changing DirectoryServiceOptions")
			dsNone := storageaccounts.StorageAccountUpdateParameters{
				Properties: &storageaccounts.StorageAccountPropertiesUpdateParameters{
					AzureFilesIdentityBasedAuthentication: &storageaccounts.AzureFilesIdentityBasedAuthentication{
						DirectoryServiceOptions: storageaccounts.DirectoryServiceOptionsNone,
					},
				},
			}

			if _, err := client.Update(ctx, *id, dsNone); err != nil {
				return fmt.Errorf("updating `azure_files_authentication` for %s: %+v", *id, err)
			}
		}

		expandAADFilesAuthentication, err := expandAccountAzureFilesAuthentication(d.Get("azure_files_authentication").([]interface{}))
		if err != nil {
			return fmt.Errorf("expanding `azure_files_authentication`: %+v", err)
		}
		opts := storageaccounts.StorageAccountUpdateParameters{
			Properties: &storageaccounts.StorageAccountPropertiesUpdateParameters{
				AzureFilesIdentityBasedAuthentication: expandAADFilesAuthentication,
			},
		}

		if _, err := client.Update(ctx, *id, opts); err != nil {
			return fmt.Errorf("updating `azure_files_authentication` for %s: %+v", *id, err)
		}
	}

	// Followings are updates to the sub-services
	supportLevel := availableFunctionalityForAccount(accountKind, accountTier, replicationType)

	if !features.FourPointOhBeta() {
		// NOTE: Since this is an update operation, we no longer need to block on the feature flag
		// because it is safe to assume that the private endpoint, if needed, has already been
		// deployed...
		if d.HasChange("blob_properties") {
			log.Printf("[DEBUG] [%s:UPDATE] 'blob_properties': %s", strings.ToUpper(storageAccountResourceName), id)
			if !supportLevel.supportBlob {
				return fmt.Errorf("`blob_properties` are not supported for account kind %q in sku tier %q", accountKind, accountTier)
			}

			blobProperties, err := expandAccountBlobServiceProperties(accountKind, d.Get("blob_properties").([]interface{}))
			if err != nil {
				return err
			}

			if blobProperties.Properties.IsVersioningEnabled != nil && *blobProperties.Properties.IsVersioningEnabled && d.Get("is_hns_enabled").(bool) {
				return fmt.Errorf("`versioning_enabled` cannot be true when `is_hns_enabled` is true")
			}

			// Disable restore_policy first. Disabling restore_policy and while setting delete_retention_policy.allow_permanent_delete to true cause error.
			// Issue : https://github.com/Azure/azure-rest-api-specs/issues/11237
			if v := d.Get("blob_properties.0.restore_policy"); d.HasChange("blob_properties.0.restore_policy") && len(v.([]interface{})) == 0 {
				log.Printf("[DEBUG] [%s:UPDATE] Disabling 'RestorePolicy' prior to changing 'DeleteRetentionPolicy': %s", strings.ToUpper(storageAccountResourceName), id)
				blobPayload := blobservice.BlobServiceProperties{
					Properties: &blobservice.BlobServicePropertiesProperties{
						RestorePolicy: expandAccountBlobPropertiesRestorePolicy(v.([]interface{})),
					},
				}

				log.Printf("[DEBUG] [%s:UPDATE] Calling 'storageClient.ResourceManager.BlobService.SetServiceProperties' to disable 'RestorePolicy': %s", strings.ToUpper(storageAccountResourceName), id)
				if _, err := storageClient.ResourceManager.BlobService.SetServiceProperties(ctx, *id, blobPayload); err != nil {
					return fmt.Errorf("updating Azure Storage Account blob restore policy %q: %+v", id.StorageAccountName, err)
				}
			}

			if d.Get("dns_endpoint_type").(string) == string(storageaccounts.DnsEndpointTypeAzureDnsZone) {
				if blobProperties.Properties.RestorePolicy != nil && blobProperties.Properties.RestorePolicy.Enabled {
					// Otherwise, API returns: "Required feature Global Dns is disabled"
					// This is confirmed with the SRP team, where they said:
					// > restorePolicy feature is incompatible with partitioned DNS
					return fmt.Errorf("`blob_properties.restore_policy` cannot be set when `dns_endpoint_type` is set to `%s`", storageaccounts.DnsEndpointTypeAzureDnsZone)
				}
			}

			log.Printf("[DEBUG] [%s:UPDATE] Calling 'storageClient.ResourceManager.BlobService.SetServiceProperties': %s", strings.ToUpper(storageAccountResourceName), id)
			if _, err = storageClient.ResourceManager.BlobService.SetServiceProperties(ctx, *id, *blobProperties); err != nil {
				return fmt.Errorf("updating `blob_properties` for %s: %+v", *id, err)
			}
		}

		if d.HasChange("queue_properties") {
			log.Printf("[DEBUG] [%s:UPDATE] 'queue_properties': %s", strings.ToUpper(storageAccountResourceName), id)
			if !supportLevel.supportQueue {
				return fmt.Errorf("`queue_properties` are not supported for account kind %q in sku tier %q", accountKind, accountTier)
			}

			log.Printf("[DEBUG] [%s:UPDATE] Calling 'storageClient.FindAccount': %s", strings.ToUpper(storageAccountResourceName), id)
			account, err := storageClient.FindAccount(ctx, id.SubscriptionId, id.StorageAccountName)
			if err != nil {
				return fmt.Errorf("retrieving %s: %+v", *id, err)
			}
			if account == nil {
				return fmt.Errorf("unable to locate %s", *id)
			}

			log.Printf("[DEBUG] [%s:UPDATE] Calling 'storageClient.QueuesDataPlaneClient': %s", strings.ToUpper(storageAccountResourceName), id)
			queueDataPlaneClient, err := storageClient.QueuesDataPlaneClient(ctx, *account, storageClient.DataPlaneOperationSupportingAnyAuthMethod())
			if err != nil {
				return fmt.Errorf("building Queues Client: %s", err)
			}

			queueProperties, err := expandAccountQueueProperties(d.Get("queue_properties").([]interface{}))
			if err != nil {
				return fmt.Errorf("expanding `queue_properties` for %s: %+v", *id, err)
			}

			log.Printf("[DEBUG] [%s:UPDATE] Calling 'queueDataPlaneClient.UpdateServiceProperties': %s", strings.ToUpper(storageAccountResourceName), id)
			if err = queueDataPlaneClient.UpdateServiceProperties(ctx, *queueProperties); err != nil {
				return fmt.Errorf("updating Queue Properties for %s: %+v", *id, err)
			}
		}

		if d.HasChange("share_properties") {
			log.Printf("[DEBUG] [%s:UPDATE] 'share_properties': %s", strings.ToUpper(storageAccountResourceName), id)
			if !supportLevel.supportShare {
				return fmt.Errorf("`share_properties` are not supported for account kind %q in sku tier %q", accountKind, accountTier)
			}

			sharePayload := expandAccountShareProperties(d.Get("share_properties").([]interface{}))
			// The API complains if any multichannel info is sent on non premium fileshares. Even if multichannel is set to false
			if accountTier != storageaccounts.SkuTierPremium {
				// Error if the user has tried to enable multichannel on a standard tier storage account
				if sharePayload.Properties.ProtocolSettings.Smb.Multichannel != nil && sharePayload.Properties.ProtocolSettings.Smb.Multichannel.Enabled != nil {
					if *sharePayload.Properties.ProtocolSettings.Smb.Multichannel.Enabled {
						return fmt.Errorf("`multichannel_enabled` isn't supported for Standard tier Storage accounts")
					}
				}

				sharePayload.Properties.ProtocolSettings.Smb.Multichannel = nil
			}

			log.Printf("[DEBUG] [%s:UPDATE] Calling 'storageClient.ResourceManager.FileService.SetServiceProperties': %s", strings.ToUpper(storageAccountResourceName), id)
			if _, err = storageClient.ResourceManager.FileService.SetServiceProperties(ctx, *id, sharePayload); err != nil {
				return fmt.Errorf("updating File Share Properties for %s: %+v", *id, err)
			}
		}

		if d.HasChange("static_website") {
			log.Printf("[DEBUG] [%s:UPDATE] 'static_website': %s", strings.ToUpper(storageAccountResourceName), id)
			if !supportLevel.supportStaticWebsite {
				return fmt.Errorf("`static_website` are not supported for account kind %q in sku tier %q", accountKind, accountTier)
			}

			log.Printf("[DEBUG] [%s:UPDATE] Calling 'storageClient.FindAccount': %s", strings.ToUpper(storageAccountResourceName), id)
			account, err := storageClient.FindAccount(ctx, id.SubscriptionId, id.StorageAccountName)
			if err != nil {
				return fmt.Errorf("retrieving %s: %+v", *id, err)
			}
			if account == nil {
				return fmt.Errorf("unable to locate %s", *id)
			}

			log.Printf("[DEBUG] [%s:UPDATE] Calling 'storageClient.AccountsDataPlaneClient': %s", strings.ToUpper(storageAccountResourceName), id)
			accountsDataPlaneClient, err := storageClient.AccountsDataPlaneClient(ctx, *account, storageClient.DataPlaneOperationSupportingAnyAuthMethod())
			if err != nil {
				return fmt.Errorf("building Data Plane client for %s: %+v", *id, err)
			}

			staticWebsiteProps := expandAccountStaticWebsiteProperties(d.Get("static_website").([]interface{}))

			log.Printf("[DEBUG] [%s:UPDATE] Calling 'accountsDataPlaneClient.SetServiceProperties': %s", strings.ToUpper(storageAccountResourceName), id)
			if _, err = accountsDataPlaneClient.SetServiceProperties(ctx, id.StorageAccountName, staticWebsiteProps); err != nil {
				return fmt.Errorf("updating `static_website` for %s: %+v", *id, err)
			}
		}
	}

	return resourceStorageAccountRead(d, meta)
}

func resourceStorageAccountRead(d *pluginsdk.ResourceData, meta interface{}) error {
	storageClient := meta.(*clients.Client).Storage
	client := storageClient.ResourceManager.StorageAccounts
	env := meta.(*clients.Client).Account.Environment
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	storageDomainSuffix, ok := meta.(*clients.Client).Account.Environment.Storage.DomainSuffix()
	if !ok {
		return fmt.Errorf("could not determine Storage domain suffix for environment %q", meta.(*clients.Client).Account.Environment.Name)
	}

	id, err := commonids.ParseStorageAccountID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.GetProperties(ctx, *id, storageaccounts.DefaultGetPropertiesOperationOptions())
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving %s: %+v", id, err)
	}

	// we then need to find the storage account
	account, err := storageClient.FindAccount(ctx, id.SubscriptionId, id.StorageAccountName)
	if err != nil {
		return fmt.Errorf("retrieving %s: %+v", *id, err)
	}
	if account == nil {
		return fmt.Errorf("unable to locate %q", id)
	}

	listKeysOpts := storageaccounts.DefaultListKeysOperationOptions()
	listKeysOpts.Expand = pointer.To(storageaccounts.ListKeyExpandKerb)
	keys, err := client.ListKeys(ctx, *id, listKeysOpts)
	if err != nil {
		hasWriteLock := response.WasConflict(keys.HttpResponse)
		doesntHavePermissions := response.WasForbidden(keys.HttpResponse) || response.WasStatusCode(keys.HttpResponse, http.StatusUnauthorized)
		if !hasWriteLock && !doesntHavePermissions {
			return fmt.Errorf("listing Keys for %s: %+v", id, err)
		}
	}

	dataPlaneOnReadEnabled := d.Get("data_plane_access_on_read_enabled").(bool)

	d.Set("name", id.StorageAccountName)
	d.Set("resource_group_name", id.ResourceGroupName)
	d.Set("data_plane_access_on_read_enabled", dataPlaneOnReadEnabled)

	supportLevel := storageAccountServiceSupportLevel{
		supportBlob:          false,
		supportQueue:         false,
		supportShare:         false,
		supportStaticWebsite: false,
	}

	var accountKind storageaccounts.Kind
	var primaryEndpoints *storageaccounts.Endpoints
	var secondaryEndpoints *storageaccounts.Endpoints
	var routingPreference *storageaccounts.RoutingPreference

	if model := resp.Model; model != nil {
		if model.Kind != nil {
			accountKind = *model.Kind
		}
		d.Set("account_kind", string(accountKind))

		var accountTier storageaccounts.SkuTier
		accountReplicationType := ""
		if sku := model.Sku; sku != nil {
			accountReplicationType = strings.Split(string(sku.Name), "_")[1]
			if sku.Tier != nil {
				accountTier = *sku.Tier
			}
		}

		d.Set("account_tier", string(accountTier))
		d.Set("account_replication_type", accountReplicationType)
		d.Set("edge_zone", flattenEdgeZone(model.ExtendedLocation))
		d.Set("location", location.Normalize(model.Location))

		if props := model.Properties; props != nil {
			primaryEndpoints = props.PrimaryEndpoints
			routingPreference = props.RoutingPreference
			secondaryEndpoints = props.SecondaryEndpoints

			d.Set("access_tier", pointer.From(props.AccessTier))
			d.Set("allowed_copy_scope", pointer.From(props.AllowedCopyScope))
			if err := d.Set("azure_files_authentication", flattenAccountAzureFilesAuthentication(props.AzureFilesIdentityBasedAuthentication)); err != nil {
				return fmt.Errorf("setting `azure_files_authentication`: %+v", err)
			}
			d.Set("cross_tenant_replication_enabled", pointer.From(props.AllowCrossTenantReplication))
			d.Set("https_traffic_only_enabled", pointer.From(props.SupportsHTTPSTrafficOnly))
			if !features.FourPointOhBeta() {
				d.Set("enable_https_traffic_only", pointer.From(props.SupportsHTTPSTrafficOnly))
			}
			d.Set("is_hns_enabled", pointer.From(props.IsHnsEnabled))
			d.Set("nfsv3_enabled", pointer.From(props.IsNfsV3Enabled))
			d.Set("primary_location", pointer.From(props.PrimaryLocation))
			if err := d.Set("routing", flattenAccountRoutingPreference(props.RoutingPreference)); err != nil {
				return fmt.Errorf("setting `routing`: %+v", err)
			}
			d.Set("secondary_location", pointer.From(props.SecondaryLocation))
			d.Set("sftp_enabled", pointer.From(props.IsSftpEnabled))

			// NOTE: The Storage API returns `null` rather than the default value in the API response for existing
			// resources when a new field gets added - meaning we need to default the values below.
			allowBlobPublicAccess := true
			if props.AllowBlobPublicAccess != nil {
				allowBlobPublicAccess = *props.AllowBlobPublicAccess
			}
			d.Set("allow_nested_items_to_be_public", allowBlobPublicAccess)

			defaultToOAuthAuthentication := false
			if props.DefaultToOAuthAuthentication != nil {
				defaultToOAuthAuthentication = *props.DefaultToOAuthAuthentication
			}
			d.Set("default_to_oauth_authentication", defaultToOAuthAuthentication)

			dnsEndpointType := storageaccounts.DnsEndpointTypeStandard
			if props.DnsEndpointType != nil {
				dnsEndpointType = *props.DnsEndpointType
			}
			d.Set("dns_endpoint_type", dnsEndpointType)

			isLocalEnabled := true
			if props.IsLocalUserEnabled != nil {
				isLocalEnabled = *props.IsLocalUserEnabled
			}
			d.Set("local_user_enabled", isLocalEnabled)

			largeFileShareEnabled := false
			if props.LargeFileSharesState != nil {
				largeFileShareEnabled = *props.LargeFileSharesState == storageaccounts.LargeFileSharesStateEnabled
			}
			d.Set("large_file_share_enabled", largeFileShareEnabled)

			minTlsVersion := string(storageaccounts.MinimumTlsVersionTLSOneZero)
			if props.MinimumTlsVersion != nil {
				minTlsVersion = string(*props.MinimumTlsVersion)
			}
			d.Set("min_tls_version", minTlsVersion)

			publicNetworkAccessEnabled := true
			if props.PublicNetworkAccess != nil && *props.PublicNetworkAccess == storageaccounts.PublicNetworkAccessDisabled {
				publicNetworkAccessEnabled = false
			}
			d.Set("public_network_access_enabled", publicNetworkAccessEnabled)

			allowSharedKeyAccess := true
			if props.AllowSharedKeyAccess != nil {
				allowSharedKeyAccess = *props.AllowSharedKeyAccess
			}
			d.Set("shared_access_key_enabled", allowSharedKeyAccess)

			if err := d.Set("custom_domain", flattenAccountCustomDomain(props.CustomDomain)); err != nil {
				return fmt.Errorf("setting `custom_domain`: %+v", err)
			}
			if err := d.Set("immutability_policy", flattenAccountImmutabilityPolicy(props.ImmutableStorageWithVersioning)); err != nil {
				return fmt.Errorf("setting `immutability_policy`: %+v", err)
			}
			if err := d.Set("network_rules", flattenAccountNetworkRules(props.NetworkAcls)); err != nil {
				return fmt.Errorf("setting `network_rules`: %+v", err)
			}

			// When the encryption key type is "Service", the queue/table is not returned in the service list, so we default
			// the encryption key type to "Service" if it is absent (must also be the default value for "Service" in the schema)
			infrastructureEncryption := false
			queueEncryptionKeyType := string(storageaccounts.KeyTypeService)
			tableEncryptionKeyType := string(storageaccounts.KeyTypeService)
			if encryption := props.Encryption; encryption != nil {
				infrastructureEncryption = pointer.From(encryption.RequireInfrastructureEncryption)
				if encryption.Services != nil {
					if encryption.Services.Queue != nil && encryption.Services.Queue.KeyType != nil {
						queueEncryptionKeyType = string(*encryption.Services.Queue.KeyType)
					}
					if encryption.Services.Table != nil && encryption.Services.Table.KeyType != nil {
						tableEncryptionKeyType = string(*encryption.Services.Table.KeyType)
					}
				}
			}
			d.Set("infrastructure_encryption_enabled", infrastructureEncryption)
			d.Set("queue_encryption_key_type", queueEncryptionKeyType)
			d.Set("table_encryption_key_type", tableEncryptionKeyType)

			customerManagedKey := flattenAccountCustomerManagedKey(props.Encryption, env)
			if err := d.Set("customer_managed_key", customerManagedKey); err != nil {
				return fmt.Errorf("setting `customer_managed_key`: %+v", err)
			}

			if err := d.Set("sas_policy", flattenAccountSASPolicy(props.SasPolicy)); err != nil {
				return fmt.Errorf("setting `sas_policy`: %+v", err)
			}

			supportLevel = availableFunctionalityForAccount(accountKind, accountTier, accountReplicationType)
		}

		flattenedIdentity, err := identity.FlattenLegacySystemAndUserAssignedMap(model.Identity)
		if err != nil {
			return fmt.Errorf("flattening `identity`: %+v", err)
		}
		if err := d.Set("identity", flattenedIdentity); err != nil {
			return fmt.Errorf("setting `identity`: %+v", err)
		}

		if err := tags.FlattenAndSet(d, model.Tags); err != nil {
			return err
		}
	}

	endpoints := flattenAccountEndpoints(primaryEndpoints, secondaryEndpoints, routingPreference)
	if err := endpoints.set(d); err != nil {
		return err
	}

	storageAccountKeys := make([]storageaccounts.StorageAccountKey, 0)
	if keys.Model != nil && keys.Model.Keys != nil {
		storageAccountKeys = *keys.Model.Keys
	}
	keysAndConnectionStrings := flattenAccountAccessKeysAndConnectionStrings(id.StorageAccountName, *storageDomainSuffix, storageAccountKeys, endpoints)
	if err := keysAndConnectionStrings.set(d); err != nil {
		return err
	}

	if dataPlaneOnReadEnabled {
		blobProperties := make([]interface{}, 0)
		queueProperties := make([]interface{}, 0)
		shareProperties := make([]interface{}, 0)
		staticWebsiteProperties := make([]interface{}, 0)

		if supportLevel.supportBlob {
			blobProps, err := storageClient.ResourceManager.BlobService.GetServiceProperties(ctx, *id)
			if err != nil {
				return fmt.Errorf("reading blob properties for %s: %+v", *id, err)
			}

			blobProperties = flattenAccountBlobServiceProperties(blobProps.Model)
		}

		if err := d.Set("blob_properties", blobProperties); err != nil {
			return fmt.Errorf("setting `blob_properties` for %s: %+v", *id, err)
		}

		if supportLevel.supportShare {
			shareProps, err := storageClient.ResourceManager.FileService.GetServiceProperties(ctx, *id)
			if err != nil {
				return fmt.Errorf("retrieving share properties for %s: %+v", *id, err)
			}

			shareProperties = flattenAccountShareProperties(shareProps.Model)
		}

		if err := d.Set("share_properties", shareProperties); err != nil {
			return fmt.Errorf("setting `share_properties` for %s: %+v", *id, err)
		}

		if supportLevel.supportQueue {
			queueClient, err := storageClient.QueuesDataPlaneClient(ctx, *account, storageClient.DataPlaneOperationSupportingAnyAuthMethod())
			if err != nil {
				return fmt.Errorf("building Queues Client: %s", err)
			}

			queueProps, err := queueClient.GetServiceProperties(ctx)
			if err != nil {
				return fmt.Errorf("retrieving queue properties for %s: %+v", *id, err)
			}

			queueProperties = flattenAccountQueueProperties(queueProps)
		}

		if err := d.Set("queue_properties", queueProperties); err != nil {
			return fmt.Errorf("setting `queue_properties`: %+v", err)
		}

		if supportLevel.supportStaticWebsite {
			accountsClient, err := storageClient.AccountsDataPlaneClient(ctx, *account, storageClient.DataPlaneOperationSupportingAnyAuthMethod())
			if err != nil {
				return fmt.Errorf("building Accounts Data Plane Client: %s", err)
			}

			staticWebsiteProps, err := accountsClient.GetServiceProperties(ctx, id.StorageAccountName)
			if err != nil {
				return fmt.Errorf("retrieving static website properties for %s: %+v", *id, err)
			}

			staticWebsiteProperties = flattenAccountStaticWebsiteProperties(staticWebsiteProps)
		}

		if err := d.Set("static_website", staticWebsiteProperties); err != nil {
			return fmt.Errorf("setting `static_website`: %+v", err)
		}
	} else {
		log.Printf("[DEBUG] [%s:READ] Setting 'blob_properties', 'queue_properties', 'share_properties' and 'static_website' skipped due to 'DataPlaneAccessOnReadEnabled' feature flag being set to 'false'", strings.ToUpper(storageAccountResourceName))
		d.Set("blob_properties", d.Get("blob_properties"))
		d.Set("queue_properties", d.Get("queue_properties"))
		d.Set("share_properties", d.Get("share_properties"))
		d.Set("static_website", d.Get("static_website"))
	}

	return nil
}

func resourceStorageAccountDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	storageClient := meta.(*clients.Client).Storage
	client := storageClient.ResourceManager.StorageAccounts
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := commonids.ParseStorageAccountID(d.Id())
	if err != nil {
		return err
	}

	locks.ByName(id.StorageAccountName, storageAccountResourceName)
	defer locks.UnlockByName(id.StorageAccountName, storageAccountResourceName)

	existing, err := client.GetProperties(ctx, *id, storageaccounts.DefaultGetPropertiesOperationOptions())
	if err != nil {
		if response.WasNotFound(existing.HttpResponse) {
			return nil
		}
		return fmt.Errorf("retrieving %s: %+v", *id, err)
	}

	// the networking api's only allow a single change to be made to a network layout at once, so let's lock to handle that
	virtualNetworkNames := make([]string, 0)
	if model := existing.Model; model != nil && model.Properties != nil {
		if acls := model.Properties.NetworkAcls; acls != nil {
			if vnr := acls.VirtualNetworkRules; vnr != nil {
				for _, v := range *vnr {
					subnetId, err := commonids.ParseSubnetIDInsensitively(v.Id)
					if err != nil {
						return err
					}

					networkName := subnetId.VirtualNetworkName
					for _, virtualNetworkName := range virtualNetworkNames {
						if networkName == virtualNetworkName {
							continue
						}
					}
					virtualNetworkNames = append(virtualNetworkNames, networkName)
				}
			}
		}
	}

	locks.MultipleByName(&virtualNetworkNames, network.VirtualNetworkResourceName)
	defer locks.UnlockMultipleByName(&virtualNetworkNames, network.VirtualNetworkResourceName)

	if _, err := client.Delete(ctx, *id); err != nil {
		return fmt.Errorf("deleting %s: %+v", *id, err)
	}

	// remove this from the cache
	storageClient.RemoveAccountFromCache(*id)

	return nil
}

func expandAccountCustomDomain(input []interface{}) *storageaccounts.CustomDomain {
	if len(input) == 0 {
		return &storageaccounts.CustomDomain{
			Name: "",
		}
	}

	domain := input[0].(map[string]interface{})
	return &storageaccounts.CustomDomain{
		Name:             domain["name"].(string),
		UseSubDomainName: pointer.To(domain["use_subdomain"].(bool)),
	}
}

func flattenAccountCustomDomain(input *storageaccounts.CustomDomain) []interface{} {
	output := make([]interface{}, 0)
	if input != nil {
		output = append(output, map[string]interface{}{
			// use_subdomain isn't returned
			"name": input.Name,
		})
	}
	return output
}

func expandAccountCustomerManagedKey(ctx context.Context, keyVaultClient *keyVaultClient.Client, subscriptionId string, input []interface{}, accountTier storageaccounts.SkuTier, accountKind storageaccounts.Kind, expandedIdentity identity.LegacySystemAndUserAssignedMap, queueEncryptionKeyType, tableEncryptionKeyType storageaccounts.KeyType) (*storageaccounts.Encryption, error) {
	if accountKind == storageaccounts.KindStorage {
		if queueEncryptionKeyType == storageaccounts.KeyTypeAccount {
			return nil, fmt.Errorf("`queue_encryption_key_type = %q` cannot be used with account kind `%q`", string(storageaccounts.KeyTypeAccount), string(storageaccounts.KindStorage))
		}
		if tableEncryptionKeyType == storageaccounts.KeyTypeAccount {
			return nil, fmt.Errorf("`table_encryption_key_type = %q` cannot be used with account kind `%q`", string(storageaccounts.KeyTypeAccount), string(storageaccounts.KindStorage))
		}
	}
	if len(input) == 0 {
		return &storageaccounts.Encryption{
			KeySource: pointer.To(storageaccounts.KeySourceMicrosoftPointStorage),
			Services: &storageaccounts.EncryptionServices{
				Queue: &storageaccounts.EncryptionService{
					KeyType: pointer.To(queueEncryptionKeyType),
				},
				Table: &storageaccounts.EncryptionService{
					KeyType: pointer.To(tableEncryptionKeyType),
				},
			},
		}, nil
	}

	if accountTier != storageaccounts.SkuTierPremium && accountKind != storageaccounts.KindStorageVTwo {
		return nil, fmt.Errorf("customer managed key can only be used with account kind `StorageV2` or account tier `Premium`")
	}

	if expandedIdentity.Type != identity.TypeUserAssigned && expandedIdentity.Type != identity.TypeSystemAssignedUserAssigned {
		return nil, fmt.Errorf("customer managed key can only be configured when the storage account uses a `UserAssigned` or `SystemAssigned, UserAssigned` managed identity but got %q", string(expandedIdentity.Type))
	}

	v := input[0].(map[string]interface{})

	var keyName, keyVersion, keyVaultURI *string
	if keyVaultKeyId, ok := v["key_vault_key_id"]; ok && keyVaultKeyId != "" {
		keyId, err := keyVaultParse.ParseOptionallyVersionedNestedItemID(keyVaultKeyId.(string))
		if err != nil {
			return nil, err
		}

		subscriptionResourceId := commonids.NewSubscriptionID(subscriptionId)
		keyVaultIdRaw, err := keyVaultClient.KeyVaultIDFromBaseUrl(ctx, subscriptionResourceId, keyId.KeyVaultBaseUrl)
		if err != nil {
			return nil, err
		}
		if keyVaultIdRaw == nil {
			return nil, fmt.Errorf("unable to find the Resource Manager ID for the Key Vault URI %q in %s", keyId.KeyVaultBaseUrl, subscriptionResourceId)
		}
		keyVaultId, err := commonids.ParseKeyVaultID(*keyVaultIdRaw)
		if err != nil {
			return nil, err
		}

		vaultsClient := keyVaultClient.VaultsClient
		keyVault, err := vaultsClient.Get(ctx, *keyVaultId)
		if err != nil {
			return nil, fmt.Errorf("retrieving %s: %+v", *keyVaultId, err)
		}

		softDeleteEnabled := false
		purgeProtectionEnabled := false
		if model := keyVault.Model; model != nil {
			if esd := model.Properties.EnableSoftDelete; esd != nil {
				softDeleteEnabled = *esd
			}
			if epp := model.Properties.EnablePurgeProtection; epp != nil {
				purgeProtectionEnabled = *epp
			}
		}
		if !softDeleteEnabled || !purgeProtectionEnabled {
			return nil, fmt.Errorf("%s must be configured for both Purge Protection and Soft Delete", *keyVaultId)
		}

		keyName = pointer.To(keyId.Name)
		keyVersion = pointer.To(keyId.Version)
		keyVaultURI = pointer.To(keyId.KeyVaultBaseUrl)
	} else if managedHSMKeyId, ok := v["managed_hsm_key_id"]; ok && managedHSMKeyId != "" {
		if keyId, err := managedHsmParse.ManagedHSMDataPlaneVersionedKeyID(managedHSMKeyId.(string), nil); err == nil {
			keyName = pointer.To(keyId.KeyName)
			keyVersion = pointer.To(keyId.KeyVersion)
			keyVaultURI = pointer.To(keyId.BaseUri())
		} else if keyId, err := managedHsmParse.ManagedHSMDataPlaneVersionlessKeyID(managedHSMKeyId.(string), nil); err == nil {
			keyName = utils.String(keyId.KeyName)
			keyVersion = utils.String("")
			keyVaultURI = utils.String(keyId.BaseUri())
		} else {
			return nil, fmt.Errorf("parsing %q as HSM key ID", managedHSMKeyId.(string))
		}
	}

	encryption := &storageaccounts.Encryption{
		Services: &storageaccounts.EncryptionServices{
			Blob: &storageaccounts.EncryptionService{
				Enabled: pointer.To(true),
				KeyType: pointer.To(storageaccounts.KeyTypeAccount),
			},
			File: &storageaccounts.EncryptionService{
				Enabled: pointer.To(true),
				KeyType: pointer.To(storageaccounts.KeyTypeAccount),
			},
			Queue: &storageaccounts.EncryptionService{
				KeyType: pointer.To(queueEncryptionKeyType),
			},
			Table: &storageaccounts.EncryptionService{
				KeyType: pointer.To(tableEncryptionKeyType),
			},
		},
		Identity: &storageaccounts.EncryptionIdentity{
			UserAssignedIdentity: utils.String(v["user_assigned_identity_id"].(string)),
		},
		KeySource: pointer.To(storageaccounts.KeySourceMicrosoftPointKeyvault),
		Keyvaultproperties: &storageaccounts.KeyVaultProperties{
			Keyname:     keyName,
			Keyversion:  keyVersion,
			Keyvaulturi: keyVaultURI,
		},
	}

	return encryption, nil
}

func flattenAccountCustomerManagedKey(input *storageaccounts.Encryption, env environments.Environment) []interface{} {
	output := make([]interface{}, 0)

	if input != nil && input.KeySource != nil && *input.KeySource == storageaccounts.KeySourceMicrosoftPointKeyvault {
		userAssignedIdentityId := ""
		if props := input.Identity; props != nil {
			userAssignedIdentityId = pointer.From(props.UserAssignedIdentity)
		}

		customerManagedKey := flattenCustomerManagedKey(input.Keyvaultproperties, env.KeyVault, env.ManagedHSM)
		output = append(output, map[string]interface{}{
			"key_vault_key_id":          customerManagedKey.keyVaultKeyUri,
			"managed_hsm_key_id":        customerManagedKey.managedHsmKeyUri,
			"user_assigned_identity_id": userAssignedIdentityId,
		})
	}

	return output
}

func expandAccountImmutabilityPolicy(input []interface{}) *storageaccounts.ImmutableStorageAccount {
	if len(input) == 0 {
		return &storageaccounts.ImmutableStorageAccount{}
	}

	v := input[0].(map[string]interface{})
	return &storageaccounts.ImmutableStorageAccount{
		Enabled: utils.Bool(true),
		ImmutabilityPolicy: &storageaccounts.AccountImmutabilityPolicyProperties{
			AllowProtectedAppendWrites:            pointer.To(v["allow_protected_append_writes"].(bool)),
			ImmutabilityPeriodSinceCreationInDays: pointer.To(int64(v["period_since_creation_in_days"].(int))),
			State:                                 pointer.To(storageaccounts.AccountImmutabilityPolicyState(v["state"].(string))),
		},
	}
}

func flattenAccountImmutabilityPolicy(input *storageaccounts.ImmutableStorageAccount) []interface{} {
	if input == nil || input.ImmutabilityPolicy == nil {
		return make([]interface{}, 0)
	}

	return []interface{}{
		map[string]interface{}{
			"allow_protected_append_writes": input.ImmutabilityPolicy.AllowProtectedAppendWrites,
			"period_since_creation_in_days": input.ImmutabilityPolicy.ImmutabilityPeriodSinceCreationInDays,
			"state":                         input.ImmutabilityPolicy.State,
		},
	}
}

func expandAccountActiveDirectoryProperties(input []interface{}) *storageaccounts.ActiveDirectoryProperties {
	if len(input) == 0 {
		return nil
	}
	m := input[0].(map[string]interface{})

	output := &storageaccounts.ActiveDirectoryProperties{
		DomainGuid: m["domain_guid"].(string),
		DomainName: m["domain_name"].(string),
	}
	if v := m["storage_sid"]; v != "" {
		output.AzureStorageSid = utils.String(v.(string))
	}
	if v := m["domain_sid"]; v != "" {
		output.DomainSid = utils.String(v.(string))
	}
	if v := m["forest_name"]; v != "" {
		output.ForestName = utils.String(v.(string))
	}
	if v := m["netbios_domain_name"]; v != "" {
		output.NetBiosDomainName = utils.String(v.(string))
	}
	return output
}

func flattenAccountActiveDirectoryProperties(input *storageaccounts.ActiveDirectoryProperties) []interface{} {
	output := make([]interface{}, 0)
	if input != nil {
		output = append(output, map[string]interface{}{
			"domain_guid":         input.DomainGuid,
			"domain_name":         input.DomainName,
			"domain_sid":          pointer.From(input.DomainSid),
			"forest_name":         pointer.From(input.ForestName),
			"netbios_domain_name": pointer.From(input.NetBiosDomainName),
			"storage_sid":         pointer.From(input.AzureStorageSid),
		})
	}
	return output
}

func expandAccountAzureFilesAuthentication(input []interface{}) (*storageaccounts.AzureFilesIdentityBasedAuthentication, error) {
	if len(input) == 0 {
		return &storageaccounts.AzureFilesIdentityBasedAuthentication{
			DirectoryServiceOptions: storageaccounts.DirectoryServiceOptionsNone,
		}, nil
	}

	v := input[0].(map[string]interface{})
	output := storageaccounts.AzureFilesIdentityBasedAuthentication{
		DirectoryServiceOptions: storageaccounts.DirectoryServiceOptions(v["directory_type"].(string)),
	}
	if output.DirectoryServiceOptions == storageaccounts.DirectoryServiceOptionsAD ||
		output.DirectoryServiceOptions == storageaccounts.DirectoryServiceOptionsAADDS ||
		output.DirectoryServiceOptions == storageaccounts.DirectoryServiceOptionsAADKERB {
		ad := expandAccountActiveDirectoryProperties(v["active_directory"].([]interface{}))

		if output.DirectoryServiceOptions == storageaccounts.DirectoryServiceOptionsAD {
			if ad == nil {
				return nil, fmt.Errorf("`active_directory` is required when `directory_type` is `AD`")
			}
			if ad.AzureStorageSid == nil {
				return nil, fmt.Errorf("`active_directory.0.storage_sid` is required when `directory_type` is `AD`")
			}
			if ad.DomainSid == nil {
				return nil, fmt.Errorf("`active_directory.0.domain_sid` is required when `directory_type` is `AD`")
			}
			if ad.ForestName == nil {
				return nil, fmt.Errorf("`active_directory.0.forest_name` is required when `directory_type` is `AD`")
			}
			if ad.NetBiosDomainName == nil {
				return nil, fmt.Errorf("`active_directory.0.netbios_domain_name` is required when `directory_type` is `AD`")
			}
		}

		output.ActiveDirectoryProperties = ad
		output.DefaultSharePermission = pointer.To(storageaccounts.DefaultSharePermission(v["default_share_level_permission"].(string)))
	}

	return &output, nil
}

func flattenAccountAzureFilesAuthentication(input *storageaccounts.AzureFilesIdentityBasedAuthentication) []interface{} {
	if input == nil || input.DirectoryServiceOptions == storageaccounts.DirectoryServiceOptionsNone {
		return make([]interface{}, 0)
	}

	return []interface{}{
		map[string]interface{}{
			"active_directory":               flattenAccountActiveDirectoryProperties(input.ActiveDirectoryProperties),
			"directory_type":                 input.DirectoryServiceOptions,
			"default_share_level_permission": input.DefaultSharePermission,
		},
	}
}

func expandAccountRoutingPreference(input []interface{}) *storageaccounts.RoutingPreference {
	if len(input) == 0 {
		return nil
	}
	v := input[0].(map[string]interface{})
	return &storageaccounts.RoutingPreference{
		PublishMicrosoftEndpoints: pointer.To(v["publish_microsoft_endpoints"].(bool)),
		PublishInternetEndpoints:  pointer.To(v["publish_internet_endpoints"].(bool)),
		RoutingChoice:             pointer.To(storageaccounts.RoutingChoice(v["choice"].(string))),
	}
}

func flattenAccountRoutingPreference(input *storageaccounts.RoutingPreference) []interface{} {
	output := make([]interface{}, 0)

	if input != nil {
		routingChoice := ""
		if input.RoutingChoice != nil {
			routingChoice = string(*input.RoutingChoice)
		}

		output = append(output, map[string]interface{}{
			"choice":                      routingChoice,
			"publish_internet_endpoints":  pointer.From(input.PublishInternetEndpoints),
			"publish_microsoft_endpoints": pointer.From(input.PublishMicrosoftEndpoints),
		})
	}

	return output
}

func expandAccountSASPolicy(input []interface{}) *storageaccounts.SasPolicy {
	if len(input) == 0 {
		return nil
	}

	raw := input[0].(map[string]interface{})
	return &storageaccounts.SasPolicy{
		ExpirationAction:    storageaccounts.ExpirationAction(raw["expiration_action"].(string)),
		SasExpirationPeriod: raw["expiration_period"].(string),
	}
}

func flattenAccountSASPolicy(input *storageaccounts.SasPolicy) []interface{} {
	output := make([]interface{}, 0)

	if input != nil {
		output = append(output, map[string]interface{}{
			"expiration_action": string(input.ExpirationAction),
			"expiration_period": input.SasExpirationPeriod,
		})
	}

	return output
}

func expandEdgeZone(input string) *edgezones.Model {
	normalized := edgezones.Normalize(input)
	if normalized == "" {
		return nil
	}

	return &edgezones.Model{
		Name: normalized,
	}
}

func flattenEdgeZone(input *edgezones.Model) string {
	output := ""
	if input != nil {
		output = edgezones.Normalize(input.Name)
	}
	return output
}
