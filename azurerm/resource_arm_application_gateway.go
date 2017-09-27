package azurerm

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"path"
	//	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/jen20/riviera/azure"
)

func resourceArmApplicationGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmApplicationGatewayCreate,
		Read:   resourceArmApplicationGatewayRead,
		Update: resourceArmApplicationGatewayCreate,
		Delete: resourceArmApplicationGatewayDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"sku": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.StandardSmall),
								string(network.StandardMedium),
								string(network.StandardLarge),
								string(network.WAFLarge),
								string(network.WAFMedium),
							}, true),
						},

						"tier": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Standard),
								string(network.WAF),
							}, true),
						},

						"capacity": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 10),
						},
					},
				},
				Set: hashApplicationGatewaySku,
			},

			"disabled_ssl_protocols": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
					ValidateFunc: validation.StringInSlice([]string{
						string(network.TLSv10),
						string(network.TLSv11),
						string(network.TLSv12),
					}, true),
				},
			},

			"waf_configuration": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},

						"firewall_mode": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Detection),
								string(network.Prevention),
							}, true),
						},

						"rule_set_type": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc:     validation.StringInSlice([]string{"OWASP"}, true),
						},

						"rule_set_version": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc:     validation.StringInSlice([]string{"2.2.9", "3.0"}, true),
						},
					},
				},
				Set: hashApplicationGatewayWafConfig,
			},

			"gateway_ip_configuration": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"subnet_id": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"frontend_port": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"port": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"frontend_ip_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"subnet_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"private_ip_address": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"public_ip_address_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"private_ip_address_allocation": {
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Dynamic),
								string(network.Static),
							}, true),
						},
					},
				},
			},

			"backend_address_pool": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"ip_address_list": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"fqdn_list": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},

			"backend_http_settings": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"port": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"protocol": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.HTTP),
								string(network.HTTPS),
							}, true),
						},

						"cookie_based_affinity": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Enabled),
								string(network.Disabled),
							}, true),
						},

						"request_timeout": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"authentication_certificate": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},

									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},

						"probe_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"probe_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"http_listener": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"frontend_ip_configuration_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"frontend_ip_configuration_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"frontend_port_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"frontend_port_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"protocol": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.HTTP),
								string(network.HTTPS),
							}, true),
						},

						"host_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"ssl_certificate_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"ssl_certificate_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"require_sni": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"probe": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"protocol": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.HTTP),
								string(network.HTTPS),
							}, true),
						},

						"path": {
							Type:     schema.TypeString,
							Required: true,
						},

						"host": {
							Type:     schema.TypeString,
							Required: true,
						},

						"interval": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"timeout": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"unhealthy_threshold": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"request_routing_rule": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"rule_type": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.Basic),
								string(network.PathBasedRouting),
							}, true),
						},

						"http_listener_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"http_listener_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"backend_address_pool_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"backend_address_pool_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"backend_http_settings_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"backend_http_settings_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"url_path_map_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"url_path_map_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"url_path_map": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"default_backend_address_pool_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"default_backend_address_pool_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"default_backend_http_settings_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"default_backend_http_settings_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"path_rule": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},

									"name": {
										Type:     schema.TypeString,
										Required: true,
									},

									"paths": {
										Type:     schema.TypeList,
										Required: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},

									"backend_address_pool_name": {
										Type:     schema.TypeString,
										Required: true,
									},

									"backend_address_pool_id": {
										Type:     schema.TypeString,
										Computed: true,
									},

									"backend_http_settings_name": {
										Type:     schema.TypeString,
										Required: true,
									},

									"backend_http_settings_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},

			"authentication_certificate": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"data": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
					},
				},
			},

			"ssl_certificate": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"data": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},

						"password": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},

						"public_cert_data": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmApplicationGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)
	client := armClient.applicationGatewayClient

	log.Printf("[INFO] preparing arguments for AzureRM ApplicationGateway creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	// Gateway ID is needed to link sub-resources together in expand functions
	gatewayID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/applicationGateways/%s",
		armClient.subscriptionId, resGroup, name)

	properties := network.ApplicationGatewayPropertiesFormat{}
	properties.Sku = expandApplicationGatewaySku(d)
	properties.SslPolicy = expandApplicationGatewaySslPolicy(d)
	properties.GatewayIPConfigurations = expandApplicationGatewayIPConfigurations(d)
	properties.FrontendPorts = expandApplicationGatewayFrontendPorts(d)
	properties.FrontendIPConfigurations = expandApplicationGatewayFrontendIPConfigurations(d)
	properties.BackendAddressPools = expandApplicationGatewayBackendAddressPools(d)
	properties.BackendHTTPSettingsCollection = expandApplicationGatewayBackendHTTPSettings(d, gatewayID)
	properties.HTTPListeners = expandApplicationGatewayHTTPListeners(d, gatewayID)
	properties.Probes = expandApplicationGatewayProbes(d)
	properties.RequestRoutingRules = expandApplicationGatewayRequestRoutingRules(d, gatewayID)
	properties.URLPathMaps = expandApplicationGatewayURLPathMaps(d, gatewayID)
	properties.AuthenticationCertificates = expandApplicationGatewayAuthenticationCertificates(d)
	properties.SslCertificates = expandApplicationGatewaySslCertificates(d)

	if _, ok := d.GetOk("waf_configuration"); ok {
		properties.WebApplicationFirewallConfiguration = expandApplicationGatewayWafConfig(d)
	}

	gateway := network.ApplicationGateway{
		Name:     azure.String(name),
		Location: azure.String(location),
		Tags:     expandTags(tags),
		ApplicationGatewayPropertiesFormat: &properties,
	}

	_, errChan := client.CreateOrUpdate(resGroup, name, gateway, make(chan struct{}))
	err := <-errChan
	if err != nil {
		return errwrap.Wrapf("Error Creating/Updating ApplicationGateway {{err}}", err)
	}

	read, err := client.Get(resGroup, name)
	if err != nil {
		return errwrap.Wrapf("Error Getting ApplicationGateway {{err}}", err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read ApplicationGateway %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmApplicationGatewayRead(d, meta)
}

func resourceArmApplicationGatewayRead(d *schema.ResourceData, meta interface{}) error {
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return errwrap.Wrapf("Error parsing ApplicationGateway ID {{err}}", err)
	}

	ApplicationGateway, exists, err := retrieveApplicationGatewayById(d.Id(), meta)
	if err != nil {
		return errwrap.Wrapf("Error Getting ApplicationGateway By ID {{err}}", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] ApplicationGateway %q not found. Removing from state", d.Get("name").(string))
		return nil
	}

	d.Set("name", ApplicationGateway.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("location", ApplicationGateway.Location)
	d.Set("sku", schema.NewSet(hashApplicationGatewaySku, flattenApplicationGatewaySku(ApplicationGateway.ApplicationGatewayPropertiesFormat.Sku)))
	d.Set("disabled_ssl_protocols", flattenApplicationGatewaySslPolicy(ApplicationGateway.ApplicationGatewayPropertiesFormat.SslPolicy))
	d.Set("gateway_ip_configuration", flattenApplicationGatewayIPConfigurations(ApplicationGateway.ApplicationGatewayPropertiesFormat.GatewayIPConfigurations))
	d.Set("frontend_port", flattenApplicationGatewayFrontendPorts(ApplicationGateway.ApplicationGatewayPropertiesFormat.FrontendPorts))
	d.Set("frontend_ip_configuration", flattenApplicationGatewayFrontendIPConfigurations(ApplicationGateway.ApplicationGatewayPropertiesFormat.FrontendIPConfigurations))
	d.Set("backend_address_pool", flattenApplicationGatewayBackendAddressPools(ApplicationGateway.ApplicationGatewayPropertiesFormat.BackendAddressPools))
	d.Set("backend_http_settings", flattenApplicationGatewayBackendHTTPSettings(ApplicationGateway.ApplicationGatewayPropertiesFormat.BackendHTTPSettingsCollection))
	d.Set("http_listener", flattenApplicationGatewayHTTPListeners(ApplicationGateway.ApplicationGatewayPropertiesFormat.HTTPListeners))
	d.Set("probe", flattenApplicationGatewayProbes(ApplicationGateway.ApplicationGatewayPropertiesFormat.Probes))
	d.Set("request_routing_rule", flattenApplicationGatewayRequestRoutingRules(ApplicationGateway.ApplicationGatewayPropertiesFormat.RequestRoutingRules))
	d.Set("url_path_map", flattenApplicationGatewayURLPathMaps(ApplicationGateway.ApplicationGatewayPropertiesFormat.URLPathMaps))
	d.Set("authentication_certificate", schema.NewSet(hashApplicationGatewayAuthenticationCertificates, flattenApplicationGatewayAuthenticationCertificates(ApplicationGateway.ApplicationGatewayPropertiesFormat.AuthenticationCertificates)))
	d.Set("ssl_certificate", schema.NewSet(hashApplicationGatewaySslCertificates, flattenApplicationGatewaySslCertificates(ApplicationGateway.ApplicationGatewayPropertiesFormat.SslCertificates)))

	if ApplicationGateway.ApplicationGatewayPropertiesFormat.WebApplicationFirewallConfiguration != nil {
		d.Set("waf_configuration", schema.NewSet(hashApplicationGatewayWafConfig,
			flattenApplicationGatewayWafConfig(ApplicationGateway.ApplicationGatewayPropertiesFormat.WebApplicationFirewallConfiguration)))
	}

	flattenAndSetTags(d, ApplicationGateway.Tags)

	return nil
}

func resourceArmApplicationGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).applicationGatewayClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return errwrap.Wrapf("Error Parsing Azure Resource ID {{err}}", err)
	}
	resGroup := id.ResourceGroup
	name := id.Path["applicationGateways"]

	_, errChan := client.Delete(resGroup, name, make(chan struct{}))
	err = <-errChan
	if err != nil {
		return errwrap.Wrapf("Error Deleting ApplicationGateway {{err}}", err)
	}

	d.SetId("")
	return nil
}

