package azurerm

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2017-03-01-preview/sql"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/response"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmSqlServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSqlServerCreateUpdate,
		Read:   resourceArmSqlServerRead,
		Update: resourceArmSqlServerCreateUpdate,
		Delete: resourceArmSqlServerDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(60 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: azure.ValidateMsSqlServerName,
			},

			"location": azure.SchemaLocation(),

			"resource_group_name": azure.SchemaResourceGroupName(),

			"version": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"2.0",
					"12.0",
				}, true),
			},

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

			"fully_qualified_domain_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"identity": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								"SystemAssigned",
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

			"blob_auditing_policies": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"state": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"Enabled", "Disabled"}, false),
						},
						"storage_endpoint": {
							Type:     schema.TypeString,
							Required: true,
						},
						"storage_account_access_key": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
						"retention_days": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"audit_actions_and_groups": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"storage_account_subscription_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								v := val.(string)
								var _, err = uuid.FromString(v)
								if err != nil {
									errs = append(errs, fmt.Errorf("%q is not in correct format:%+v", key, err))
								}
								return
							},
						},
						"is_storage_secondary_key_in_use": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"tags": tags.Schema(),
		},
	}
}

func resourceArmSqlServerCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).Sql.ServersClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*ArmClient).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)
	location := azure.NormalizeLocation(d.Get("location").(string))
	adminUsername := d.Get("administrator_login").(string)
	version := d.Get("version").(string)

	t := d.Get("tags").(map[string]interface{})
	metadata := tags.Expand(t)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resGroup, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing SQL Server %q (Resource Group %q): %+v", name, resGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_sql_server", *existing.ID)
		}
	}

	parameters := sql.Server{
		Location: utils.String(location),
		Tags:     metadata,
		ServerProperties: &sql.ServerProperties{
			Version:            utils.String(version),
			AdministratorLogin: utils.String(adminUsername),
		},
	}

	if _, ok := d.GetOk("identity"); ok {
		sqlServerIdentity := expandAzureRmSqlServerIdentity(d)
		parameters.Identity = sqlServerIdentity
	}

	if d.HasChange("administrator_login_password") {
		adminPassword := d.Get("administrator_login_password").(string)
		parameters.ServerProperties.AdministratorLoginPassword = utils.String(adminPassword)
	}

	future, err := client.CreateOrUpdate(ctx, resGroup, name, parameters)
	if err != nil {
		return fmt.Errorf("Error issuing create/update request for SQL Server %q (Resource Group %q): %+v", name, resGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		if response.WasConflict(future.Response()) {
			return fmt.Errorf("SQL Server names need to be globally unique and %q is already in use.", name)
		}

		return fmt.Errorf("Error waiting on create/update future for SQL Server %q (Resource Group %q): %+v", name, resGroup, err)
	}

	resp, err := client.Get(ctx, resGroup, name)
	if err != nil {
		return fmt.Errorf("Error issuing get request for SQL Server %q (Resource Group %q): %+v", name, resGroup, err)
	}

	d.SetId(*resp.ID)

	if _, ok := d.GetOk("blob_auditing_policies"); ok {
		auditingClient := meta.(*ArmClient).Sql.ServerBlobAuditingPoliciesClient
		serverBlobAuditingPolicyProperties := expandAzureRmSqlServerBlobAuditingPolicies(d)
		auditingParameters := sql.ServerBlobAuditingPolicy{
			ServerBlobAuditingPolicyProperties: serverBlobAuditingPolicyProperties,
		}
		future, err := auditingClient.CreateOrUpdate(ctx, resGroup, name, auditingParameters)
		if err != nil {
			return fmt.Errorf("Error issuing create/update request for SQL Server %q Blob Auditing Policies(Resource Group %q): %+v", name, resGroup, err)
		}

		if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
			if response.WasConflict(future.Response()) {
				return fmt.Errorf("SQL Server names need to be globally unique and %q is already in use.", name)
			}

			return fmt.Errorf("Error waiting on create/update future for SQL Server %q Blob Auditing Policies (Resource Group %q): %+v", name, resGroup, err)
		}
	}

	return resourceArmSqlServerRead(d, meta)
}

func resourceArmSqlServerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).Sql.ServersClient
	ctx, cancel := timeouts.ForRead(meta.(*ArmClient).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resGroup := id.ResourceGroup
	name := id.Path["servers"]

	resp, err := client.Get(ctx, resGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] Error reading SQL Server %q - removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading SQL Server %s: %v", name, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resGroup)
	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}

	if err := d.Set("identity", flattenAzureRmSqlServerIdentity(resp.Identity)); err != nil {
		return fmt.Errorf("Error setting `identity`: %+v", err)
	}

	if serverProperties := resp.ServerProperties; serverProperties != nil {
		d.Set("version", serverProperties.Version)
		d.Set("administrator_login", serverProperties.AdministratorLogin)
		d.Set("fully_qualified_domain_name", serverProperties.FullyQualifiedDomainName)
	}

	auditingClient := meta.(*ArmClient).Sql.ServerBlobAuditingPoliciesClient
	auditingResp, err := auditingClient.Get(ctx, resGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] Error reading SQL Server %q Blob Auditing Policies - removing from state", d.Id())
		}

		return fmt.Errorf("Error reading SQL Server %s: %v Blob Auditing Policies", name, err)
	}

	d.Set("blob_auditing_policies", flattenAzureRmSqlServerBlobAuditingPolicies(&auditingResp))

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmSqlServerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).Sql.ServersClient
	ctx, cancel := timeouts.ForDelete(meta.(*ArmClient).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resGroup := id.ResourceGroup
	name := id.Path["servers"]

	future, err := client.Delete(ctx, resGroup, name)
	if err != nil {
		return fmt.Errorf("Error deleting SQL Server %s: %+v", name, err)
	}

	return future.WaitForCompletionRef(ctx, client.Client)
}

