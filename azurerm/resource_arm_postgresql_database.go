package azurerm

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/arm/postgresql"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmPostgreSQLDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmPostgreSQLDatabaseCreate,
		Read:   resourceArmPostgreSQLDatabaseRead,
		Delete: resourceArmPostgreSQLDatabaseDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": resourceGroupNameSchema(),

			"server_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"charset": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"collation": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceArmPostgreSQLDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).postgresqlDatabasesClient

	log.Printf("[INFO] preparing arguments for AzureRM PostgreSQL Database creation.")

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)
	serverName := d.Get("server_name").(string)

	charset := d.Get("charset").(string)
	collation := d.Get("collation").(string)

	properties := postgresql.Database{
		DatabaseProperties: &postgresql.DatabaseProperties{
			Charset:   utils.String(charset),
			Collation: utils.String(collation),
		},
	}

	_, error := client.CreateOrUpdate(resGroup, serverName, name, properties, make(chan struct{}))
	err := <-error
	if err != nil {
		return err
	}

	read, err := client.Get(resGroup, serverName, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read PostgreSQL Database %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmPostgreSQLDatabaseRead(d, meta)
}

func resourceArmPostgreSQLDatabaseRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).postgresqlDatabasesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	serverName := id.Path["servers"]
	name := id.Path["databases"]

	resp, err := client.Get(resGroup, serverName, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[WARN] PostgreSQL Database '%s' was not found (resource group '%s')", name, resGroup)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error making Read request on Azure PostgreSQL Database %s: %+v", name, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("server_name", serverName)
	d.Set("charset", resp.Charset)
	d.Set("collation", resp.Collation)

	return nil
}

func resourceArmPostgreSQLDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).postgresqlDatabasesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	serverName := id.Path["servers"]
	name := id.Path["databases"]

	_, error := client.Delete(resGroup, serverName, name, make(chan struct{}))
	err = <-error

	return err
}