func ApplicationGatewayResGroupAndNameFromID(ApplicationGatewayID string) (string, string, error) {
	id, err := parseAzureResourceID(ApplicationGatewayID)
	if err != nil {
		return "", "", err
	}
	name := id.Path["applicationGateways"]
	resGroup := id.ResourceGroup

	return resGroup, name, nil
}

func retrieveApplicationGatewayById(ApplicationGatewayID string, meta interface{}) (*network.ApplicationGateway, bool, error) {
	client := meta.(*ArmClient).applicationGatewayClient

	resGroup, name, err := ApplicationGatewayResGroupAndNameFromID(ApplicationGatewayID)
	if err != nil {
		return nil, false, errwrap.Wrapf("Error Getting ApplicationGateway Name and Group: {{err}}", err)
	}

	resp, err := client.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("Error making Read request on Azure ApplicationGateway %s: %+v", name, err)
	}

	return &resp, true, nil
}

func ApplicationGatewayStateRefreshFunc(client *ArmClient, resourceGroupName string, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.applicationGatewayClient.Get(resourceGroupName, name)
		if err != nil {
			return nil, "", fmt.Errorf(
				"Error issuing read request in ApplicationGatewayStateRefreshFunc to Azure ARM for ApplicationGateway '%s' (RG: '%s'): %+v",
				name, resourceGroupName, err)
		}

		return res, *res.ApplicationGatewayPropertiesFormat.ProvisioningState, nil
	}
}

