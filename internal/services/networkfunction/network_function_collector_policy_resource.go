package networkfunction

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/location"
	"github.com/hashicorp/go-azure-sdk/resource-manager/networkfunction/2022-11-01/azuretrafficcollectors"
	"github.com/hashicorp/go-azure-sdk/resource-manager/networkfunction/2022-11-01/collectorpolicies"
	"github.com/hashicorp/terraform-provider-azurerm/internal/sdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
)

type NetworkFunctionCollectorPolicyModel struct {
	Name                                   string                                  `tfschema:"name"`
	NetworkFunctionAzureTrafficCollectorId string                                  `tfschema:"network_function_azure_traffic_collector_id"`
	EmissionPolicies                       []EmissionPoliciesPropertiesFormatModel `tfschema:"emission_policy"`
	IngestionPolicy                        []IngestionPolicyPropertiesFormatModel  `tfschema:"ingestion_policy"`
	Location                               string                                  `tfschema:"location"`
	Tags                                   map[string]string                       `tfschema:"tags"`
}

type EmissionPoliciesPropertiesFormatModel struct {
	EmissionDestinations []EmissionPolicyDestinationModel `tfschema:"emission_destination"`
	EmissionType         collectorpolicies.EmissionType   `tfschema:"emission_type"`
}

type EmissionPolicyDestinationModel struct {
	DestinationType collectorpolicies.DestinationType `tfschema:"destination_type"`
}

type IngestionPolicyPropertiesFormatModel struct {
	IngestionSources []IngestionSourcesPropertiesFormatModel `tfschema:"ingestion_source"`
	IngestionType    collectorpolicies.IngestionType         `tfschema:"ingestion_type"`
}

type IngestionSourcesPropertiesFormatModel struct {
	ResourceId string                       `tfschema:"resource_id"`
	SourceType collectorpolicies.SourceType `tfschema:"source_type"`
}

type NetworkFunctionCollectorPolicyResource struct{}

var _ sdk.ResourceWithUpdate = NetworkFunctionCollectorPolicyResource{}

func (r NetworkFunctionCollectorPolicyResource) ResourceType() string {
	return "azurerm_network_function_collector_policy"
}

func (r NetworkFunctionCollectorPolicyResource) ModelObject() interface{} {
	return &NetworkFunctionCollectorPolicyModel{}
}

func (r NetworkFunctionCollectorPolicyResource) IDValidationFunc() pluginsdk.SchemaValidateFunc {
	return collectorpolicies.ValidateCollectorPolicyID
}

func (r NetworkFunctionCollectorPolicyResource) Arguments() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"name": {
			Type:     pluginsdk.TypeString,
			Required: true,
			ForceNew: true,
			ValidateFunc: validation.StringMatch(
				regexp.MustCompile("^[a-zA-Z0-9]([-._a-zA-Z0-9]{0,78}[a-zA-Z0-9_])?$"),
				"The name can contain only letters, numbers, periods (.), hyphens (-),and underscores (_), up to 80 characters, and it must begin with a letter or number and end with a letter, number or underscore.",
			),
		},

		"network_function_azure_traffic_collector_id": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: azuretrafficcollectors.ValidateAzureTrafficCollectorID,
		},

		"emission_policy": {
			Type:     pluginsdk.TypeList,
			Required: true,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"emission_destination": {
						Type:     pluginsdk.TypeList,
						Required: true,
						MinItems: 1,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"destination_type": {
									Type:     pluginsdk.TypeString,
									Required: true,
									ValidateFunc: validation.StringInSlice([]string{
										string(collectorpolicies.DestinationTypeAzureMonitor),
									}, false),
								},
							},
						},
					},

					"emission_type": {
						Type:     pluginsdk.TypeString,
						Optional: true,
						ValidateFunc: validation.StringInSlice([]string{
							string(collectorpolicies.EmissionTypeIPFIX),
						}, false),
						Default: string(collectorpolicies.EmissionTypeIPFIX),
					},
				},
			},
		},

		"ingestion_policy": {
			Type:     pluginsdk.TypeList,
			Required: true,
			MaxItems: 1,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"ingestion_source": {
						Type:     pluginsdk.TypeList,
						Required: true,
						MinItems: 1,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"resource_id": {
									Type:         pluginsdk.TypeString,
									Required:     true,
									ValidateFunc: azure.ValidateResourceID,
								},

								"source_type": {
									Type:     pluginsdk.TypeString,
									Required: true,
									ValidateFunc: validation.StringInSlice([]string{
										string(collectorpolicies.SourceTypeResource),
									}, false),
								},
							},
						},
					},

					"ingestion_type": {
						Type:     pluginsdk.TypeString,
						Optional: true,
						ValidateFunc: validation.StringInSlice([]string{
							string(collectorpolicies.IngestionTypeIPFIX),
						}, false),
						Default: string(collectorpolicies.IngestionTypeIPFIX),
					},
				},
			},
		},

		"location": commonschema.Location(),

		"tags": commonschema.Tags(),
	}
}

