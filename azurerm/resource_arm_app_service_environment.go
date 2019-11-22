package azurerm

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2019-08-01/web"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/response"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/suppress"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

type AppServiceEnvironmentFrontendPool struct {
	VMSize string
	Count  int32
}

type AppServiceEnvironmentFrontEndSKU string

const (
	SmallSKU  AppServiceEnvironmentFrontEndSKU = "Standard_D1_V2"
	MediumSKU AppServiceEnvironmentFrontEndSKU = "Standard_D2_V2"
	LargeSKU                                   = "Standard_D3_V2"
)

func resourceArmAppServiceEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmAppServiceEnvironmentCreate,
		Read:   resourceArmAppServiceEnvironmentRead,
		Update: resourceArmAppServiceEnvironmentUpdate,
		Delete: resourceArmAppServiceEnvironmentDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(time.Hour * 30),
			Update: schema.DefaultTimeout(time.Hour * 30),
			Delete: schema.DefaultTimeout(time.Hour * 30),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAppServicePlanName,
			},

			"resource_group_name": azure.SchemaResourceGroupName(),

			"location": azure.SchemaLocation(),

			"number_of_ip_addresses": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 10),
			},

			"internal_load_balancing_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  string(web.InternalLoadBalancingModeNone),
				ValidateFunc: validation.StringInSlice([]string{
					string(web.InternalLoadBalancingModeNone),
					string(web.InternalLoadBalancingModePublishing),
					string(web.InternalLoadBalancingModeWeb),
				}, true),
				DiffSuppressFunc: suppress.CaseDifference,
			},

			"virtual_network": {
				Type:     schema.TypeList,
				Required: true,
				Optional: false,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"virtual_network_id": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: suppress.CaseDifference,
						},
						"subnet_name": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: suppress.CaseDifference,
						},
					},
				},
			},

			"frontend_pool": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vm_size": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(web.WorkerSizeOptionsSmall),
								string(web.WorkerSizeOptionsMedium),
								string(web.WorkerSizeOptionsLarge),
							}, true),
							DiffSuppressFunc: suppress.CaseDifference,
						},
						"number_of_workers": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntAtLeast(2),
						},
					},
				},
			},
			"worker_pool": {
				Type:     schema.TypeList,
				Optional: true, //not required for ASEV2
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"worker_size_id": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"worker_size": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(web.WorkerSizeOptionsSmall),
								string(web.WorkerSizeOptionsMedium),
								string(web.WorkerSizeOptionsLarge),
							}, true),
							DiffSuppressFunc: suppress.CaseDifference,
						},
						"worker_count": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"tags": tags.Schema(),
		},
	}
}
func resourceArmAppServiceEnvironmentCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).Web.AppServiceEnvironmentsClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*ArmClient).StopContext, d)
	defer cancel()

	resourceGroup := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)
	location := azure.NormalizeLocation(d.Get("location").(string))
	numberOfSSLPublicIPs := d.Get("number_of_ip_addresses").(int)
	internalLoadBalancingMode := d.Get("internal_load_balancing_mode").(string)
	tags := d.Get("tags").(map[string]interface{})

	frontendPool, err := expandAppServiceEnvironmentFrontendPool(d)
	if err != nil {
		return fmt.Errorf("Error expanding `frontend_pool`: %+v", err)
	}

	workerPools, err := expandAppServiceEnvironmentWorkerPool(d)
	if err != nil {
		return fmt.Errorf("Error expanding `worker_pool`: %+v", err)
	}

	// TODO: cluster settings (which are optional).
	// this needs investigation to see if this should this be a whitelist or just a Map
	/*
		 "clusterSettings": [
			  {
					"name": "DefaultSslCertificateThumbprint",
					"value": "SomeThumbprint.."
			  }
		 ]
	*/

	var virtualNetwork *web.VirtualNetworkProfile

	if _, ok := d.GetOk("virtual_network"); ok {
		virtualNetwork, err = expandAppServiceEnvironmentVirtualNetwork(d)
		if err != nil {
			return fmt.Errorf("Error expanding `virtual_network`: %+v", err)
		}
	}

	envelope := web.AppServiceEnvironmentResource{
		Location: utils.String(location),
		// TODO: work out how's best to handle ASEV2 support
		Kind: utils.String("ASEV2"),
		Tags: expandTags(tags),

		AppServiceEnvironment: &web.AppServiceEnvironment{
			// this is one of the older API's where name + location are required in this block
			Name:     utils.String(name),
			Location: utils.String(location),

			IpsslAddressCount:         utils.Int32(int32(numberOfSSLPublicIPs)),
			InternalLoadBalancingMode: web.InternalLoadBalancingMode(internalLoadBalancingMode),
			WorkerPools:               workerPools,
			MultiSize:                 utils.String(frontendPool.VMSize),
			MultiRoleCount:            utils.Int32(frontendPool.Count),
			VirtualNetwork:            virtualNetwork,
		},
	}

	future, err := client.CreateOrUpdate(ctx, resourceGroup, name, envelope)
	if err != nil {
		return fmt.Errorf("Error creating App Service Environment %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return fmt.Errorf("Error waiting for the creation of App Service Environment %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	read, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		return fmt.Errorf("Error retrieving App Service Environment %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	d.SetId(*read.ID)

	return resourceArmAppServiceEnvironmentRead(d, meta)
}

func resourceArmAppServiceEnvironmentUpdate(d *schema.ResourceData, meta interface{}) error {
	//client := meta.(*ArmClient).appServiceEnvironmentsClient
	// TODO: note there is a separate update function in the SDK, don't use CreateOrUpdate
	return resourceArmAppServiceEnvironmentRead(d, meta)
}

func resourceArmAppServiceEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).Web.AppServiceEnvironmentsClient
	ctx, cancel := timeouts.ForRead(meta.(*ArmClient).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resourceGroup := id.ResourceGroup
	name := id.Path["hostingEnvironments"]

	appServiceEnvironment, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(appServiceEnvironment.Response) {
			log.Printf("[DEBUG] App Service Environmment %q (Resource Group %q) was not found!", name, resourceGroup)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving App Service Environmment %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resourceGroup)
	if location := appServiceEnvironment.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(d.Get("location").(string)))
	}
	flattenAndSetTags(d, appServiceEnvironment.Tags)

	if props := appServiceEnvironment.AppServiceEnvironment; props != nil {
		d.Set("internal_load_balancing_mode", props.InternalLoadBalancingMode)

		if count := props.IpsslAddressCount; count != nil {
			d.Set("number_of_ip_addresses", int(*count))
		}

		frontendPool := flattenAppServiceEnvironmentFrontendPool(props)
		if err := d.Set("frontend_pool", frontendPool); err != nil {
			return fmt.Errorf("Error flattening `frontend_pool`: %+v", err)
		}

		if workerPools := props.WorkerPools; workerPools != nil {
			workerPools := flattenAppServiceEnvironmentWorkerPools(props.WorkerPools)
			if err := d.Set("worker_pool", workerPools); err != nil {
				return fmt.Errorf("Error flattening `worker_pool`: %+v", err)
			}
		}

		virtualNetwork := flattenAppServiceEnvironmentVirtualNetwork(props.VirtualNetwork)
		if err := d.Set("virtual_network", virtualNetwork); err != nil {
			return fmt.Errorf("Error flattening `virtual_network`: %+v", err)
		}
	}

	return nil
}

func resourceArmAppServiceEnvironmentDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).Web.AppServiceEnvironmentsClient
	ctx, cancel := timeouts.ForDelete(meta.(*ArmClient).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resGroup := id.ResourceGroup
	name := id.Path["hostingEnvironments"]

	log.Printf("[DEBUG] Deleting App Service Environment %q (Resource Group %q)", name, resGroup)

	// `true` will delete any child resources (e.g. App Services / Plans / Certificates etc)
	forceDelete := utils.Bool(true)
	future, err := client.Delete(ctx, resGroup, name, forceDelete)
	if err != nil {
		if response.WasNotFound(future.Response()) {
			return nil
		}

		return err
	}

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		if response.WasNotFound(future.Response()) {
			return nil
		}

		return err
	}

	return nil
}

func expandAppServiceEnvironmentVirtualNetwork(d *schema.ResourceData) (*web.VirtualNetworkProfile, error) {
	networks := d.Get("virtual_network").([]interface{})
	if len(networks) == 0 {
		return &web.VirtualNetworkProfile{}, nil
	}

	network := networks[0].(map[string]interface{})

	virtualNetworkId := network["virtual_network_id"].(string)
	subnetName := network["subnet_name"].(string)

	profile := web.VirtualNetworkProfile{
		ID:     utils.String(virtualNetworkId),
		Subnet: utils.String(subnetName),
	}
	return &profile, nil
}