func expandApplicationGatewaySku(d *schema.ResourceData) *network.ApplicationGatewaySku {
	skuSet := d.Get("sku").(*schema.Set).List()
	sku := skuSet[0].(map[string]interface{})

	name := sku["name"].(string)
	tier := sku["tier"].(string)
	capacity := int32(sku["capacity"].(int))

	return &network.ApplicationGatewaySku{
		Name:     network.ApplicationGatewaySkuName(name),
		Tier:     network.ApplicationGatewayTier(tier),
		Capacity: &capacity,
	}
}

func expandApplicationGatewayWafConfig(d *schema.ResourceData) *network.ApplicationGatewayWebApplicationFirewallConfiguration {
	wafSet := d.Get("waf_configuration").(*schema.Set).List()
	waf := wafSet[0].(map[string]interface{})

	enabled := waf["enabled"].(bool)
	mode := waf["firewall_mode"].(string)
<<<<<<< HEAD
<<<<<<< HEAD
	rulesettype := waf["rule_set_type"].(string)
	rulesetversion := waf["rule_set_version"].(string)

	return &network.ApplicationGatewayWebApplicationFirewallConfiguration{
		Enabled:        &enabled,
		FirewallMode:   network.ApplicationGatewayFirewallMode(mode),
		RuleSetType:    &rulesettype,
		RuleSetVersion: &rulesetversion,
=======

	return &network.ApplicationGatewayWebApplicationFirewallConfiguration{
		Enabled:      &enabled,
		FirewallMode: network.ApplicationGatewayFirewallMode(mode),
>>>>>>> made sure files are added
=======
	rulesettype := waf["rule_set_type"].(string)
	rulesetversion := waf["rule_set_version"].(string)

	return &network.ApplicationGatewayWebApplicationFirewallConfiguration{
		Enabled:        &enabled,
		FirewallMode:   network.ApplicationGatewayFirewallMode(mode),
		RuleSetType:    &rulesettype,
		RuleSetVersion: &rulesetversion,
>>>>>>> changes based on reviewer feedback
	}
}

func expandApplicationGatewaySslPolicy(d *schema.ResourceData) *network.ApplicationGatewaySslPolicy {
	disabledProtoList := d.Get("disabled_ssl_protocols").([]interface{})
	disabled := []network.ApplicationGatewaySslProtocol{}

	for _, proto := range disabledProtoList {
		disabled = append(disabled, network.ApplicationGatewaySslProtocol(proto.(string)))
	}

	return &network.ApplicationGatewaySslPolicy{
		DisabledSslProtocols: &disabled,
	}
}

func expandApplicationGatewayIPConfigurations(d *schema.ResourceData) *[]network.ApplicationGatewayIPConfiguration {
	configs := d.Get("gateway_ip_configuration").([]interface{})
	ipConfigurations := make([]network.ApplicationGatewayIPConfiguration, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		subnetID := data["subnet_id"].(string)

		ipConfig := network.ApplicationGatewayIPConfiguration{
			Name: &name,
			ApplicationGatewayIPConfigurationPropertiesFormat: &network.ApplicationGatewayIPConfigurationPropertiesFormat{
				Subnet: &network.SubResource{
					ID: &subnetID,
				},
			},
		}
		ipConfigurations = append(ipConfigurations, ipConfig)
	}

	return &ipConfigurations
}

func expandApplicationGatewayFrontendPorts(d *schema.ResourceData) *[]network.ApplicationGatewayFrontendPort {
	configs := d.Get("frontend_port").([]interface{})
	frontendPorts := make([]network.ApplicationGatewayFrontendPort, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		port := int32(data["port"].(int))

		portConfig := network.ApplicationGatewayFrontendPort{
			Name: &name,
			ApplicationGatewayFrontendPortPropertiesFormat: &network.ApplicationGatewayFrontendPortPropertiesFormat{
				Port: &port,
			},
		}
		frontendPorts = append(frontendPorts, portConfig)
	}

	return &frontendPorts
}

func expandApplicationGatewayFrontendIPConfigurations(d *schema.ResourceData) *[]network.ApplicationGatewayFrontendIPConfiguration {
	configs := d.Get("frontend_ip_configuration").([]interface{})
	frontEndConfigs := make([]network.ApplicationGatewayFrontendIPConfiguration, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		properties := network.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{}

		if v := data["subnet_id"].(string); v != "" {
			properties.Subnet = &network.SubResource{
				ID: &v,
			}
		}

		if v := data["private_ip_address_allocation"].(string); v != "" {
			properties.PrivateIPAllocationMethod = network.IPAllocationMethod(v)
		}

		if v := data["private_ip_address"].(string); v != "" {
			properties.PrivateIPAddress = &v
		}

		if v := data["public_ip_address_id"].(string); v != "" {
			properties.PublicIPAddress = &network.SubResource{
				ID: &v,
			}
		}

		name := data["name"].(string)
		frontEndConfig := network.ApplicationGatewayFrontendIPConfiguration{
			Name: &name,
			ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &properties,
		}

		frontEndConfigs = append(frontEndConfigs, frontEndConfig)
	}

	return &frontEndConfigs
}

func expandApplicationGatewayBackendAddressPools(d *schema.ResourceData) *[]network.ApplicationGatewayBackendAddressPool {
	configs := d.Get("backend_address_pool").([]interface{})
	backendPools := make([]network.ApplicationGatewayBackendAddressPool, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		backendAddresses := []network.ApplicationGatewayBackendAddress{}

		for _, rawIP := range data["ip_address_list"].([]interface{}) {
			ip := rawIP.(string)
			backendAddresses = append(backendAddresses, network.ApplicationGatewayBackendAddress{IPAddress: &ip})
		}

		for _, rawFQDN := range data["fqdn_list"].([]interface{}) {
			fqdn := rawFQDN.(string)
			backendAddresses = append(backendAddresses, network.ApplicationGatewayBackendAddress{Fqdn: &fqdn})
		}

		name := data["name"].(string)
		pool := network.ApplicationGatewayBackendAddressPool{
			Name: &name,
			ApplicationGatewayBackendAddressPoolPropertiesFormat: &network.ApplicationGatewayBackendAddressPoolPropertiesFormat{
				BackendAddresses: &backendAddresses,
			},
		}

		backendPools = append(backendPools, pool)
	}

	return &backendPools
}

func expandApplicationGatewayBackendHTTPSettings(d *schema.ResourceData, gatewayID string) *[]network.ApplicationGatewayBackendHTTPSettings {
	configs := d.Get("backend_http_settings").([]interface{})
	backendSettings := make([]network.ApplicationGatewayBackendHTTPSettings, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		port := int32(data["port"].(int))
		protocol := data["protocol"].(string)
		cookieBasedAffinity := data["cookie_based_affinity"].(string)
		requestTimeout := int32(data["request_timeout"].(int))

		setting := network.ApplicationGatewayBackendHTTPSettings{
			Name: &name,
			ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
				Port:                &port,
				Protocol:            network.ApplicationGatewayProtocol(protocol),
				CookieBasedAffinity: network.ApplicationGatewayCookieBasedAffinity(cookieBasedAffinity),
				RequestTimeout:      &requestTimeout,
			},
		}

		if data["authentication_certificate"] != nil {
			authCerts := data["authentication_certificate"].([]interface{})
			authCertSubResources := make([]network.SubResource, 0, len(authCerts))

			for _, rawAuthCert := range authCerts {
				authCert := rawAuthCert.(map[string]interface{})
				authCertID := fmt.Sprintf("%s/authenticationCertificates/%s", gatewayID, authCert["name"])
				authCertSubResource := network.SubResource{
					ID: &authCertID,
				}

				authCertSubResources = append(authCertSubResources, authCertSubResource)
			}

			setting.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.AuthenticationCertificates = &authCertSubResources
		}

		probeName := data["probe_name"].(string)
		if probeName != "" {
			probeID := fmt.Sprintf("%s/probes/%s", gatewayID, probeName)
			setting.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.Probe = &network.SubResource{
				ID: &probeID,
			}
		}

		backendSettings = append(backendSettings, setting)
	}

	return &backendSettings
}

func expandApplicationGatewayHTTPListeners(d *schema.ResourceData, gatewayID string) *[]network.ApplicationGatewayHTTPListener {
	configs := d.Get("http_listener").([]interface{})
	httpListeners := make([]network.ApplicationGatewayHTTPListener, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		frontendIPConfigName := data["frontend_ip_configuration_name"].(string)
		frontendIPConfigID := fmt.Sprintf("%s/frontendIPConfigurations/%s", gatewayID, frontendIPConfigName)
		frontendPortName := data["frontend_port_name"].(string)
		frontendPortID := fmt.Sprintf("%s/frontendPorts/%s", gatewayID, frontendPortName)
		protocol := data["protocol"].(string)

		listener := network.ApplicationGatewayHTTPListener{
			Name: &name,
			ApplicationGatewayHTTPListenerPropertiesFormat: &network.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration: &network.SubResource{
					ID: &frontendIPConfigID,
				},
				FrontendPort: &network.SubResource{
					ID: &frontendPortID,
				},
				Protocol: network.ApplicationGatewayProtocol(protocol),
			},
		}

		if host := data["host_name"].(string); host != "" {
			listener.ApplicationGatewayHTTPListenerPropertiesFormat.HostName = &host
		}

		if sslCertName := data["ssl_certificate_name"].(string); sslCertName != "" {
			certID := fmt.Sprintf("%s/sslCertificates/%s", gatewayID, sslCertName)
			listener.ApplicationGatewayHTTPListenerPropertiesFormat.SslCertificate = &network.SubResource{
				ID: &certID,
			}
		}

		if requireSNI, ok := data["require_sni"].(bool); ok {
			listener.ApplicationGatewayHTTPListenerPropertiesFormat.RequireServerNameIndication = &requireSNI
		}

		httpListeners = append(httpListeners, listener)
	}

	return &httpListeners
}

