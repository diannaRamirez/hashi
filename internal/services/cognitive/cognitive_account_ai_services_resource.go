// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cognitive

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/identity"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/location"
	"github.com/hashicorp/go-azure-sdk/resource-manager/cognitive/2023-05-01/cognitiveservicesaccounts"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	commonValidate "github.com/hashicorp/terraform-provider-azurerm/helpers/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/locks"
	"github.com/hashicorp/terraform-provider-azurerm/internal/sdk"
	cognitiveValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/cognitive/validate"
	keyVaultParse "github.com/hashicorp/terraform-provider-azurerm/internal/services/keyvault/parse"
	keyVaultValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/keyvault/validate"
	managedHsmHelpers "github.com/hashicorp/terraform-provider-azurerm/internal/services/managedhsm/helpers"
	managedHsmParse "github.com/hashicorp/terraform-provider-azurerm/internal/services/managedhsm/parse"
	managedHsmValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/managedhsm/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/network"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/set"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

var _ sdk.ResourceWithUpdate = AIServicesAccountResource{}

var _ sdk.ResourceWithCustomImporter = AIServicesAccountResource{}

type AIServicesAccountResource struct{}

func (r AIServicesAccountResource) CustomImporter() sdk.ResourceRunFunc {
	return func(ctx context.Context, metadata sdk.ResourceMetaData) error {
		_, err := cognitiveservicesaccounts.ParseAccountID(metadata.ResourceData.Id())
		if err != nil {
			return err
		}
		return nil
	}
}

type AIServicesAccountVirtualNetworkRules struct {
	SubnetID                         string `tfschema:"subnet_id"`
	IgnoreMissingVnetServiceEndpoint bool   `tfschema:"ignore_missing_vnet_service_endpoint"`
}

type AIServicesAccountNetworkACLs struct {
	DefaultAction       string                                 `tfschema:"default_action"`
	IpRules             []string                               `tfschema:"ip_rules"`
	VirtualNetworkRules []AIServicesAccountVirtualNetworkRules `tfschema:"virtual_network_rules"`
}

type AIServicesAccountCustomerManagedKey struct {
	IdentityClientID string `tfschema:"identity_client_id"`
	KeyVaultKeyID    string `tfschema:"key_vault_key_id"`
	ManagedHsmKeyID  string `tfschema:"managed_hsm_key_id"`
}

type AIServicesAccountResourceResourceModel struct {
	Name                            string                                     `tfschema:"name"`
	ResourceGroupName               string                                     `tfschema:"resource_group_name"`
	Location                        string                                     `tfschema:"location"`
	SkuName                         string                                     `tfschema:"sku_name"`
	CustomSubdomainName             string                                     `tfschema:"custom_subdomain_name"`
	CustomerManagedKey              []AIServicesAccountCustomerManagedKey      `tfschema:"customer_managed_key"`
	Fqdns                           []string                                   `tfschema:"fqdns"`
	Identity                        []identity.ModelSystemAssignedUserAssigned `tfschema:"identity"`
	LocalAuthorizationEnabled       bool                                       `tfschema:"local_authentication_enabled"`
	NetworkACLs                     []AIServicesAccountNetworkACLs             `tfschema:"network_acls"`
	OutboundNetworkAccessRestricted bool                                       `tfschema:"outbound_network_access_restricted"`
	PublicNetworkAccess             string                                     `tfschema:"public_network_access"`
	Tags                            map[string]string                          `tfschema:"tags"`
	Endpoint                        string                                     `tfschema:"endpoint"`
	PrimaryAccessKey                string                                     `tfschema:"primary_access_key"`
	SecondaryAccessKey              string                                     `tfschema:"secondary_access_key"`
}

