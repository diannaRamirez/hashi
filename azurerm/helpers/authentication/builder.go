package authentication

import (
	"fmt"
	"log"
)

type Builder struct {
	// Core
	ClientID                 string
	SubscriptionID           string
	TenantID                 string
	Environment              string
	SkipProviderRegistration bool

	// Azure CLI Parsing / CloudShell Auth
	SupportsAzureCliCloudShellParsing bool

	// Managed Service Identity Auth
	SupportsManagedServiceIdentity bool
	MsiEndpoint                    string

	// Service Principal (Client Secret) Auth
	SupportsClientSecretAuth bool
	ClientSecret             string
}

func (b Builder) Build() (*Config, error) {
	config := Config{
		ClientID:                 b.ClientID,
		SubscriptionID:           b.SubscriptionID,
		TenantID:                 b.TenantID,
		Environment:              b.Environment,
		SkipProviderRegistration: b.SkipProviderRegistration,
	}

	if b.SupportsClientSecretAuth && b.ClientSecret != "" {
		log.Printf("[DEBUG] Using Service Principal / Client Secret for Authentication")
		config.AuthenticatedAsAServicePrincipal = true

		config.authMethod = newServicePrincipalClientSecretAuth(b)
		return config.validate()
	}

	if b.SupportsManagedServiceIdentity {
		log.Printf("[DEBUG] Using Managed Service Identity for Authentication")
		method, err := newManagedServiceIdentityAuth(b)
		if err != nil {
			return nil, err
		}
		config.authMethod = method
		return config.validate()
	}

	// note: this includes CloudShell
	if b.SupportsAzureCliCloudShellParsing {
		log.Printf("[DEBUG] Parsing credentials from the Azure CLI for Authentication")

		method, err := newAzureCliParsingAuth(b)
		if err != nil {
			return nil, err
		}

		// as credentials are parsed from the Azure CLI's Profile we actually need to
		// obtain the ClientId, Environment, Subscription ID & TenantID here
		config.ClientID = method.profile.clientId
		config.Environment = method.profile.environment
		config.SubscriptionID = method.profile.subscriptionId
		config.TenantID = method.profile.tenantId

		config.authMethod = method
		return config.validate()
	}

	return nil, fmt.Errorf("No supported authentication methods were found!")
}