func expandApplicationGatewayProbes(d *schema.ResourceData) *[]network.ApplicationGatewayProbe {
	configs := d.Get("probe").([]interface{})
	backendSettings := make([]network.ApplicationGatewayProbe, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		protocol := data["protocol"].(string)
		probePath := data["path"].(string)
		host := data["host"].(string)
		interval := int32(data["interval"].(int))
		timeout := int32(data["timeout"].(int))
		unhealthyThreshold := int32(data["unhealthy_threshold"].(int))

		setting := network.ApplicationGatewayProbe{
			Name: &name,
			ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
				Protocol:           network.ApplicationGatewayProtocol(protocol),
				Path:               &probePath,
				Host:               &host,
				Interval:           &interval,
				Timeout:            &timeout,
				UnhealthyThreshold: &unhealthyThreshold,
			},
		}

		backendSettings = append(backendSettings, setting)
	}

	return &backendSettings
}

func expandApplicationGatewayRequestRoutingRules(d *schema.ResourceData, gatewayID string) *[]network.ApplicationGatewayRequestRoutingRule {
	configs := d.Get("request_routing_rule").([]interface{})
	rules := make([]network.ApplicationGatewayRequestRoutingRule, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		ruleType := data["rule_type"].(string)
		httpListenerName := data["http_listener_name"].(string)
		httpListenerID := fmt.Sprintf("%s/httpListeners/%s", gatewayID, httpListenerName)

		rule := network.ApplicationGatewayRequestRoutingRule{
			Name: &name,
			ApplicationGatewayRequestRoutingRulePropertiesFormat: &network.ApplicationGatewayRequestRoutingRulePropertiesFormat{
				RuleType: network.ApplicationGatewayRequestRoutingRuleType(ruleType),
				HTTPListener: &network.SubResource{
					ID: &httpListenerID,
				},
			},
		}

		if backendAddressPoolName := data["backend_address_pool_name"].(string); backendAddressPoolName != "" {
			backendAddressPoolID := fmt.Sprintf("%s/backendAddressPools/%s", gatewayID, backendAddressPoolName)
			rule.ApplicationGatewayRequestRoutingRulePropertiesFormat.BackendAddressPool = &network.SubResource{
				ID: &backendAddressPoolID,
			}
		}

		if backendHTTPSettingsName := data["backend_http_settings_name"].(string); backendHTTPSettingsName != "" {
			backendHTTPSettingsID := fmt.Sprintf("%s/backendHttpSettingsCollection/%s", gatewayID, backendHTTPSettingsName)
			rule.ApplicationGatewayRequestRoutingRulePropertiesFormat.BackendHTTPSettings = &network.SubResource{
				ID: &backendHTTPSettingsID,
			}
		}

		if urlPathMapName := data["url_path_map_name"].(string); urlPathMapName != "" {
			urlPathMapID := fmt.Sprintf("%s/urlPathMaps/%s", gatewayID, urlPathMapName)
			rule.ApplicationGatewayRequestRoutingRulePropertiesFormat.URLPathMap = &network.SubResource{
				ID: &urlPathMapID,
			}
		}

		rules = append(rules, rule)
	}

	return &rules
}