func (AIServicesAccountResource) Arguments() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{

		"name": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: cognitiveValidate.AccountName(),
		},

		"location": commonschema.Location(),

		"resource_group_name": commonschema.ResourceGroupName(),

		"sku_name": {
			Type:     pluginsdk.TypeString,
			Required: true,
			ValidateFunc: validation.StringInSlice([]string{
				"F0", "F1", "S0", "S", "S1", "S2", "S3", "S4", "S5", "S6", "P0", "P1", "P2", "E0", "DC0",
			}, false),
		},

		"custom_subdomain_name": {
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ForceNew:     true,
			ValidateFunc: validation.StringIsNotEmpty,
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

					"identity_client_id": {
						Type:         pluginsdk.TypeString,
						Optional:     true,
						ValidateFunc: validation.IsUUID,
					},
				},
			},
		},

		"fqdns": {
			Type:     pluginsdk.TypeList,
			Optional: true,
			Elem: &pluginsdk.Schema{
				Type:         pluginsdk.TypeString,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},

		"identity": commonschema.SystemAssignedUserAssignedIdentityOptional(),

		"local_authentication_enabled": {
			Type:     pluginsdk.TypeBool,
			Optional: true,
			Default:  true,
		},

		"network_acls": {
			Type:         pluginsdk.TypeList,
			Optional:     true,
			MaxItems:     1,
			RequiredWith: []string{"custom_subdomain_name"},
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"default_action": {
						Type:     pluginsdk.TypeString,
						Required: true,
						ValidateFunc: validation.StringInSlice([]string{
							string(cognitiveservicesaccounts.NetworkRuleActionAllow),
							string(cognitiveservicesaccounts.NetworkRuleActionDeny),
						}, false),
					},
					"ip_rules": {
						Type:     pluginsdk.TypeSet,
						Optional: true,
						Set:      set.HashIPv4AddressOrCIDR,
						Elem: &pluginsdk.Schema{
							Type: pluginsdk.TypeString,
							ValidateFunc: validation.Any(
								commonValidate.IPv4Address,
								commonValidate.CIDR,
							),
						},
					},

					"virtual_network_rules": {
						Type:     pluginsdk.TypeSet,
						Optional: true,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"subnet_id": {
									Type:     pluginsdk.TypeString,
									Required: true,
								},

								"ignore_missing_vnet_service_endpoint": {
									Type:     pluginsdk.TypeBool,
									Optional: true,
									Default:  false,
								},
							},
						},
					},
				},
			},
		},

		"outbound_network_access_restricted": {
			Type:     pluginsdk.TypeBool,
			Optional: true,
			Default:  false,
		},

		"public_network_access": {
			Type:     pluginsdk.TypeString,
			Optional: true,
			ValidateFunc: validation.StringInSlice([]string{
				string(cognitiveservicesaccounts.PublicNetworkAccessEnabled),
				string(cognitiveservicesaccounts.PublicNetworkAccessDisabled),
			}, false),
			Default: string(cognitiveservicesaccounts.PublicNetworkAccessEnabled),
		},

		"storage": {
			Type:     pluginsdk.TypeList,
			Optional: true,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"storage_account_id": {
						Type:         pluginsdk.TypeString,
						Required:     true,
						ValidateFunc: commonids.ValidateStorageAccountID,
					},

					"identity_client_id": {
						Type:         pluginsdk.TypeString,
						Optional:     true,
						ValidateFunc: validation.IsUUID,
					},
				},
			},
		},

		"tags": commonschema.Tags(),
	}
}

func (AIServicesAccountResource) Attributes() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"endpoint": {
			Type:     pluginsdk.TypeString,
			Computed: true,
		},

		"primary_access_key": {
			Type:      pluginsdk.TypeString,
			Computed:  true,
			Sensitive: true,
		},

		"secondary_access_key": {
			Type:      pluginsdk.TypeString,
			Computed:  true,
			Sensitive: true,
		},
	}
}

func (AIServicesAccountResource) ModelObject() interface{} {
	return &AIServicesAccountResourceResourceModel{}
}

func (AIServicesAccountResource) ResourceType() string {
	return "azurerm_cognitive_account_ai_services"
}