func (r NetworkFunctionCollectorPolicyResource) Attributes() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{}
}

func (r NetworkFunctionCollectorPolicyResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			var model NetworkFunctionCollectorPolicyModel
			if err := metadata.Decode(&model); err != nil {
				return fmt.Errorf("decoding: %+v", err)
			}

			client := metadata.Client.NetworkFunction.CollectorPoliciesClient
			azureTrafficCollectorId, err := azuretrafficcollectors.ParseAzureTrafficCollectorID(model.NetworkFunctionAzureTrafficCollectorId)
			if err != nil {
				return err
			}

			id := collectorpolicies.NewCollectorPolicyID(azureTrafficCollectorId.SubscriptionId, azureTrafficCollectorId.ResourceGroupName, azureTrafficCollectorId.AzureTrafficCollectorName, model.Name)
			existing, err := client.Get(ctx, id)
			if err != nil && !response.WasNotFound(existing.HttpResponse) {
				return fmt.Errorf("checking for existing %s: %+v", id, err)
			}

			if !response.WasNotFound(existing.HttpResponse) {
				return metadata.ResourceRequiresImport(r.ResourceType(), id)
			}

			properties := &collectorpolicies.CollectorPolicy{
				Location:   location.Normalize(model.Location),
				Properties: &collectorpolicies.CollectorPolicyPropertiesFormat{},
				Tags:       &model.Tags,
			}

			emissionPoliciesValue, err := expandEmissionPoliciesPropertiesFormatModelArray(model.EmissionPolicies)
			if err != nil {
				return err
			}

			properties.Properties.EmissionPolicies = emissionPoliciesValue

			ingestionPolicyValue, err := expandIngestionPolicyPropertiesFormatModel(model.IngestionPolicy)
			if err != nil {
				return err
			}

			properties.Properties.IngestionPolicy = ingestionPolicyValue

			if err := client.CreateOrUpdateThenPoll(ctx, id, *properties); err != nil {
				return fmt.Errorf("creating %s: %+v", id, err)
			}

			metadata.SetID(id)
			return nil
		},
	}
}