func expandApplicationGatewayURLPathMaps(d *schema.ResourceData, gatewayID string) *[]network.ApplicationGatewayURLPathMap {
	configs := d.Get("url_path_map").([]interface{})
	pathMaps := make([]network.ApplicationGatewayURLPathMap, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		name := data["name"].(string)
		defaultBackendAddressPoolName := data["default_backend_address_pool_name"].(string)
		defaultBackendAddressPoolID := fmt.Sprintf("%s/backendAddressPools/%s", gatewayID, defaultBackendAddressPoolName)
		defaultBackendHTTPSettingsName := data["default_backend_http_settings_name"].(string)
		defaultBackendHTTPSettingsID := fmt.Sprintf("%s/backendHttpSettingsCollection/%s", gatewayID, defaultBackendHTTPSettingsName)

		pathRules := []network.ApplicationGatewayPathRule{}
		for _, ruleConfig := range data["path_rule"].([]interface{}) {
			ruleConfigMap := ruleConfig.(map[string]interface{})

			ruleName := ruleConfigMap["name"].(string)

			rulePaths := []string{}
			for _, rulePath := range ruleConfigMap["paths"].([]interface{}) {
				rulePaths = append(rulePaths, rulePath.(string))
			}

			rule := network.ApplicationGatewayPathRule{
				Name: &ruleName,
				ApplicationGatewayPathRulePropertiesFormat: &network.ApplicationGatewayPathRulePropertiesFormat{
					Paths: &rulePaths,
				},
			}

			if backendAddressPoolName := ruleConfigMap["backend_address_pool_name"].(string); backendAddressPoolName != "" {
				backendAddressPoolID := fmt.Sprintf("%s/backendAddressPools/%s", gatewayID, backendAddressPoolName)
				rule.ApplicationGatewayPathRulePropertiesFormat.BackendAddressPool = &network.SubResource{
					ID: &backendAddressPoolID,
				}
			}

			if backendHTTPSettingsName := ruleConfigMap["backend_http_settings_name"].(string); backendHTTPSettingsName != "" {
				backendHTTPSettingsID := fmt.Sprintf("%s/backendHttpSettingsCollection/%s", gatewayID, backendHTTPSettingsName)
				rule.ApplicationGatewayPathRulePropertiesFormat.BackendHTTPSettings = &network.SubResource{
					ID: &backendHTTPSettingsID,
				}
			}

			pathRules = append(pathRules, rule)
		}

		pathMap := network.ApplicationGatewayURLPathMap{
			Name: &name,
			ApplicationGatewayURLPathMapPropertiesFormat: &network.ApplicationGatewayURLPathMapPropertiesFormat{
				DefaultBackendAddressPool: &network.SubResource{
					ID: &defaultBackendAddressPoolID,
				},
				DefaultBackendHTTPSettings: &network.SubResource{
					ID: &defaultBackendHTTPSettingsID,
				},
				PathRules: &pathRules,
			},
		}

		pathMaps = append(pathMaps, pathMap)
	}

	return &pathMaps
}

