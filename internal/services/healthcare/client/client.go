package client

import (
	"github.com/Azure/azure-sdk-for-go/services/healthcareapis/mgmt/2021-11-01/healthcareapis"
	"github.com/hashicorp/terraform-provider-azurerm/internal/common"
)

type Client struct {
	HealthcareServiceClient                              *healthcareapis.ServicesClient
	HealthcareWorkspaceClient                            *healthcareapis.WorkspacesClient
	HealthcareWorkspaceIotConnectorClient                *healthcareapis.IotConnectorsClient
	HealthcareWorkspaceIotConnectorFhirDestinationClient *healthcareapis.IotConnectorFhirDestinationClient
}

func NewClient(o *common.ClientOptions) *Client {
	HealthcareServiceClient := healthcareapis.NewServicesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&HealthcareServiceClient.Client, o.ResourceManagerAuthorizer)

	HealthcareWorkspaceClient := healthcareapis.NewWorkspacesClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&HealthcareWorkspaceClient.Client, o.ResourceManagerAuthorizer)

	HealthcareWorkspaceIotConnectorClient := healthcareapis.NewIotConnectorsClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&HealthcareWorkspaceIotConnectorClient.Client, o.ResourceManagerAuthorizer)

	HealthcareWorkspaceIotConnectorFhirDestinationClient := healthcareapis.NewIotConnectorFhirDestinationClientWithBaseURI(o.ResourceManagerEndpoint, o.SubscriptionId)
	o.ConfigureClient(&HealthcareWorkspaceIotConnectorFhirDestinationClient.Client, o.ResourceManagerAuthorizer)

	return &Client{
		HealthcareServiceClient:                              &HealthcareServiceClient,
		HealthcareWorkspaceClient:                            &HealthcareWorkspaceClient,
		HealthcareWorkspaceIotConnectorClient:                &HealthcareWorkspaceIotConnectorClient,
		HealthcareWorkspaceIotConnectorFhirDestinationClient: &HealthcareWorkspaceIotConnectorFhirDestinationClient,
	}
}
