package databox

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/databox/mgmt/2019-09-01/databox"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/hashicorp/go-azure-helpers/response"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/suppress"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/databox/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	azSchema "github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmDataBoxJob() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDataBoxJobCreate,
		Read:   resourceArmDataBoxJobRead,
		Update: resourceArmDataBoxJobUpdate,
		Delete: resourceArmDataBoxJobDelete,

		Importer: azSchema.ValidateResourceIDPriorToImport(func(id string) error {
			_, err := parse.ParseDataBoxJobID(id)
			return err
		}),

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: ValidateDataBoxJobName,
			},

			"location": azure.SchemaLocation(),

			"resource_group_name": azure.SchemaResourceGroupName(),

			"contact_details": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: ValidateDataBoxJobContactName,
						},
						"emails": {
							Type:     schema.TypeSet,
							Required: true,
							MaxItems: 10,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: ValidateDataBoxJobEmail,
							},
						},
						"phone_number": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: ValidateDataBoxJobPhoneNumber,
						},
						"mobile": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},
						"notification_preference": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"at_azure_dc": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
									},
									"data_copied": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
									},
									"delivered": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
									},
									"device_prepared": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
									},
									"dispatched": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
									},
									"picked_up": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
									},
								},
							},
						},
						"phone_extension": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: ValidateDataBoxJobPhoneExtension,
						},
					},
				},
			},

			"destination_account": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 10,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(databox.DataDestinationTypeStorageAccount),
								string(databox.DataDestinationTypeManagedDisk),
							}, false),
						},
						"resource_group_id": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: azure.ValidateResourceID,
						},
						"share_password": {
							Type:         schema.TypeString,
							Optional:     true,
							Sensitive:    true,
							ForceNew:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},
						"staging_storage_account_id": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: azure.ValidateResourceID,
						},
						"storage_account_id": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: azure.ValidateResourceID,
						},
					},
				},
			},

			"preferred_shipment_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(databox.CustomerManaged),
					string(databox.MicrosoftManaged),
				}, false),
			},

			"shipping_address": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"city": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: ValidateDataBoxJobCity,
						},
						"country": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},
						"postal_code": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: ValidateDataBoxJobPostCode,
						},
						"state_or_province": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},
						"street_address_1": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: ValidateDataBoxJobStreetAddress,
						},
						"address_type": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(databox.Commercial),
								string(databox.None),
								string(databox.Residential),
							}, false),
						},
						"company_name": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: ValidateDataBoxJobCompanyName,
						},
						"street_address_2": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: ValidateDataBoxJobStreetAddress,
						},
						"street_address_3": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: ValidateDataBoxJobStreetAddress,
						},
						"postal_code_ext": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: ValidateDataBoxJobPostCode,
						},
					},
				},
			},

			"sku_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(databox.DataBox),
					string(databox.DataBoxDisk),
					string(databox.DataBoxHeavy),
				}, false),
			},

			"databox_disk_passkey": {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ForceNew:     true,
				ValidateFunc: ValidateDataBoxJobDiskPassKey,
			},

			"databox_preferred_disk_count": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},

			"databox_preferred_disk_size_in_tb": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},

			"datacenter_region_preference": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},

			"delivery_scheduled_date_time": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppress.RFC3339Time,
				ValidateFunc:     validation.IsRFC3339Time,
			},

			"delivery_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(databox.NonScheduled),
					string(databox.Scheduled),
				}, false),
			},

			"device_password": {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"expected_data_size_in_tb": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntBetween(1, 1000000),
			},

			"tags": tags.Schema(),
		},
	}
}

func resourceArmDataBoxJobCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DataBox.JobClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resourceGroup, name, "Details")
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for present of existing DataBox Job (DataBox Job Name %q / Resource Group %q): %+v", name, resourceGroup, err)
			}
		}
		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_databox_job", *existing.ID)
		}
	}

	location := azure.NormalizeLocation(d.Get("location").(string))
	contactDetails := d.Get("contact_details").([]interface{})
	deliveryType := d.Get("delivery_type").(string)
	destinationAccount := d.Get("destination_account").([]interface{})
	devicePassword := d.Get("device_password").(string)
	diskPassKey := d.Get("databox_disk_passkey").(string)
	preferredShipmentType := d.Get("preferred_shipment_type").(string)
	datacenterRegionPreference := d.Get("datacenter_region_preference").(*schema.Set).List()
	shippingAddress := d.Get("shipping_address").([]interface{})
	skuName := d.Get("sku_name").(string)
	t := d.Get("tags").(map[string]interface{})

	var expectedDataSizeInTB *int32
	if v, ok := d.GetOkExists("expected_data_size_in_tb"); ok {
		expectedDataSizeInTB = utils.Int32(int32(v.(int)))
	}

	var databoxPreferredDiskCount *int
	if v, ok := d.GetOkExists("databox_preferred_disk_count"); ok {
		databoxPreferredDiskCount = utils.Int(v.(int))
	}

	var databoxPreferredDiskSizeInTB *int
	if v, ok := d.GetOkExists("databox_preferred_disk_size_in_tb"); ok {
		databoxPreferredDiskSizeInTB = utils.Int(v.(int))
	}

	if !(databoxPreferredDiskCount == nil && databoxPreferredDiskSizeInTB == nil) && !(databoxPreferredDiskCount != nil && databoxPreferredDiskSizeInTB != nil) {
		return fmt.Errorf("Error setting `databox_preferred_disk_count` and `databox_preferred_disk_size_in_tb`: They should be set or not set together")
	}

	parameters := databox.JobResource{
		Location: utils.String(location),
		JobProperties: &databox.JobProperties{
			DeliveryType: databox.JobDeliveryType(deliveryType),
		},
		Sku: &databox.Sku{
			Name: databox.SkuName(skuName),
		},
		Tags: tags.Expand(t),
	}

	if v, ok := d.GetOk("delivery_scheduled_date_time"); ok {
		t, _ := time.Parse(time.RFC3339, v.(string))
		parameters.JobProperties.DeliveryInfo = &databox.JobDeliveryInfo{
			ScheduledDateTime: &date.Time{Time: t},
		}
	}

	switch skuName {
	case string(databox.DataBox):
		parameters.JobProperties.Details = &databox.JobDetailsType{
			ContactDetails:              expandArmDataBoxJobContactDetails(contactDetails),
			DestinationAccountDetails:   expandArmDataBoxJobDestinationAccount(destinationAccount),
			DevicePassword:              utils.String(devicePassword),
			ExpectedDataSizeInTerabytes: expectedDataSizeInTB,
			Preferences: &databox.Preferences{
				TransportPreferences: &databox.TransportPreferences{
					PreferredShipmentType: databox.TransportShipmentTypes(preferredShipmentType),
				},
				PreferredDataCenterRegion: utils.ExpandStringSlice(datacenterRegionPreference),
			},
			ShippingAddress: expandArmDataBoxJobShippingAddress(shippingAddress),
		}
	case string(databox.DataBoxDisk):
		parameters.JobProperties.Details = &databox.DiskJobDetails{
			ContactDetails:              expandArmDataBoxJobContactDetails(contactDetails),
			DestinationAccountDetails:   expandArmDataBoxJobDestinationAccount(destinationAccount),
			ExpectedDataSizeInTerabytes: expectedDataSizeInTB,
			Passkey:                     utils.String(diskPassKey),
			Preferences: &databox.Preferences{
				TransportPreferences: &databox.TransportPreferences{
					PreferredShipmentType: databox.TransportShipmentTypes(preferredShipmentType),
				},
				PreferredDataCenterRegion: utils.ExpandStringSlice(datacenterRegionPreference),
			},
			PreferredDisks:  expandArmDataBoxJobPreferredDisks(databoxPreferredDiskSizeInTB, databoxPreferredDiskCount),
			ShippingAddress: expandArmDataBoxJobShippingAddress(shippingAddress),
		}
	case string(databox.DataBoxHeavy):
		parameters.JobProperties.Details = &databox.HeavyJobDetails{
			ContactDetails:              expandArmDataBoxJobContactDetails(contactDetails),
			DevicePassword:              utils.String(devicePassword),
			DestinationAccountDetails:   expandArmDataBoxJobDestinationAccount(destinationAccount),
			ExpectedDataSizeInTerabytes: expectedDataSizeInTB,
			Preferences: &databox.Preferences{
				TransportPreferences: &databox.TransportPreferences{
					PreferredShipmentType: databox.TransportShipmentTypes(preferredShipmentType),
				},
				PreferredDataCenterRegion: utils.ExpandStringSlice(datacenterRegionPreference),
			},
			ShippingAddress: expandArmDataBoxJobShippingAddress(shippingAddress),
		}
	}

	future, err := client.Create(ctx, resourceGroup, name, parameters)
	if err != nil {
		return fmt.Errorf("Error creating DataBox Job (DataBox Job Name %q / Resource Group %q): %+v", name, resourceGroup, err)
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for creation of DataBox Job (DataBox Job Name %q / Resource Group %q): %+v", name, resourceGroup, err)
	}

	resp, err := client.Get(ctx, resourceGroup, name, "Details")
	if err != nil {
		return fmt.Errorf("Error retrieving DataBox Job (DataBox Job Name %q / Resource Group %q): %+v", name, resourceGroup, err)
	}
	if resp.ID == nil || *resp.ID == "" {
		return fmt.Errorf("Cannot read DataBox Job (DataBox Job Name %q / Resource Group %q) ID", name, resourceGroup)
	}
	d.SetId(*resp.ID)

	return resourceArmDataBoxJobRead(d, meta)
}

func resourceArmDataBoxJobRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DataBox.JobClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.ParseDataBoxJobID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.Name, "Details")
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] DataBox Job %q does not exist - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading DataBox Job (DataBox Job Name %q / Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}
	d.Set("sku_name", resp.Sku.Name)

	if props := resp.JobProperties; props != nil {
		if props.DeliveryInfo != nil && props.DeliveryInfo.ScheduledDateTime != nil {
			d.Set("delivery_scheduled_date_time", (*props.DeliveryInfo.ScheduledDateTime).Format(time.RFC3339))
		}
		d.Set("delivery_type", props.DeliveryType)

		if details := props.Details; details != nil {
			if v, ok := details.AsJobDetailsType(); ok && v != nil {
				d.Set("device_password", v.DevicePassword)

				if err := d.Set("contact_details", flattenArmDataBoxJobContactDetails(v.ContactDetails)); err != nil {
					return fmt.Errorf("Error setting `contact_details`: %+v", err)
				}
				if err := d.Set("destination_account", flattenArmDataBoxJobDestinationAccount(v.DestinationAccountDetails)); err != nil {
					return fmt.Errorf("Error setting `destination_account`: %+v", err)
				}
				if v.Preferences != nil {
					if v.Preferences.TransportPreferences != nil {
						if err := d.Set("preferred_shipment_type", v.Preferences.TransportPreferences.PreferredShipmentType); err != nil {
							return fmt.Errorf("Error setting `preferred_shipment_type`: %+v", err)
						}
					}
					if err := d.Set("datacenter_region_preference", utils.FlattenStringSlice(v.Preferences.PreferredDataCenterRegion)); err != nil {
						return fmt.Errorf("Error setting `datacenter_region_preference`: %+v", err)
					}
				}
				if err := d.Set("shipping_address", flattenArmDataBoxJobShippingAddress(v.ShippingAddress)); err != nil {
					return fmt.Errorf("Error setting `shipping_address`: %+v", err)
				}
			} else if v, ok := details.AsDiskJobDetails(); ok && v != nil {
				for k, v := range v.PreferredDisks {
					diskSizeInTB, err := strconv.Atoi(k)
					if err != nil {
						return fmt.Errorf("Error converting `databox_preferred_disk_size_in_tb` %q to an int: %+v", k, err)
					}
					if err := d.Set("databox_preferred_disk_count", v); err != nil {
						return fmt.Errorf("Error setting `databox_preferred_disk_count`: %+v", err)
					}
					if err := d.Set("databox_preferred_disk_size_in_tb", diskSizeInTB); err != nil {
						return fmt.Errorf("Error setting `databox_preferred_disk_size_in_tb`: %+v", err)
					}
				}
				if err := d.Set("contact_details", flattenArmDataBoxJobContactDetails(v.ContactDetails)); err != nil {
					return fmt.Errorf("Error setting `contact_details`: %+v", err)
				}
				if err := d.Set("destination_account", flattenArmDataBoxJobDestinationAccount(v.DestinationAccountDetails)); err != nil {
					return fmt.Errorf("Error setting `destination_account`: %+v", err)
				}
				if v.Preferences != nil {
					if v.Preferences.TransportPreferences != nil {
						if err := d.Set("preferred_shipment_type", v.Preferences.TransportPreferences.PreferredShipmentType); err != nil {
							return fmt.Errorf("Error setting `preferred_shipment_type`: %+v", err)
						}
					}
					if err := d.Set("datacenter_region_preference", utils.FlattenStringSlice(v.Preferences.PreferredDataCenterRegion)); err != nil {
						return fmt.Errorf("Error setting `datacenter_region_preference`: %+v", err)
					}
				}
				if err := d.Set("shipping_address", flattenArmDataBoxJobShippingAddress(v.ShippingAddress)); err != nil {
					return fmt.Errorf("Error setting `shipping_address`: %+v", err)
				}
			} else if v, ok := details.AsHeavyJobDetails(); ok && v != nil {
				d.Set("device_password", v.DevicePassword)

				if err := d.Set("contact_details", flattenArmDataBoxJobContactDetails(v.ContactDetails)); err != nil {
					return fmt.Errorf("Error setting `contact_details`: %+v", err)
				}
				if err := d.Set("destination_account", flattenArmDataBoxJobDestinationAccount(v.DestinationAccountDetails)); err != nil {
					return fmt.Errorf("Error setting `destination_account`: %+v", err)
				}
				if v.Preferences != nil {
					if v.Preferences.TransportPreferences != nil {
						if err := d.Set("preferred_shipment_type", v.Preferences.TransportPreferences.PreferredShipmentType); err != nil {
							return fmt.Errorf("Error setting `preferred_shipment_type`: %+v", err)
						}
					}
					if err := d.Set("datacenter_region_preference", utils.FlattenStringSlice(v.Preferences.PreferredDataCenterRegion)); err != nil {
						return fmt.Errorf("Error setting `datacenter_region_preference`: %+v", err)
					}
				}
				if err := d.Set("shipping_address", flattenArmDataBoxJobShippingAddress(v.ShippingAddress)); err != nil {
					return fmt.Errorf("Error setting `shipping_address`: %+v", err)
				}
			}
		}
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmDataBoxJobUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DataBox.JobClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for DataBox Job update.")

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	contactDetails := d.Get("contact_details").([]interface{})
	shippingAddress := d.Get("shipping_address").([]interface{})
	t := d.Get("tags").(map[string]interface{})

	parameters := databox.JobResourceUpdateParameter{
		UpdateJobProperties: &databox.UpdateJobProperties{
			Details: &databox.UpdateJobDetails{
				ContactDetails:  expandArmDataBoxJobContactDetails(contactDetails),
				ShippingAddress: expandArmDataBoxJobShippingAddress(shippingAddress),
			},
		},
		Tags: tags.Expand(t),
	}

	future, err := client.Update(ctx, resourceGroup, name, parameters, "")
	if err != nil {
		return fmt.Errorf("Error updating DataBox Job %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for update of DataBox Job %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	resp, err := client.Get(ctx, resourceGroup, name, "Details")
	if err != nil {
		return fmt.Errorf("Error retrieving DataBox Job %q (Resource Group %q): %+v", name, resourceGroup, err)
	}
	if resp.ID == nil || *resp.ID == "" {
		return fmt.Errorf("Cannot read DataBox Job %s (resource group %s) ID", name, resourceGroup)
	}

	d.SetId(*resp.ID)

	return resourceArmDataBoxJobRead(d, meta)
}

func resourceArmDataBoxJobDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DataBox.JobClient
	lockClient := meta.(*clients.Client).Resource.LocksClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.ParseDataBoxJobID(d.Id())
	if err != nil {
		return err
	}

	reason := &databox.CancellationReason{
		Reason: utils.String("Cancel the order for deleting"),
	}

	resp, err := client.Cancel(ctx, id.ResourceGroup, id.Name, *reason)
	if err != nil {
		if utils.ResponseWasNotFound(resp) {
			return nil
		}

		return fmt.Errorf("Error cancelling Order (DataBox Job Name %q / Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	future, err := client.Delete(ctx, id.ResourceGroup, id.Name)
	if err != nil {
		return fmt.Errorf("Error deleting DataBox Job (DataBox Job Name %q / Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		if !response.WasNotFound(future.Response()) {
			return fmt.Errorf("Error waiting for deleting DataBox Job (DataBox Job Name %q / Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
		}
	}

	destinationAccount := d.Get("destination_account").([]interface{})
	for _, item := range destinationAccount {
		if item != nil {
			v := item.(map[string]interface{})
			dataDestinationType := v["type"].(string)
			lockName := "DATABOX_SERVICE"
			scope := ""

			switch dataDestinationType {
			case string(databox.DataDestinationTypeStorageAccount):
				scope = v["storage_account_id"].(string)
			case string(databox.DataDestinationTypeManagedDisk):
				scope = v["staging_storage_account_id"].(string)
			}

			if scope != "" {
				resp, err := lockClient.DeleteByScope(ctx, scope, lockName)
				if err != nil {
					if utils.ResponseWasNotFound(resp) {
						return nil
					}

					return fmt.Errorf("Error deleting Management Lock (Lock Name %q / Scope %q): %+v", lockName, scope, err)
				}
			}
		}
	}

	return nil
}

func expandArmDataBoxJobShippingAddress(input []interface{}) *databox.ShippingAddress {
	v := input[0].(map[string]interface{})

	return &databox.ShippingAddress{
		AddressType:     databox.AddressType(v["address_type"].(string)),
		City:            utils.String(v["city"].(string)),
		CompanyName:     utils.String(v["company_name"].(string)),
		Country:         utils.String(v["country"].(string)),
		PostalCode:      utils.String(v["postal_code"].(string)),
		StateOrProvince: utils.String(v["state_or_province"].(string)),
		StreetAddress1:  utils.String(v["street_address_1"].(string)),
		StreetAddress2:  utils.String(v["street_address_2"].(string)),
		StreetAddress3:  utils.String(v["street_address_3"].(string)),
		ZipExtendedCode: utils.String(v["postal_code_ext"].(string)),
	}
}

func expandArmDataBoxJobContactDetails(input []interface{}) *databox.ContactDetails {
	v := input[0].(map[string]interface{})

	return &databox.ContactDetails{
		ContactName:            utils.String(v["name"].(string)),
		EmailList:              utils.ExpandStringSlice(v["emails"].(*schema.Set).List()),
		Mobile:                 utils.String(v["mobile"].(string)),
		NotificationPreference: expandArmDataBoxJobNotificationPreference(v["notification_preference"].([]interface{})),
		Phone:                  utils.String(v["phone_number"].(string)),
		PhoneExtension:         utils.String(v["phone_extension"].(string)),
	}
}

func expandArmDataBoxJobDestinationAccount(input []interface{}) *[]databox.BasicDestinationAccountDetails {
	results := make([]databox.BasicDestinationAccountDetails, 0)
	for _, item := range input {
		if item != nil {
			v := item.(map[string]interface{})
			dataDestinationType := v["type"].(string)

			switch dataDestinationType {
			case string(databox.DataDestinationTypeStorageAccount):
				result := &databox.DestinationStorageAccountDetails{
					DataDestinationType: databox.DataDestinationTypeBasicDestinationAccountDetails(dataDestinationType),
					SharePassword:       utils.String(v["share_password"].(string)),
					StorageAccountID:    utils.String(v["storage_account_id"].(string)),
				}
				results = append(results, result)
			case string(databox.DataDestinationTypeManagedDisk):
				result := &databox.DestinationManagedDiskDetails{
					DataDestinationType:     databox.DataDestinationTypeBasicDestinationAccountDetails(dataDestinationType),
					ResourceGroupID:         utils.String(v["resource_group_id"].(string)),
					SharePassword:           utils.String(v["share_password"].(string)),
					StagingStorageAccountID: utils.String(v["staging_storage_account_id"].(string)),
				}
				results = append(results, result)
			}
		}
	}

	return &results
}

func expandArmDataBoxJobNotificationPreference(input []interface{}) *[]databox.NotificationPreference {
	results := make([]databox.NotificationPreference, 0)
	if len(input) == 0 {
		return &results
	}

	v := input[0].(map[string]interface{})

	devicePrepared := v["device_prepared"].(bool)
	results = append(results, databox.NotificationPreference{
		SendNotification: utils.Bool(devicePrepared),
		StageName:        databox.DevicePrepared,
	})

	dispatched := v["dispatched"].(bool)
	results = append(results, databox.NotificationPreference{
		SendNotification: utils.Bool(dispatched),
		StageName:        databox.Dispatched,
	})

	delivered := v["delivered"].(bool)
	results = append(results, databox.NotificationPreference{
		SendNotification: utils.Bool(delivered),
		StageName:        databox.Delivered,
	})

	pickedUp := v["picked_up"].(bool)
	results = append(results, databox.NotificationPreference{
		SendNotification: utils.Bool(pickedUp),
		StageName:        databox.PickedUp,
	})

	atAzureDC := v["at_azure_dc"].(bool)
	results = append(results, databox.NotificationPreference{
		SendNotification: utils.Bool(atAzureDC),
		StageName:        databox.AtAzureDC,
	})

	dataCopied := v["data_copied"].(bool)
	results = append(results, databox.NotificationPreference{
		SendNotification: utils.Bool(dataCopied),
		StageName:        databox.DataCopy,
	})

	return &results
}

func expandArmDataBoxJobPreferredDisks(preferredDiskSizeInTB *int, preferredDiskCount *int) map[string]*int32 {
	results := make(map[string]*int32)
	if preferredDiskCount == nil || preferredDiskSizeInTB == nil {
		return results
	}

	results[*utils.String(strconv.Itoa(*preferredDiskSizeInTB))] = utils.Int32(int32(*preferredDiskCount))

	return results
}

func flattenArmDataBoxJobShippingAddress(input *databox.ShippingAddress) []interface{} {
	results := make([]interface{}, 0)
	if input == nil {
		return results
	}

	city := ""
	if input.City != nil {
		city = *input.City
	}

	companyName := ""
	if input.CompanyName != nil {
		companyName = *input.CompanyName
	}

	country := ""
	if input.Country != nil {
		country = *input.Country
	}

	postalCode := ""
	if input.PostalCode != nil {
		postalCode = *input.PostalCode
	}

	stateOrProvince := ""
	if input.StateOrProvince != nil {
		stateOrProvince = *input.StateOrProvince
	}

	streetAddress1 := ""
	if input.StreetAddress1 != nil {
		streetAddress1 = *input.StreetAddress1
	}

	streetAddress2 := ""
	if input.StreetAddress2 != nil {
		streetAddress2 = *input.StreetAddress2
	}

	streetAddress3 := ""
	if input.StreetAddress3 != nil {
		streetAddress3 = *input.StreetAddress3
	}

	postalCodeExt := ""
	if input.ZipExtendedCode != nil {
		postalCodeExt = *input.ZipExtendedCode
	}

	results = append(results, map[string]interface{}{
		"address_type":      input.AddressType,
		"city":              city,
		"company_name":      companyName,
		"country":           country,
		"postal_code":       postalCode,
		"state_or_province": stateOrProvince,
		"street_address_1":  streetAddress1,
		"street_address_2":  streetAddress2,
		"street_address_3":  streetAddress3,
		"postal_code_ext":   postalCodeExt,
	})

	return results
}

func flattenArmDataBoxJobDestinationAccount(input *[]databox.BasicDestinationAccountDetails) []interface{} {
	results := make([]interface{}, 0)
	if input == nil {
		return results
	}

	for _, item := range *input {
		if item != nil {
			if v, ok := item.AsDestinationStorageAccountDetails(); ok && v != nil {
				dataDestinationType := v.DataDestinationType

				sharePassword := ""
				if v.SharePassword != nil {
					sharePassword = *v.SharePassword
				}

				storageAccountID := ""
				if v.StorageAccountID != nil {
					storageAccountID = *v.StorageAccountID
				}

				results = append(results, map[string]interface{}{
					"type":               dataDestinationType,
					"share_password":     sharePassword,
					"storage_account_id": storageAccountID,
				})
			} else if v, ok := item.AsDestinationManagedDiskDetails(); ok && v != nil {
				dataDestinationType := v.DataDestinationType

				resourceGroupID := ""
				if v.ResourceGroupID != nil {
					resourceGroupID = *v.ResourceGroupID
				}

				sharePassword := ""
				if v.SharePassword != nil {
					sharePassword = *v.SharePassword
				}

				stagingStorageAccountID := ""
				if v.StagingStorageAccountID != nil {
					stagingStorageAccountID = *v.StagingStorageAccountID
				}

				results = append(results, map[string]interface{}{
					"type":                       dataDestinationType,
					"resource_group_id":          resourceGroupID,
					"share_password":             sharePassword,
					"staging_storage_account_id": stagingStorageAccountID,
				})
			}
		}
	}

	return results
}

func flattenArmDataBoxJobContactDetails(input *databox.ContactDetails) []interface{} {
	results := make([]interface{}, 0)
	if input == nil {
		return results
	}

	contactName := ""
	if input.ContactName != nil {
		contactName = *input.ContactName
	}

	emails := []string{}
	if v := input.EmailList; v != nil {
		emails = *v
	}

	mobile := ""
	if input.Mobile != nil {
		mobile = *input.Mobile
	}

	phoneExtension := ""
	if input.PhoneExtension != nil {
		phoneExtension = *input.PhoneExtension
	}

	phoneNumber := ""
	if input.Phone != nil {
		phoneNumber = *input.Phone
	}

	results = append(results, map[string]interface{}{
		"name":                    contactName,
		"emails":                  utils.FlattenStringSlice(&emails),
		"mobile":                  mobile,
		"notification_preference": flattenArmDataBoxJobNotificationPreference(input.NotificationPreference),
		"phone_extension":         phoneExtension,
		"phone_number":            phoneNumber,
	})

	return results
}

func flattenArmDataBoxJobNotificationPreference(input *[]databox.NotificationPreference) []interface{} {
	results := make([]interface{}, 0)
	if len(*input) == 0 {
		return results
	}

	devicePrepared := false
	dispatched := false
	delivered := false
	pickedUp := false
	atAzureDC := false
	dataCopied := false
	for _, item := range *input {
		switch string(item.StageName) {
		case string(databox.DevicePrepared):
			devicePrepared = *item.SendNotification
		case string(databox.Dispatched):
			dispatched = *item.SendNotification
		case string(databox.Delivered):
			delivered = *item.SendNotification
		case string(databox.PickedUp):
			pickedUp = *item.SendNotification
		case string(databox.AtAzureDC):
			atAzureDC = *item.SendNotification
		case string(databox.DataCopy):
			dataCopied = *item.SendNotification
		}
	}

	results = append(results, map[string]interface{}{
		"device_prepared": devicePrepared,
		"dispatched":      dispatched,
		"delivered":       delivered,
		"picked_up":       pickedUp,
		"at_azure_dc":     atAzureDC,
		"data_copied":     dataCopied,
	})

	return results
}