func (r NetworkFunctionCollectorPolicyResource) Update() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.NetworkFunction.CollectorPoliciesClient

			id, err := collectorpolicies.ParseCollectorPolicyID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			var model NetworkFunctionCollectorPolicyModel
			if err := metadata.Decode(&model); err != nil {
				return fmt.Errorf("decoding: %+v", err)
			}

			resp, err := client.Get(ctx, *id)
			if err != nil {
				return fmt.Errorf("retrieving %s: %+v", *id, err)
			}

			properties := resp.Model
			if properties == nil {
				return fmt.Errorf("retrieving %s: properties was nil", id)
			}

			if metadata.ResourceData.HasChange("location") {
				properties.Location = location.Normalize(model.Location)
			}

			if metadata.ResourceData.HasChange("emission_policy") {
				emissionPoliciesValue, err := expandEmissionPoliciesPropertiesFormatModelArray(model.EmissionPolicies)
				if err != nil {
					return err
				}

				properties.Properties.EmissionPolicies = emissionPoliciesValue
			}

			if metadata.ResourceData.HasChange("ingestion_policy") {
				ingestionPolicyValue, err := expandIngestionPolicyPropertiesFormatModel(model.IngestionPolicy)
				if err != nil {
					return err
				}

				properties.Properties.IngestionPolicy = ingestionPolicyValue
			}

			properties.SystemData = nil

			if metadata.ResourceData.HasChange("tags") {
				properties.Tags = &model.Tags
			}

			if err := client.CreateOrUpdateThenPoll(ctx, *id, *properties); err != nil {
				return fmt.Errorf("updating %s: %+v", *id, err)
			}

			return nil
		},
	}
}

func (r NetworkFunctionCollectorPolicyResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 5 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.NetworkFunction.CollectorPoliciesClient

			id, err := collectorpolicies.ParseCollectorPolicyID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			resp, err := client.Get(ctx, *id)
			if err != nil {
				if response.WasNotFound(resp.HttpResponse) {
					return metadata.MarkAsGone(id)
				}

				return fmt.Errorf("retrieving %s: %+v", *id, err)
			}

			model := resp.Model
			if model == nil {
				return fmt.Errorf("retrieving %s: model was nil", id)
			}

			state := NetworkFunctionCollectorPolicyModel{
				Name:                                   id.CollectorPolicyName,
				NetworkFunctionAzureTrafficCollectorId: azuretrafficcollectors.NewAzureTrafficCollectorID(id.SubscriptionId, id.ResourceGroupName, id.AzureTrafficCollectorName).ID(),
				Location:                               location.Normalize(model.Location),
			}

			if properties := model.Properties; properties != nil {
				emissionPoliciesValue, err := flattenEmissionPoliciesPropertiesFormatModelArray(properties.EmissionPolicies)
				if err != nil {
					return err
				}

				state.EmissionPolicies = emissionPoliciesValue

				ingestionPolicyValue, err := flattenIngestionPolicyPropertiesFormatModel(properties.IngestionPolicy)
				if err != nil {
					return err
				}

				state.IngestionPolicy = ingestionPolicyValue
			}

			if model.Tags != nil {
				state.Tags = *model.Tags
			}

			return metadata.Encode(&state)
		},
	}
}

func (r NetworkFunctionCollectorPolicyResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.NetworkFunction.CollectorPoliciesClient

			id, err := collectorpolicies.ParseCollectorPolicyID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			if err := client.DeleteThenPoll(ctx, *id); err != nil {
				return fmt.Errorf("deleting %s: %+v", id, err)
			}

			return nil
		},
	}
}

func expandEmissionPoliciesPropertiesFormatModelArray(inputList []EmissionPoliciesPropertiesFormatModel) (*[]collectorpolicies.EmissionPoliciesPropertiesFormat, error) {
	var outputList []collectorpolicies.EmissionPoliciesPropertiesFormat
	for _, v := range inputList {
		input := v
		output := collectorpolicies.EmissionPoliciesPropertiesFormat{
			EmissionType: &input.EmissionType,
		}

		emissionDestinationsValue, err := expandEmissionPolicyDestinationModelArray(input.EmissionDestinations)
		if err != nil {
			return nil, err
		}

		output.EmissionDestinations = emissionDestinationsValue

		outputList = append(outputList, output)
	}

	return &outputList, nil
}

func expandEmissionPolicyDestinationModelArray(inputList []EmissionPolicyDestinationModel) (*[]collectorpolicies.EmissionPolicyDestination, error) {
	var outputList []collectorpolicies.EmissionPolicyDestination
	for _, v := range inputList {
		input := v
		output := collectorpolicies.EmissionPolicyDestination{
			DestinationType: &input.DestinationType,
		}

		outputList = append(outputList, output)
	}

	return &outputList, nil
}