func (AIServicesAccountResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 60 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			var model AIServicesAccountResourceResourceModel
			if err := metadata.Decode(&model); err != nil {
				return err
			}

			client := metadata.Client.Cognitive.AccountsClient
			subscriptionId := metadata.Client.Account.SubscriptionId

			id := cognitiveservicesaccounts.NewAccountID(subscriptionId, model.ResourceGroupName, model.Name)
			existing, err := client.AccountsGet(ctx, id)
			if err != nil {
				if !response.WasNotFound(existing.HttpResponse) {
					return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
				}
			}

			if !response.WasNotFound(existing.HttpResponse) {
				return tf.ImportAsExistsError("azurerm_cognitive_account_ai_services", id.ID())
			}

			networkACLs, subnetIds := expandAIServicesAccountNetworkACLs(model.NetworkACLs)

			// also lock on the Virtual Network ID's since modifications in the networking stack are exclusive
			virtualNetworkNames := make([]string, 0)
			for _, v := range subnetIds {
				subnetId, err := commonids.ParseSubnetID(v)
				if err != nil {
					return err
				}
				if !utils.SliceContainsValue(virtualNetworkNames, subnetId.VirtualNetworkName) {
					virtualNetworkNames = append(virtualNetworkNames, subnetId.VirtualNetworkName)
				}
			}

			locks.MultipleByName(&virtualNetworkNames, network.VirtualNetworkResourceName)
			defer locks.UnlockMultipleByName(&virtualNetworkNames, network.VirtualNetworkResourceName)

			props := cognitiveservicesaccounts.Account{
				Kind:     utils.String("AIServices"),
				Location: utils.String(azure.NormalizeLocation(model.Location)),
				Sku: &cognitiveservicesaccounts.Sku{
					Name: model.SkuName,
				},
				Properties: &cognitiveservicesaccounts.AccountProperties{
					NetworkAcls:                   networkACLs,
					CustomSubDomainName:           pointer.FromString(model.CustomSubdomainName),
					AllowedFqdnList:               pointer.To(model.Fqdns),
					PublicNetworkAccess:           pointer.To(cognitiveservicesaccounts.PublicNetworkAccess(model.PublicNetworkAccess)),
					RestrictOutboundNetworkAccess: pointer.To(model.OutboundNetworkAccessRestricted),
					DisableLocalAuth:              pointer.To(!model.LocalAuthorizationEnabled),
				},
				Tags: pointer.To(model.Tags),
			}

			customMangedKey, err := expandAIServicesAccountCustomerManagedKey(model.CustomerManagedKey)
			if err != nil {
				return fmt.Errorf("expanding `customer_managed_key`: %+v", err)
			}
			props.Properties.Encryption = customMangedKey

			expandIdentity, err := identity.ExpandSystemAndUserAssignedMapFromModel(model.Identity)
			if err != nil {
				return fmt.Errorf("expanding `identity`: %+v", err)
			}
			props.Identity = expandIdentity

			future, err := client.AccountsCreate(ctx, id, props)
			if err != nil {
				return fmt.Errorf("creating %s: %+v", id, err)
			}

			if err := future.Poller.PollUntilDone(ctx); err != nil {
				return fmt.Errorf("waiting for creating of %s: %+v", id, err)
			}

			metadata.SetID(id)

			return nil
		},
	}
}

func (AIServicesAccountResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 5 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.Cognitive.AccountsClient
			env := metadata.Client.Account.Environment

			state := AIServicesAccountResourceResourceModel{}
			id, err := cognitiveservicesaccounts.ParseAccountID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			resp, err := client.AccountsGet(ctx, *id)
			if err != nil {
				if response.WasNotFound(resp.HttpResponse) {
					return metadata.MarkAsGone(id)
				}
				return fmt.Errorf("retrieving %s: %+v", *id, err)
			}

			keys, err := client.AccountsListKeys(ctx, *id)
			if err != nil {
				// note for the resource we shouldn't gracefully fail since we have permission to CRUD it
				return fmt.Errorf("listing the Keys for %s: %+v", id, err)
			}

			if model := keys.Model; model != nil {
				state.PrimaryAccessKey = pointer.From(model.Key1)
				state.SecondaryAccessKey = pointer.From(model.Key2)
			}

			state.Name = id.AccountName
			state.ResourceGroupName = id.ResourceGroupName

			if model := resp.Model; model != nil {
				state.Location = location.NormalizeNilable(model.Location)
				if sku := model.Sku; sku != nil {
					state.SkuName = sku.Name
				}

				identityFlatten, err := identity.FlattenSystemAndUserAssignedMapToModel(model.Identity)
				if err != nil {
					return err
				}
				state.Identity = *identityFlatten

				if props := model.Properties; props != nil {
					state.Endpoint = pointer.From(props.Endpoint)
					state.CustomSubdomainName = pointer.From(props.CustomSubDomainName)
					state.NetworkACLs = flattenAIServicesAccountNetworkACLs(props.NetworkAcls)
					state.Fqdns = pointer.From(props.AllowedFqdnList)

					state.PublicNetworkAccess = string(*props.PublicNetworkAccess)

					outboundNetworkAccessRestricted := false
					if props.RestrictOutboundNetworkAccess != nil {
						outboundNetworkAccessRestricted = *props.RestrictOutboundNetworkAccess
					}
					state.OutboundNetworkAccessRestricted = outboundNetworkAccessRestricted

					localAuthEnabled := true
					if props.DisableLocalAuth != nil {
						localAuthEnabled = !*props.DisableLocalAuth
					}
					state.LocalAuthorizationEnabled = localAuthEnabled

					customerManagedKey, err := flattenAIServicesAccountCustomerManagedKey(props.Encryption, env)
					if err != nil {
						return err
					}
					state.CustomerManagedKey = customerManagedKey
				}

				state.Tags = pointer.From(model.Tags)
			}

			return metadata.Encode(&state)
		},
	}
}