func expandApplicationGatewayAuthenticationCertificates(d *schema.ResourceData) *[]network.ApplicationGatewayAuthenticationCertificate {
	configs := d.Get("authentication_certificate").([]interface{})
	authCerts := make([]network.ApplicationGatewayAuthenticationCertificate, 0, len(configs))

	for _, configRaw := range configs {
		raw := configRaw.(map[string]interface{})

		name := raw["name"].(string)
		data := raw["data"].(string)

		// data must be base64 encoded
		data = base64.StdEncoding.EncodeToString([]byte(data))

		cert := network.ApplicationGatewayAuthenticationCertificate{
			Name: &name,
			ApplicationGatewayAuthenticationCertificatePropertiesFormat: &network.ApplicationGatewayAuthenticationCertificatePropertiesFormat{
				Data: &data,
			},
		}

		authCerts = append(authCerts, cert)
	}

	return &authCerts
}

func expandApplicationGatewaySslCertificates(d *schema.ResourceData) *[]network.ApplicationGatewaySslCertificate {
	configs := d.Get("ssl_certificate").([]interface{})
	sslCerts := make([]network.ApplicationGatewaySslCertificate, 0, len(configs))

	for _, configRaw := range configs {
		raw := configRaw.(map[string]interface{})

		name := raw["name"].(string)
		data := raw["data"].(string)
		password := raw["password"].(string)

		// data must be base64 encoded
		data = base64.StdEncoding.EncodeToString([]byte(data))

		cert := network.ApplicationGatewaySslCertificate{
			Name: &name,
			ApplicationGatewaySslCertificatePropertiesFormat: &network.ApplicationGatewaySslCertificatePropertiesFormat{
				Data:     &data,
				Password: &password,
			},
		}

		sslCerts = append(sslCerts, cert)
	}

	return &sslCerts
}

func flattenApplicationGatewaySku(sku *network.ApplicationGatewaySku) []interface{} {
	result := make(map[string]interface{})

	result["name"] = string(sku.Name)
	result["tier"] = string(sku.Tier)
	result["capacity"] = int(*sku.Capacity)

	return []interface{}{result}
}

func flattenApplicationGatewayWafConfig(waf *network.ApplicationGatewayWebApplicationFirewallConfiguration) []interface{} {
	result := make(map[string]interface{})

	result["enabled"] = *waf.Enabled
	result["firewall_mode"] = string(waf.FirewallMode)
<<<<<<< HEAD
<<<<<<< HEAD
	result["rule_set_type"] = waf.RuleSetType
	result["rule_set_version"] = waf.RuleSetVersion
=======
>>>>>>> made sure files are added
=======
	result["rule_set_type"] = waf.RuleSetType
	result["rule_set_version"] = waf.RuleSetVersion
>>>>>>> changes based on reviewer feedback

	return []interface{}{result}
}

func flattenApplicationGatewaySslPolicy(policy *network.ApplicationGatewaySslPolicy) []interface{} {
	result := make([]interface{}, 0, len(*policy.DisabledSslProtocols))

	for _, proto := range *policy.DisabledSslProtocols {
		result = append(result, string(proto))
	}

	return result
}

func flattenApplicationGatewayIPConfigurations(ipConfigs *[]network.ApplicationGatewayIPConfiguration) []interface{} {
	result := make([]interface{}, 0, len(*ipConfigs))

	for _, config := range *ipConfigs {
		ipConfig := map[string]interface{}{
			"id":        *config.ID,
			"name":      *config.Name,
			"subnet_id": *config.ApplicationGatewayIPConfigurationPropertiesFormat.Subnet.ID,
		}

		result = append(result, ipConfig)
	}

	return result
}

func flattenApplicationGatewayFrontendPorts(portConfigs *[]network.ApplicationGatewayFrontendPort) []interface{} {
	result := make([]interface{}, 0, len(*portConfigs))

	for _, config := range *portConfigs {
		port := map[string]interface{}{
			"id":   *config.ID,
			"name": *config.Name,
			"port": int(*config.ApplicationGatewayFrontendPortPropertiesFormat.Port),
		}

		result = append(result, port)
	}

	return result
}

func flattenApplicationGatewayFrontendIPConfigurations(ipConfigs *[]network.ApplicationGatewayFrontendIPConfiguration) []interface{} {
	result := make([]interface{}, 0, len(*ipConfigs))
	for _, config := range *ipConfigs {
		ipConfig := make(map[string]interface{})
		ipConfig["id"] = *config.ID
		ipConfig["name"] = *config.Name

		if config.ApplicationGatewayFrontendIPConfigurationPropertiesFormat.PrivateIPAllocationMethod != "" {
			ipConfig["private_ip_address_allocation"] = config.ApplicationGatewayFrontendIPConfigurationPropertiesFormat.PrivateIPAllocationMethod
		}

		if config.ApplicationGatewayFrontendIPConfigurationPropertiesFormat.Subnet != nil {
			ipConfig["subnet_id"] = *config.ApplicationGatewayFrontendIPConfigurationPropertiesFormat.Subnet.ID
		}

		if config.ApplicationGatewayFrontendIPConfigurationPropertiesFormat.PrivateIPAddress != nil {
			ipConfig["private_ip_address"] = *config.ApplicationGatewayFrontendIPConfigurationPropertiesFormat.PrivateIPAddress
		}

		if config.ApplicationGatewayFrontendIPConfigurationPropertiesFormat.PublicIPAddress != nil {
			ipConfig["public_ip_address_id"] = *config.ApplicationGatewayFrontendIPConfigurationPropertiesFormat.PublicIPAddress.ID
		}

		result = append(result, ipConfig)
	}
	return result
}

