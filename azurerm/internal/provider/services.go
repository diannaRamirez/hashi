package provider

import (
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/analysisservices"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/apimanagement"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/applicationinsights"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/authorization"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/automation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/batch"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/bot"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/cdn"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/cognitive"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/common"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/compute"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/containers"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/cosmos"
)

func SupportedServices() []common.ServiceRegistration {
	return []common.ServiceRegistration{
		analysisservices.Registration{},
		apimanagement.Registration{},
		applicationinsights.Registration{},
		authorization.Registration{},
		automation.Registration{},
		batch.Registration{},
		bot.Registration{},
		cdn.Registration{},
		cognitive.Registration{},
		compute.Registration{},
		containers.Registration{},
		cosmos.Registration{},
	}
}