func (AIServicesAccountResource) Update() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 60 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.Cognitive.AccountsClient

			var model AIServicesAccountResourceResourceModel

			if err := metadata.Decode(&model); err != nil {
				return err
			}

			id, err := cognitiveservicesaccounts.ParseAccountID(metadata.ResourceData.Id())
			if err != nil {
				return fmt.Errorf(" Cannot parse Cognitive Account AI service ID: %s", err)
			}

			networkACLs, subnetIds := expandAIServicesAccountNetworkACLs(model.NetworkACLs)
			locks.MultipleByName(&subnetIds, network.VirtualNetworkResourceName)
			defer locks.UnlockMultipleByName(&subnetIds, network.VirtualNetworkResourceName)

			// also lock on the Virtual Network ID's since modifications in the networking stack are exclusive
			virtualNetworkNames := make([]string, 0)
			for _, v := range subnetIds {
				subnetId, err := commonids.ParseSubnetIDInsensitively(v)
				if err != nil {
					return err
				}
				if !utils.SliceContainsValue(virtualNetworkNames, subnetId.VirtualNetworkName) {
					virtualNetworkNames = append(virtualNetworkNames, subnetId.VirtualNetworkName)
				}
			}

			locks.MultipleByName(&virtualNetworkNames, network.VirtualNetworkResourceName)
			defer locks.UnlockMultipleByName(&virtualNetworkNames, network.VirtualNetworkResourceName)

			props := cognitiveservicesaccounts.Account{
				Sku: &cognitiveservicesaccounts.Sku{
					Name: model.SkuName,
				},
				Properties: &cognitiveservicesaccounts.AccountProperties{
					NetworkAcls:                   networkACLs,
					CustomSubDomainName:           pointer.FromString(model.CustomSubdomainName),
					AllowedFqdnList:               pointer.To(model.Fqdns),
					PublicNetworkAccess:           pointer.To(cognitiveservicesaccounts.PublicNetworkAccess(model.PublicNetworkAccess)),
					RestrictOutboundNetworkAccess: pointer.To(model.OutboundNetworkAccessRestricted),
					DisableLocalAuth:              pointer.To(!model.LocalAuthorizationEnabled),
					Encryption: &cognitiveservicesaccounts.Encryption{
						KeySource: pointer.To(cognitiveservicesaccounts.KeySourceMicrosoftPointCognitiveServices),
					},
				},
				Tags: pointer.To(model.Tags),
			}

			customMangedKey, err := expandAIServicesAccountCustomerManagedKey(model.CustomerManagedKey)
			if err != nil {
				return fmt.Errorf("expanding `customer_managed_key`: %+v", err)
			}
			if customMangedKey != nil {
				props.Properties.Encryption = customMangedKey
			}

			expandIdentity, err := identity.ExpandSystemAndUserAssignedMapFromModel(model.Identity)
			if err != nil {
				return fmt.Errorf("expanding `identity`: %+v", err)
			}
			props.Identity = expandIdentity

			future, err := client.AccountsUpdate(ctx, *id, props)
			if err != nil {
				return fmt.Errorf("updating %s: %+v", id, err)
			}

			if err := future.Poller.PollUntilDone(ctx); err != nil {
				return fmt.Errorf("waiting for updating of %s: %+v", id, err)
			}

			return nil
		},
	}
}