func flattenApplicationGatewayBackendAddressPools(poolConfigs *[]network.ApplicationGatewayBackendAddressPool) []interface{} {
	result := make([]interface{}, 0, len(*poolConfigs))

	for _, config := range *poolConfigs {
		ipAddressList := []interface{}{}
		fqdnList := []interface{}{}
		for _, address := range *config.ApplicationGatewayBackendAddressPoolPropertiesFormat.BackendAddresses {
			if address.IPAddress != nil {
				ipAddressList = append(ipAddressList, *address.IPAddress)
			} else if address.Fqdn != nil {
				fqdnList = append(fqdnList, *address.Fqdn)
			}
		}

		pool := map[string]interface{}{
			"id":              *config.ID,
			"name":            *config.Name,
			"ip_address_list": ipAddressList,
			"fqdn_list":       fqdnList,
		}

		result = append(result, pool)
	}

	return result
}

func flattenApplicationGatewayBackendHTTPSettings(backendSettings *[]network.ApplicationGatewayBackendHTTPSettings) []interface{} {
	result := make([]interface{}, 0, len(*backendSettings))

	for _, config := range *backendSettings {
		settings := map[string]interface{}{
			"id":                    *config.ID,
			"name":                  *config.Name,
			"port":                  int(*config.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.Port),
			"protocol":              string(config.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.Protocol),
			"cookie_based_affinity": string(config.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.CookieBasedAffinity),
			"request_timeout":       int(*config.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.RequestTimeout),
		}

		if config.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.AuthenticationCertificates != nil {
			authCerts := make([]interface{}, 0, len(*config.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.AuthenticationCertificates))

			for _, config := range *config.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.AuthenticationCertificates {
				authCert := map[string]interface{}{
					"name": path.Base(*config.ID),
					"id":   *config.ID,
				}

				authCerts = append(authCerts, authCert)
			}

			settings["authentication_certificate"] = authCerts
		}

		if config.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.Probe != nil {
			settings["probe_name"] = path.Base(*config.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.Probe.ID)
			settings["probe_id"] = *config.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.Probe.ID
		}

		result = append(result, settings)
	}

	return result
}

func flattenApplicationGatewayHTTPListeners(httpListeners *[]network.ApplicationGatewayHTTPListener) []interface{} {
	result := make([]interface{}, 0, len(*httpListeners))

	for _, config := range *httpListeners {
		listener := map[string]interface{}{
			"id":   *config.ID,
			"name": *config.Name,
			"frontend_ip_configuration_id":   *config.ApplicationGatewayHTTPListenerPropertiesFormat.FrontendIPConfiguration.ID,
			"frontend_ip_configuration_name": path.Base(*config.ApplicationGatewayHTTPListenerPropertiesFormat.FrontendIPConfiguration.ID),
			"frontend_port_name":             path.Base(*config.ApplicationGatewayHTTPListenerPropertiesFormat.FrontendPort.ID),
			"frontend_port_id":               *config.ApplicationGatewayHTTPListenerPropertiesFormat.FrontendPort.ID,
			"protocol":                       string(config.ApplicationGatewayHTTPListenerPropertiesFormat.Protocol),
		}

		if config.ApplicationGatewayHTTPListenerPropertiesFormat.HostName != nil {
			listener["host_name"] = *config.ApplicationGatewayHTTPListenerPropertiesFormat.HostName
		}

		if config.ApplicationGatewayHTTPListenerPropertiesFormat.SslCertificate != nil {
			listener["ssl_certificate_name"] = path.Base(*config.ApplicationGatewayHTTPListenerPropertiesFormat.SslCertificate.ID)
			listener["ssl_certificate_id"] = *config.ApplicationGatewayHTTPListenerPropertiesFormat.SslCertificate.ID
		}

		if config.ApplicationGatewayHTTPListenerPropertiesFormat.RequireServerNameIndication != nil {
			listener["require_sni"] = *config.ApplicationGatewayHTTPListenerPropertiesFormat.RequireServerNameIndication
		}

		result = append(result, listener)
	}

	return result
}

func flattenApplicationGatewayProbes(probes *[]network.ApplicationGatewayProbe) []interface{} {
	result := make([]interface{}, 0, len(*probes))

	for _, config := range *probes {
		settings := map[string]interface{}{
			"id":                  *config.ID,
			"name":                *config.Name,
			"protocol":            string(config.ApplicationGatewayProbePropertiesFormat.Protocol),
			"path":                *config.ApplicationGatewayProbePropertiesFormat.Path,
			"host":                *config.ApplicationGatewayProbePropertiesFormat.Host,
			"interval":            int(*config.ApplicationGatewayProbePropertiesFormat.Interval),
			"timeout":             int(*config.ApplicationGatewayProbePropertiesFormat.Timeout),
			"unhealthy_threshold": int(*config.ApplicationGatewayProbePropertiesFormat.UnhealthyThreshold),
		}

		result = append(result, settings)
	}

	return result
}