func expandAzureRmSqlServerIdentity(d *schema.ResourceData) *sql.ResourceIdentity {
	identities := d.Get("identity").([]interface{})
	if len(identities) == 0 {
		return &sql.ResourceIdentity{}
	}
	identity := identities[0].(map[string]interface{})
	identityType := sql.IdentityType(identity["type"].(string))
	return &sql.ResourceIdentity{
		Type: identityType,
	}
}
func flattenAzureRmSqlServerIdentity(identity *sql.ResourceIdentity) []interface{} {
	if identity == nil {
		return []interface{}{}
	}
	result := make(map[string]interface{})
	result["type"] = identity.Type
	if identity.PrincipalID != nil {
		result["principal_id"] = identity.PrincipalID.String()
	}
	if identity.TenantID != nil {
		result["tenant_id"] = identity.TenantID.String()
	}

	return []interface{}{result}
}
func expandAzureRmSqlServerBlobAuditingPolicies(d *schema.ResourceData) *sql.ServerBlobAuditingPolicyProperties {
	serverBlobAuditingPoliciesList := d.Get("blob_auditing_policies").([]interface{})
	if len(serverBlobAuditingPoliciesList) == 0 {
		return &sql.ServerBlobAuditingPolicyProperties{}
	}
	serverBlobAuditingPolicies := serverBlobAuditingPoliciesList[0].(map[string]interface{})
	state := sql.BlobAuditingPolicyState(serverBlobAuditingPolicies["state"].(string))
	storageEndpoint := serverBlobAuditingPolicies["storage_endpoint"].(string)
	storageAccountAccessKey := serverBlobAuditingPolicies["storage_account_access_key"].(string)

	ServerBlobAuditingPolicyProperties := sql.ServerBlobAuditingPolicyProperties{
		State:                   state,
		StorageEndpoint:         &storageEndpoint,
		StorageAccountAccessKey: &storageAccountAccessKey,
		AuditActionsAndGroups  : nil,
	}
	//retention_days
	if retentionDays, ok := serverBlobAuditingPolicies["retention_days"]; ok {
		retentionDays := int32(retentionDays.(int))
		ServerBlobAuditingPolicyProperties.RetentionDays = &retentionDays
	}
	//audit_actions_and_groups
	if auditActionsAndGroups, ok := serverBlobAuditingPolicies["audit_actions_and_groups"] ; ok && auditActionsAndGroups!=""{
		auditActionsAndGroups := strings.Split(auditActionsAndGroups.(string), ",")
		ServerBlobAuditingPolicyProperties.AuditActionsAndGroups = &auditActionsAndGroups
	}
	//storage_account_subscription_id
	if storageAccountSubscriptionID, ok := serverBlobAuditingPolicies["storage_account_subscription_id"]; ok && storageAccountSubscriptionID!=""{
		storageAccountSubscriptionID, _ := uuid.FromString(storageAccountSubscriptionID.(string))
		ServerBlobAuditingPolicyProperties.StorageAccountSubscriptionID = &storageAccountSubscriptionID
	}
	//is_storage_secondary_key_in_use
	if isStorageSecondaryKeyInUse, ok := serverBlobAuditingPolicies["is_storage_secondary_key_in_use"]; ok {
		isStorageSecondaryKeyInUse := isStorageSecondaryKeyInUse.(bool)
		ServerBlobAuditingPolicyProperties.IsStorageSecondaryKeyInUse = &isStorageSecondaryKeyInUse
	}
	return &ServerBlobAuditingPolicyProperties
}
func flattenAzureRmSqlServerBlobAuditingPolicies(serverBlobAuditingPolicy *sql.ServerBlobAuditingPolicy) []interface{} {
	if serverBlobAuditingPolicy == nil {
		return []interface{}{}
	}
	result := make(map[string]interface{})

	result["state"] = serverBlobAuditingPolicy.State
	result["is_storage_secondary_key_in_use"] = serverBlobAuditingPolicy.IsStorageSecondaryKeyInUse
	if auditActionsAndGroups := serverBlobAuditingPolicy.AuditActionsAndGroups; auditActionsAndGroups != nil {
		result["audit_actions_and_groups"] = strings.Join(*auditActionsAndGroups, ",")
	}
	if RetentionDays := serverBlobAuditingPolicy.RetentionDays; RetentionDays != nil {
		result["retention_days"] = RetentionDays
	}
	if StorageAccountSubscriptionID := serverBlobAuditingPolicy.StorageAccountSubscriptionID; StorageAccountSubscriptionID != nil {
		result["storage_account_subscription_id"] = StorageAccountSubscriptionID.String()
	}
	if StorageEndpoint := serverBlobAuditingPolicy.StorageEndpoint; StorageEndpoint != nil {
		result["storage_endpoint"] = StorageEndpoint
	}

	return []interface{}{result}
}
