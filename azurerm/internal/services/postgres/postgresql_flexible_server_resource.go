package postgres

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/postgresql/mgmt/2020-02-14-preview/postgresqlflexibleservers"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/location"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/postgres/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/postgres/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	azSchema "github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmPostgresqlFlexibleServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmPostgresqlFlexibleServerCreate,
		Read:   resourceArmPostgresqlFlexibleServerRead,
		Update: resourceArmPostgresqlFlexibleServerUpdate,
		Delete: resourceArmPostgresqlFlexibleServerDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Hour),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Hour),
			Delete: schema.DefaultTimeout(3 * time.Hour),
		},

		Importer: azSchema.ValidateResourceIDPriorToImport(func(id string) error {
			_, err := parse.FlexibleServerID(id)
			return err
		}),

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.FlexibleServerName(),
			},

			"resource_group_name": azure.SchemaResourceGroupName(),

			"location": azure.SchemaLocation(),

			"administrator_login": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"administrator_login_password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},

			"sku": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validate.FlexibleServerSkuName(),
						},

						"tier": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(postgresqlflexibleservers.Burstable),
								string(postgresqlflexibleservers.GeneralPurpose),
								string(postgresqlflexibleservers.MemoryOptimized),
							}, false),
						},
					},
				},
			},

			"version": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(postgresqlflexibleservers.OneOne),
					string(postgresqlflexibleservers.OneTwo),
				}, false),
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"1",
					"2",
					"3",
				}, false),
			},

			"create_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  string(postgresqlflexibleservers.Default),
				ValidateFunc: validation.StringInSlice([]string{
					string(postgresqlflexibleservers.Default),
					string(postgresqlflexibleservers.PointInTimeRestore),
				}, false),
			},

			"delegated_subnet_resource_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: azure.ValidateResourceID,
			},

			"display_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"identity": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(postgresqlflexibleservers.SystemAssigned),
							}, false),
						},

						"principal_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"tenant_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"point_in_time_utc": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsRFC3339Time,
			},

			"source_server_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ha_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"maintenance_window": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"day_of_week": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validation.IntBetween(0, 6),
						},

						"start_hour": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validation.IntBetween(0, 23),
						},

						"start_minute": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validation.IntBetween(0, 59),
						},
					},
				},
			},

			"backup_retention_days": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      7,
				ValidateFunc: validation.IntBetween(7, 35),
			},

			"storage_mb": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      32768,
				ValidateFunc: validation.IntInSlice([]int{32768, 65536, 131072, 262144, 524288, 1048576, 2097152, 4194304, 8388608, 16777216, 33554432}),
			},

			"byok_enforcement": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"fqdn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ha_state": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"public_network_access": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"standby_availability_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"properties_tags": tags.Schema(),

			"tags": tags.Schema(),
		},
	}
}
func resourceArmPostgresqlFlexibleServerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Postgres.FlexibleServersClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	existing, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		if !utils.ResponseWasNotFound(existing.Response) {
			return fmt.Errorf("checking for present of existing Postgresqlflexibleservers Server %q (Resource Group %q): %+v", name, resourceGroup, err)
		}
	}
	if existing.ID != nil && *existing.ID != "" {
		return tf.ImportAsExistsError("azurerm_postgresql_flexible_server", *existing.ID)
	}

	if v, ok := d.GetOk("create_mode"); ok && postgresqlflexibleservers.CreateMode(v.(string)) == postgresqlflexibleservers.PointInTimeRestore {
		if _, ok := d.GetOk("source_server_name"); !ok {
			return fmt.Errorf("`source_server_name` is required when `create_mode` is `PointInTimeRestore`")
		}
		if _, ok := d.GetOk("point_in_time_utc"); !ok {
			return fmt.Errorf("`point_in_time_utc` is required when `create_mode` is `PointInTimeRestore`")
		}
	}

	haEnabled := postgresqlflexibleservers.Disabled
	if d.Get("ha_enabled").(bool) {
		haEnabled = postgresqlflexibleservers.Enabled
	}

	parameters := postgresqlflexibleservers.Server{
		Location: utils.String(location.Normalize(d.Get("location").(string))),
		Identity: expandArmServerIdentity(d.Get("identity").([]interface{})),
		ServerProperties: &postgresqlflexibleservers.ServerProperties{
			AdministratorLogin:         utils.String(d.Get("administrator_login").(string)),
			AvailabilityZone:           utils.String(d.Get("availability_zone").(string)),
			CreateMode:                 postgresqlflexibleservers.CreateMode(d.Get("create_mode").(string)),
			DelegatedSubnetArguments:   expandArmServerServerPropertiesDelegatedSubnetArguments(d.Get("delegated_subnet_resource_id").(string)),
			DisplayName:                utils.String(d.Get("display_name").(string)),
			SourceServerName:           utils.String(d.Get("source_server_name").(string)),
			Version:                    postgresqlflexibleservers.ServerVersion(d.Get("version").(string)),
			AdministratorLoginPassword: utils.String(d.Get("administrator_login_password").(string)),
			HaEnabled:                  haEnabled,
			MaintenanceWindow:          expandArmServerMaintenanceWindow(d.Get("maintenance_window").([]interface{})),
			StorageProfile:             expandArmServerStorageProfile(d),
			Tags:                       tags.Expand(d.Get("properties_tags").(map[string]interface{})),
		},
		Sku:  expandArmServerSku(d.Get("sku").([]interface{})),
		Tags: tags.Expand(d.Get("tags").(map[string]interface{})),
	}

	pointInTimeUTC := d.Get("point_in_time_utc").(string)
	if pointInTimeUTC != "" {
		v, err := time.Parse(time.RFC3339, pointInTimeUTC)
		if err != nil {
			return fmt.Errorf("unable to parse `point_in_time_utc` value")
		}
		parameters.ServerProperties.PointInTimeUTC = &date.Time{Time: v}
	}

	future, err := client.Create(ctx, resourceGroup, name, parameters)
	if err != nil {
		return fmt.Errorf("creating Postgresqlflexibleservers Server %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting on creating future for Postgresqlflexibleservers Server %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	resp, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		return fmt.Errorf("retrieving Postgresqlflexibleservers Server %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if resp.ID == nil || *resp.ID == "" {
		return fmt.Errorf("empty or nil ID returned for Postgresqlflexibleservers Server %q (Resource Group %q) ID", name, resourceGroup)
	}

	d.SetId(*resp.ID)
	return resourceArmPostgresqlFlexibleServerRead(d, meta)
}

func resourceArmPostgresqlFlexibleServerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Postgres.FlexibleServersClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.FlexibleServerID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.Name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] postgresqlflexibleservers %q does not exist - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("retrieving Postgresqlflexibleservers Server %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	name := id.Name
	resourceGroup := id.ResourceGroup

	d.Set("name", name)
	d.Set("resource_group_name", resourceGroup)
	d.Set("location", location.NormalizeNilable(resp.Location))
	if err := d.Set("identity", flattenArmServerIdentity(resp.Identity)); err != nil {
		return fmt.Errorf("setting `identity`: %+v", err)
	}
	if props := resp.ServerProperties; props != nil {
		d.Set("administrator_login", props.AdministratorLogin)

		// sensitive prop not returned by API, pull it from config and write to state.
		adminPassword := d.Get("administrator_login_password").(string)
		d.Set("administrator_login_password", adminPassword)
		d.Set("availability_zone", props.AvailabilityZone)

		// CreateMode currently isn't returned by the API
		if props.CreateMode == "" {
			d.Set("create_mode", string(postgresqlflexibleservers.Default))
		} else {
			d.Set("create_mode", props.CreateMode)
		}

		if props.DelegatedSubnetArguments != nil && props.DelegatedSubnetArguments.SubnetArmResourceID != nil {
			d.Set("delegated_subnet_resource_id", props.DelegatedSubnetArguments.SubnetArmResourceID)
		}

		d.Set("display_name", props.DisplayName)
		d.Set("ha_enabled", props.HaEnabled == postgresqlflexibleservers.Enabled)

		if err := d.Set("maintenance_window", flattenArmServerMaintenanceWindow(props.MaintenanceWindow)); err != nil {
			return fmt.Errorf("setting `maintenance_window`: %+v", err)
		}

		if props.PointInTimeUTC != nil {
			d.Set("point_in_time_utc", props.PointInTimeUTC.Format(time.RFC3339))
		}

		if props.SourceServerName != nil {
			d.Set("source_server_name", props.SourceServerName)
		}

		if storage := props.StorageProfile; storage != nil {
			if storage.StorageMB != nil {
				d.Set("storage_mb", storage.StorageMB)
			}

			if storage.BackupRetentionDays != nil {
				d.Set("backup_retention_days", storage.BackupRetentionDays)
			}
		}

		d.Set("version", props.Version)

		// computed
		d.Set("byok_enforcement", string(*props.ByokEnforcement))
		d.Set("fqdn", string(*props.FullyQualifiedDomainName))
		d.Set("public_network_access", string(props.PublicNetworkAccess))
		d.Set("ha_state", string(props.HaState))

		if props.StandbyAvailabilityZone != nil {
			d.Set("standby_availability_zone", string(*props.StandbyAvailabilityZone))
		}
	}

	if err := d.Set("sku", flattenArmServerSku(resp.Sku)); err != nil {
		return fmt.Errorf("setting `sku`: %+v", err)
	}

	if resp.ServerProperties.Tags != nil && len(resp.ServerProperties.Tags) > 0 {
		if err := parse.FlattenAndSetPropertyTags(d, resp.ServerProperties.Tags); err != nil {
			return fmt.Errorf("setting `properties_tags`: %+v", err)
		}
	}

	if resp.Tags != nil && len(resp.Tags) > 0 {
		tags.FlattenAndSet(d, resp.Tags)
	}

	return nil
}

func resourceArmPostgresqlFlexibleServerUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Postgres.FlexibleServersClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.FlexibleServerID(d.Id())
	if err != nil {
		return err
	}

	parameters := postgresqlflexibleservers.ServerForUpdate{
		Location:                  utils.String(location.Normalize(d.Get("location").(string))),
		ServerPropertiesForUpdate: &postgresqlflexibleservers.ServerPropertiesForUpdate{},
	}
	if d.HasChange("administrator_login_password") {
		parameters.ServerPropertiesForUpdate.AdministratorLoginPassword = utils.String(d.Get("administrator_login_password").(string))
	}
	if d.HasChange("backup_retention_days") || d.HasChange("storage_mb") {
		parameters.ServerPropertiesForUpdate.StorageProfile = expandArmServerStorageProfile(d)
	}
	if d.HasChange("ha_enabled") {
		haEnabled := postgresqlflexibleservers.Disabled
		if d.Get("ha_enabled").(bool) {
			haEnabled = postgresqlflexibleservers.Enabled
		}
		parameters.ServerPropertiesForUpdate.HaEnabled = haEnabled
	}
	if d.HasChange("maintenance_window") {
		parameters.ServerPropertiesForUpdate.MaintenanceWindow = expandArmServerMaintenanceWindow(d.Get("maintenance_window").([]interface{}))
	}
	if d.HasChange("sku") {
		parameters.Sku = expandArmServerSku(d.Get("sku").([]interface{}))
	}
	if d.HasChange("tags") {
		parameters.Tags = tags.Expand(d.Get("tags").(map[string]interface{}))
	}

	future, err := client.Update(ctx, id.ResourceGroup, id.Name, parameters)
	if err != nil {
		return fmt.Errorf("updating Postgresqlflexibleservers Server %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting on updating future for Postgresqlflexibleservers Server %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}
	return resourceArmPostgresqlFlexibleServerRead(d, meta)
}

func resourceArmPostgresqlFlexibleServerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Postgres.FlexibleServersClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.FlexibleServerID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.Delete(ctx, id.ResourceGroup, id.Name)
	if err != nil {
		return fmt.Errorf("deleting Postgresqlflexibleservers Server %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting on deleting future for Postgresqlflexibleservers Server %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}
	return nil
}

func expandArmServerIdentity(input []interface{}) *postgresqlflexibleservers.Identity {
	if len(input) == 0 {
		return nil
	}
	v := input[0].(map[string]interface{})
	return &postgresqlflexibleservers.Identity{
		Type: postgresqlflexibleservers.ResourceIdentityType(v["type"].(string)),
	}
}

func expandArmServerServerPropertiesDelegatedSubnetArguments(input string) *postgresqlflexibleservers.ServerPropertiesDelegatedSubnetArguments {
	if len(input) == 0 {
		return nil
	}

	return &postgresqlflexibleservers.ServerPropertiesDelegatedSubnetArguments{
		SubnetArmResourceID: utils.String(input),
	}
}

func expandArmServerMaintenanceWindow(input []interface{}) *postgresqlflexibleservers.MaintenanceWindow {
	if len(input) == 0 {
		return nil
	}
	v := input[0].(map[string]interface{})

	maintenanceWindow := postgresqlflexibleservers.MaintenanceWindow{
		StartHour:   utils.Int32(int32(v["start_hour"].(int))),
		StartMinute: utils.Int32(int32(v["start_minute"].(int))),
		DayOfWeek:   utils.Int32(int32(v["day_of_week"].(int))),
	}

	maintenanceWindow.CustomWindow = utils.String(string(postgresqlflexibleservers.Disabled))
	if v["enabled"].(bool) {
		maintenanceWindow.CustomWindow = utils.String(string(postgresqlflexibleservers.Enabled))
	}

	return &maintenanceWindow
}

func expandArmServerStorageProfile(d *schema.ResourceData) *postgresqlflexibleservers.StorageProfile {
	storage := postgresqlflexibleservers.StorageProfile{}

	if v, ok := d.GetOk("backup_retention_days"); ok {
		storage.BackupRetentionDays = utils.Int32(int32(v.(int)))
	}

	if v, ok := d.GetOk("storage_mb"); ok {
		storage.StorageMB = utils.Int32(int32(v.(int)))
	}

	return &storage
}

func expandArmServerSku(input []interface{}) *postgresqlflexibleservers.Sku {
	if len(input) == 0 {
		return nil
	}
	v := input[0].(map[string]interface{})
	return &postgresqlflexibleservers.Sku{
		Name: utils.String(v["name"].(string)),
		Tier: postgresqlflexibleservers.SkuTier(v["tier"].(string)),
	}
}

func flattenArmServerIdentity(input *postgresqlflexibleservers.Identity) []interface{} {
	if input == nil {
		return make([]interface{}, 0)
	}

	var t postgresqlflexibleservers.ResourceIdentityType
	if input.Type != "" {
		t = input.Type
	}
	var principalId string
	if input.PrincipalID != nil {
		principalId = *input.PrincipalID
	}
	var tenantId string
	if input.TenantID != nil {
		tenantId = *input.TenantID
	}
	return []interface{}{
		map[string]interface{}{
			"type":         t,
			"principal_id": principalId,
			"tenant_id":    tenantId,
		},
	}
}

func flattenArmServerServerPropertiesDelegatedSubnetArguments(input *postgresqlflexibleservers.ServerPropertiesDelegatedSubnetArguments) []interface{} {
	if input == nil {
		return make([]interface{}, 0)
	}

	var subnetArmResourceId string
	if input.SubnetArmResourceID != nil {
		subnetArmResourceId = *input.SubnetArmResourceID
	}
	return []interface{}{
		map[string]interface{}{
			"subnet_arm_resource_id": subnetArmResourceId,
		},
	}
}

func flattenArmServerMaintenanceWindow(input *postgresqlflexibleservers.MaintenanceWindow) []interface{} {
	if input == nil {
		return make([]interface{}, 0)
	}

	var enabled bool
	if input.CustomWindow != nil {
		enabled = (*input.CustomWindow == string(postgresqlflexibleservers.Enabled))
	}

	var dayOfWeek int32
	if input.DayOfWeek != nil {
		dayOfWeek = *input.DayOfWeek
	}
	var startHour int32
	if input.StartHour != nil {
		startHour = *input.StartHour
	}
	var startMinute int32
	if input.StartMinute != nil {
		startMinute = *input.StartMinute
	}
	return []interface{}{
		map[string]interface{}{
			"enabled":      enabled,
			"day_of_week":  dayOfWeek,
			"start_hour":   startHour,
			"start_minute": startMinute,
		},
	}
}

func flattenArmServerSku(input *postgresqlflexibleservers.Sku) []interface{} {
	if input == nil {
		return make([]interface{}, 0)
	}

	var name string
	if input.Name != nil {
		name = *input.Name
	}
	var tier postgresqlflexibleservers.SkuTier
	if input.Tier != "" {
		tier = input.Tier
	}
	return []interface{}{
		map[string]interface{}{
			"name": name,
			"tier": tier,
		},
	}
}