func flattenApplicationGatewayRequestRoutingRules(rules *[]network.ApplicationGatewayRequestRoutingRule) []interface{} {
	result := make([]interface{}, 0, len(*rules))

	for _, config := range *rules {
		listener := map[string]interface{}{
			"id":                 *config.ID,
			"name":               *config.Name,
			"rule_type":          string(config.ApplicationGatewayRequestRoutingRulePropertiesFormat.RuleType),
			"http_listener_id":   *config.ApplicationGatewayRequestRoutingRulePropertiesFormat.HTTPListener.ID,
			"http_listener_name": path.Base(*config.ApplicationGatewayRequestRoutingRulePropertiesFormat.HTTPListener.ID),
		}

		if config.ApplicationGatewayRequestRoutingRulePropertiesFormat.BackendAddressPool != nil {
			listener["backend_address_pool_name"] = path.Base(*config.ApplicationGatewayRequestRoutingRulePropertiesFormat.BackendAddressPool.ID)
			listener["backend_address_pool_id"] = *config.ApplicationGatewayRequestRoutingRulePropertiesFormat.BackendAddressPool.ID
		}

		if config.ApplicationGatewayRequestRoutingRulePropertiesFormat.BackendHTTPSettings != nil {
			listener["backend_http_settings_name"] = path.Base(*config.ApplicationGatewayRequestRoutingRulePropertiesFormat.BackendHTTPSettings.ID)
			listener["backend_http_settings_id"] = *config.ApplicationGatewayRequestRoutingRulePropertiesFormat.BackendHTTPSettings.ID
		}

		if config.ApplicationGatewayRequestRoutingRulePropertiesFormat.URLPathMap != nil {
			listener["url_path_map_name"] = path.Base(*config.ApplicationGatewayRequestRoutingRulePropertiesFormat.URLPathMap.ID)
			listener["url_path_map_id"] = *config.ApplicationGatewayRequestRoutingRulePropertiesFormat.URLPathMap.ID
		}

		result = append(result, listener)
	}

	return result
}

func flattenApplicationGatewayURLPathMaps(pathMaps *[]network.ApplicationGatewayURLPathMap) []interface{} {
	result := make([]interface{}, 0, len(*pathMaps))

	for _, config := range *pathMaps {
		pathMap := map[string]interface{}{
			"id":   *config.ID,
			"name": *config.Name,
		}

		if config.ApplicationGatewayURLPathMapPropertiesFormat.DefaultBackendAddressPool != nil {
			pathMap["default_backend_address_pool_name"] = path.Base(*config.ApplicationGatewayURLPathMapPropertiesFormat.DefaultBackendAddressPool.ID)
			pathMap["default_backend_address_pool_id"] = *config.ApplicationGatewayURLPathMapPropertiesFormat.DefaultBackendAddressPool.ID
		}

		if config.ApplicationGatewayURLPathMapPropertiesFormat.DefaultBackendHTTPSettings != nil {
			pathMap["default_backend_http_settings_name"] = path.Base(*config.ApplicationGatewayURLPathMapPropertiesFormat.DefaultBackendHTTPSettings.ID)
			pathMap["default_backend_http_settings_id"] = *config.ApplicationGatewayURLPathMapPropertiesFormat.DefaultBackendHTTPSettings.ID
		}

		pathRules := make([]interface{}, 0, len(*config.ApplicationGatewayURLPathMapPropertiesFormat.PathRules))
		for _, pathRuleConfig := range *config.ApplicationGatewayURLPathMapPropertiesFormat.PathRules {
			rule := map[string]interface{}{
				"id":   *pathRuleConfig.ID,
				"name": *pathRuleConfig.Name,
			}

			if pathRuleConfig.ApplicationGatewayPathRulePropertiesFormat.BackendAddressPool != nil {
				rule["backend_address_pool_name"] = path.Base(*pathRuleConfig.ApplicationGatewayPathRulePropertiesFormat.BackendAddressPool.ID)
				rule["backend_address_pool_id"] = *pathRuleConfig.ApplicationGatewayPathRulePropertiesFormat.BackendAddressPool.ID
			}

			if pathRuleConfig.ApplicationGatewayPathRulePropertiesFormat.BackendHTTPSettings != nil {
				rule["backend_http_settings_name"] = path.Base(*pathRuleConfig.ApplicationGatewayPathRulePropertiesFormat.BackendHTTPSettings.ID)
				rule["backend_http_settings_id"] = *pathRuleConfig.ApplicationGatewayPathRulePropertiesFormat.BackendHTTPSettings.ID
			}

			paths := make([]interface{}, 0, len(*pathRuleConfig.ApplicationGatewayPathRulePropertiesFormat.Paths))
			for _, rulePath := range *pathRuleConfig.ApplicationGatewayPathRulePropertiesFormat.Paths {
				paths = append(paths, rulePath)
			}
			rule["paths"] = paths

			pathRules = append(pathRules, rule)
		}
		pathMap["path_rule"] = pathRules

		result = append(result, pathMap)
	}

	return result
}

func flattenApplicationGatewayAuthenticationCertificates(certs *[]network.ApplicationGatewayAuthenticationCertificate) []interface{} {
	result := make([]interface{}, 0, len(*certs))

	for _, config := range *certs {
		certConfig := map[string]interface{}{
			"id":   *config.ID,
			"name": *config.Name,
		}

		result = append(result, certConfig)
	}

	return result
}

func flattenApplicationGatewaySslCertificates(certs *[]network.ApplicationGatewaySslCertificate) []interface{} {
	result := make([]interface{}, 0, len(*certs))

	for _, config := range *certs {
		certConfig := map[string]interface{}{
			"id":               *config.ID,
			"name":             *config.Name,
			"public_cert_data": *config.ApplicationGatewaySslCertificatePropertiesFormat.PublicCertData,
		}

		result = append(result, certConfig)
	}

	return result
}

func hashApplicationGatewaySku(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["tier"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["capacity"].(int)))

	return hashcode.String(buf.String())
}

func hashApplicationGatewayWafConfig(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%t-", m["enabled"].(bool)))
	buf.WriteString(fmt.Sprintf("%s-", m["firewall_mode"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["rule_set_type"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["rule_set_version"].(string)))

	return hashcode.String(buf.String())
}

func hashApplicationGatewayAuthenticationCertificates(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))

	return hashcode.String(buf.String())
}

func hashApplicationGatewaySslCertificates(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["public_cert_data"].(string)))

	return hashcode.String(buf.String())
}