func expandIngestionPolicyPropertiesFormatModel(inputList []IngestionPolicyPropertiesFormatModel) (*collectorpolicies.IngestionPolicyPropertiesFormat, error) {
	if len(inputList) == 0 {
		return nil, nil
	}

	input := &inputList[0]
	output := collectorpolicies.IngestionPolicyPropertiesFormat{
		IngestionType: &input.IngestionType,
	}

	ingestionSourcesValue, err := expandIngestionSourcesPropertiesFormatModelArray(input.IngestionSources)
	if err != nil {
		return nil, err
	}

	output.IngestionSources = ingestionSourcesValue

	return &output, nil
}

func expandIngestionSourcesPropertiesFormatModelArray(inputList []IngestionSourcesPropertiesFormatModel) (*[]collectorpolicies.IngestionSourcesPropertiesFormat, error) {
	var outputList []collectorpolicies.IngestionSourcesPropertiesFormat
	for _, v := range inputList {
		input := v
		output := collectorpolicies.IngestionSourcesPropertiesFormat{
			SourceType: &input.SourceType,
		}

		if input.ResourceId != "" {
			output.ResourceId = &input.ResourceId
		}

		outputList = append(outputList, output)
	}

	return &outputList, nil
}

func flattenEmissionPoliciesPropertiesFormatModelArray(inputList *[]collectorpolicies.EmissionPoliciesPropertiesFormat) ([]EmissionPoliciesPropertiesFormatModel, error) {
	var outputList []EmissionPoliciesPropertiesFormatModel
	if inputList == nil {
		return outputList, nil
	}

	for _, input := range *inputList {
		output := EmissionPoliciesPropertiesFormatModel{}

		emissionDestinationsValue, err := flattenEmissionPolicyDestinationModelArray(input.EmissionDestinations)
		if err != nil {
			return nil, err
		}

		output.EmissionDestinations = emissionDestinationsValue

		if input.EmissionType != nil {
			output.EmissionType = *input.EmissionType
		}

		outputList = append(outputList, output)
	}

	return outputList, nil
}

func flattenEmissionPolicyDestinationModelArray(inputList *[]collectorpolicies.EmissionPolicyDestination) ([]EmissionPolicyDestinationModel, error) {
	var outputList []EmissionPolicyDestinationModel
	if inputList == nil {
		return outputList, nil
	}

	for _, input := range *inputList {
		output := EmissionPolicyDestinationModel{}

		if input.DestinationType != nil {
			output.DestinationType = *input.DestinationType
		}

		outputList = append(outputList, output)
	}

	return outputList, nil
}

func flattenIngestionPolicyPropertiesFormatModel(input *collectorpolicies.IngestionPolicyPropertiesFormat) ([]IngestionPolicyPropertiesFormatModel, error) {
	var outputList []IngestionPolicyPropertiesFormatModel
	if input == nil {
		return outputList, nil
	}

	output := IngestionPolicyPropertiesFormatModel{}

	ingestionSourcesValue, err := flattenIngestionSourcesPropertiesFormatModelArray(input.IngestionSources)
	if err != nil {
		return nil, err
	}

	output.IngestionSources = ingestionSourcesValue

	if input.IngestionType != nil {
		output.IngestionType = *input.IngestionType
	}

	return append(outputList, output), nil
}

func flattenIngestionSourcesPropertiesFormatModelArray(inputList *[]collectorpolicies.IngestionSourcesPropertiesFormat) ([]IngestionSourcesPropertiesFormatModel, error) {
	var outputList []IngestionSourcesPropertiesFormatModel
	if inputList == nil {
		return outputList, nil
	}

	for _, input := range *inputList {
		output := IngestionSourcesPropertiesFormatModel{}

		if input.ResourceId != nil {
			output.ResourceId = *input.ResourceId
		}

		if input.SourceType != nil {
			output.SourceType = *input.SourceType
		}

		outputList = append(outputList, output)
	}

	return outputList, nil
}