func flattenAppServiceEnvironmentVirtualNetwork(input *web.VirtualNetworkProfile) []interface{} {
	output := make(map[string]interface{}, 0)

	if id := input.ID; id != nil {
		output["virtual_network_id"] = *id
	}

	if subnetName := input.Subnet; subnetName != nil {
		output["subnet_name"] = *subnetName
	}

	return []interface{}{output}
}

func expandAppServiceEnvironmentFrontendPool(d *schema.ResourceData) (*AppServiceEnvironmentFrontendPool, error) {
	inputs := d.Get("frontend_pool").([]interface{})
	input := inputs[0].(map[string]interface{})

	vmSize := input["vm_size"].(string)
	count := input["number_of_workers"].(int)
	pool := AppServiceEnvironmentFrontendPool{
		VMSize: vmSize,
		Count:  int32(count),
	}

	return &pool, nil
}

func flattenAppServiceEnvironmentFrontendPool(input *web.AppServiceEnvironment) []interface{} {
	output := make(map[string]interface{}, 0)

	if size := input.MultiSize; size != nil {
		output["vm_size"] = translateSKUToSimpleSize(AppServiceEnvironmentFrontEndSKU(*size))
	}

	if count := input.MultiRoleCount; count != nil {
		output["number_of_workers"] = *count
	}

	return []interface{}{output}
}

func expandAppServiceEnvironmentWorkerPool(d *schema.ResourceData) (*[]web.WorkerPool, error) {
	inputs := d.Get("worker_pool").([]interface{})
	outputs := make([]web.WorkerPool, 0)

	for _, v := range inputs {
		input := v.(map[string]interface{})

		workerSizeId := input["worker_size_id"].(int)
		workerSize := input["worker_size"].(string)
		workerCount := input["worker_count"].(int)

		output := web.WorkerPool{
			WorkerSizeID: utils.Int32(int32(workerSizeId)),
			WorkerSize:   utils.String(workerSize),
			WorkerCount:  utils.Int32(int32(workerCount)),
		}

		outputs = append(outputs, output)
	}

	return &outputs, nil
}

func flattenAppServiceEnvironmentWorkerPools(input *[]web.WorkerPool) []interface{} {
	outputs := make([]interface{}, 0)

	for _, pool := range *input {
		output := make(map[string]interface{}, 0)

		if sizeId := pool.WorkerSizeID; sizeId != nil {
			output["worker_size_id"] = int(*sizeId)
		}
		if size := pool.WorkerSize; size != nil {
			output["worker_size"] = *size
		}
		if count := pool.WorkerCount; count != nil {
			output["worker_count"] = int(*count)
		}

		outputs = append(outputs, output)
	}

	return outputs
}

// this is required as the API returns the actual SKUs for MultiSize instead of the simple (Small,Medium,Large) values
func translateSKUToSimpleSize(sku AppServiceEnvironmentFrontEndSKU) string {
	switch sku {
	case SmallSKU:
		return string(web.WorkerSizeOptionsSmall)
	case MediumSKU:
		return string(web.WorkerSizeOptionsMedium)
	case LargeSKU:
		return string(web.WorkerSizeOptionsLarge)
	}
	return string(web.WorkerSizeOptionsSmall)
}