func (AIServicesAccountResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.Cognitive.AccountsClient

			id, err := cognitiveservicesaccounts.ParseAccountID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			log.Printf("[DEBUG] Retrieving %s..", *id)
			account, err := client.AccountsGet(ctx, *id)
			if err != nil || account.Model == nil || account.Model.Location == nil {
				return fmt.Errorf("retrieving %s: %+v", *id, err)
			}

			deletedAccountId := cognitiveservicesaccounts.NewDeletedAccountID(id.SubscriptionId, *account.Model.Location, id.ResourceGroupName, id.AccountName)
			if err != nil {
				return err
			}

			log.Printf("[DEBUG] Deleting %s..", *id)
			if err := client.AccountsDeleteThenPoll(ctx, *id); err != nil {
				return fmt.Errorf("deleting %s: %+v", *id, err)
			}
			if metadata.Client.Features.CognitiveAccount.PurgeSoftDeleteOnDestroy {
				log.Printf("[DEBUG] Purging %s..", *id)
				if err := client.DeletedAccountsPurgeThenPoll(ctx, deletedAccountId); err != nil {
					return fmt.Errorf("purging %s: %+v", *id, err)
				}
			} else {
				log.Printf("[DEBUG] Skipping Purge of %s", *id)
			}

			return nil
		},
	}
}

func (AIServicesAccountResource) IDValidationFunc() pluginsdk.SchemaValidateFunc {
	return cognitiveservicesaccounts.ValidateAccountID
}

func expandAIServicesAccountCustomerManagedKey(input []AIServicesAccountCustomerManagedKey) (*cognitiveservicesaccounts.Encryption, error) {
	if len(input) == 0 {
		return nil, nil
	}

	v := input[0]
	keySource := cognitiveservicesaccounts.KeySourceMicrosoftPointKeyVault

	var identityClientId string
	if value := v.IdentityClientID; value != "" {
		identityClientId = value
	}

	encryption := &cognitiveservicesaccounts.Encryption{
		KeySource: &keySource,
		KeyVaultProperties: &cognitiveservicesaccounts.KeyVaultProperties{
			IdentityClientId: utils.String(identityClientId),
		},
	}

	if v.KeyVaultKeyID != "" {
		keyId, err := keyVaultParse.ParseOptionallyVersionedNestedItemID(v.KeyVaultKeyID)
		if err != nil {
			return nil, fmt.Errorf(" Failed to parse '%s' as Key Vault key ID", keySource)
		}
		encryption.KeyVaultProperties.KeyName = utils.String(keyId.Name)
		encryption.KeyVaultProperties.KeyVersion = utils.String(keyId.Version)
		encryption.KeyVaultProperties.KeyVaultUri = utils.String(keyId.KeyVaultBaseUrl)
	} else {
		hsmKyId, err := managedHsmParse.ManagedHSMDataPlaneVersionedKeyID(v.ManagedHsmKeyID, nil)
		if err != nil {
			return nil, fmt.Errorf(" Failed to parse '%s' as Key Vault Managed HSM key ID", hsmKyId)
		}

		encryption.KeyVaultProperties.KeyName = utils.String(hsmKyId.KeyName)
		encryption.KeyVaultProperties.KeyVersion = utils.String(hsmKyId.KeyVersion)
		encryption.KeyVaultProperties.KeyVaultUri = utils.String(hsmKyId.BaseUri())
	}
	return encryption, nil

}

func flattenAIServicesAccountCustomerManagedKey(input *cognitiveservicesaccounts.Encryption, env environments.Environment) ([]AIServicesAccountCustomerManagedKey, error) {
	if input == nil || *input.KeySource == cognitiveservicesaccounts.KeySourceMicrosoftPointCognitiveServices {
		return []AIServicesAccountCustomerManagedKey{}, nil
	}

	keyName := ""
	keyVaultURI := ""
	keyVersion := ""
	customerManagerKey := AIServicesAccountCustomerManagedKey{}

	if props := input.KeyVaultProperties; props != nil {
		if props.KeyName != nil {
			keyName = *props.KeyName
		}
		if props.KeyVaultUri != nil {
			keyVaultURI = *props.KeyVaultUri
		}
		if props.KeyVersion != nil {
			keyVersion = *props.KeyVersion
		}

		isHsmURI, err, instanceName, domainSuffix := managedHsmHelpers.IsManagedHSMURI(env, keyVaultURI)
		if err != nil {
			return nil, err
		}

		if props.IdentityClientId != nil {
			customerManagerKey.IdentityClientID = *props.IdentityClientId
		}

		switch {
		case isHsmURI && keyVersion == "":
			{
				keyVaultKeyId := managedHsmParse.NewManagedHSMDataPlaneVersionlessKeyID(instanceName, domainSuffix, keyName)
				customerManagerKey.ManagedHsmKeyID = keyVaultKeyId.ID()
			}
		case isHsmURI && keyVersion != "":
			{
				keyVaultKeyId := managedHsmParse.NewManagedHSMDataPlaneVersionedKeyID(instanceName, domainSuffix, keyName, keyVersion)
				customerManagerKey.ManagedHsmKeyID = keyVaultKeyId.ID()
			}
		case !isHsmURI:
			{
				keyVaultKeyId, err := keyVaultParse.NewNestedItemID(keyVaultURI, keyVaultParse.NestedItemTypeKey, keyName, keyVersion)
				if err != nil {
					return nil, fmt.Errorf("parsing `key_vault_key_id`: %+v", err)
				}
				customerManagerKey.KeyVaultKeyID = keyVaultKeyId.ID()
			}
		}
	}

	return []AIServicesAccountCustomerManagedKey{customerManagerKey}, nil
}

func expandAIServicesAccountNetworkACLs(input []AIServicesAccountNetworkACLs) (*cognitiveservicesaccounts.NetworkRuleSet, []string) {
	subnetIds := make([]string, 0)
	if len(input) == 0 {
		return nil, subnetIds
	}

	v := input[0]

	defaultAction := cognitiveservicesaccounts.NetworkRuleAction(v.DefaultAction)

	ipRules := make([]cognitiveservicesaccounts.IPRule, 0)

	for _, val := range v.IpRules {
		rule := cognitiveservicesaccounts.IPRule{
			Value: val,
		}
		ipRules = append(ipRules, rule)
	}

	networkRules := make([]cognitiveservicesaccounts.VirtualNetworkRule, 0)
	for _, val := range v.VirtualNetworkRules {
		subnetId := val.SubnetID
		subnetIds = append(subnetIds, subnetId)
		rule := cognitiveservicesaccounts.VirtualNetworkRule{
			Id:                               subnetId,
			IgnoreMissingVnetServiceEndpoint: utils.Bool(val.IgnoreMissingVnetServiceEndpoint),
		}
		networkRules = append(networkRules, rule)
	}

	ruleSet := cognitiveservicesaccounts.NetworkRuleSet{
		DefaultAction:       &defaultAction,
		IPRules:             &ipRules,
		VirtualNetworkRules: &networkRules,
	}
	return &ruleSet, subnetIds
}

func flattenAIServicesAccountNetworkACLs(input *cognitiveservicesaccounts.NetworkRuleSet) []AIServicesAccountNetworkACLs {
	if input == nil {
		return []AIServicesAccountNetworkACLs{}
	}

	ipRules := make([]string, 0)
	if input.IPRules != nil {
		for _, v := range *input.IPRules {
			ipRules = append(ipRules, v.Value)
		}
	}

	virtualNetworkRules := make([]AIServicesAccountVirtualNetworkRules, 0)
	if input.VirtualNetworkRules != nil {
		for _, v := range *input.VirtualNetworkRules {
			id := v.Id
			subnetId, err := commonids.ParseSubnetIDInsensitively(v.Id)
			if err == nil {
				id = subnetId.ID()
			}

			virtualNetworkRules = append(virtualNetworkRules, AIServicesAccountVirtualNetworkRules{
				SubnetID:                         id,
				IgnoreMissingVnetServiceEndpoint: pointer.From(v.IgnoreMissingVnetServiceEndpoint),
			})
		}
	}

	return []AIServicesAccountNetworkACLs{{
		DefaultAction:       string(*input.DefaultAction),
		IpRules:             ipRules,
		VirtualNetworkRules: virtualNetworkRules,
	}}
}
