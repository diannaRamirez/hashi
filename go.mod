module github.com/terraform-providers/terraform-provider-azurerm

require (
	github.com/Azure/azure-sdk-for-go v44.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.10.0
	github.com/Azure/go-autorest/autorest/azure/cli v0.3.1 // indirect
	github.com/Azure/go-autorest/autorest/date v0.2.0
	github.com/btubbs/datetime v0.1.0
	github.com/davecgh/go-spew v1.1.1
	github.com/google/uuid v1.1.1
	github.com/hashicorp/go-azure-helpers v0.11.1
	github.com/hashicorp/go-getter v1.4.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/go-uuid v1.0.1
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.13.1
	github.com/rickb777/date v1.12.5-0.20200422084442-6300e543c4d9
	github.com/satori/go.uuid v1.2.0
	github.com/satori/uuid v0.0.0-20160927100844-b061729afc07
	github.com/sergi/go-diff v1.1.0
	github.com/terraform-providers/terraform-provider-azuread v0.9.0
	github.com/tombuildsstuff/giovanni v0.11.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20200625001655-4c5254603344
	golang.org/x/tools v0.0.0-20200708183856-df98bc6d456c // indirect
	gopkg.in/yaml.v2 v2.2.4
)

replace github.com/Azure/go-autorest => github.com/tombuildsstuff/go-autorest v14.0.1-0.20200416184303-d4e299a3c04a+incompatible

replace github.com/Azure/go-autorest/autorest => github.com/tombuildsstuff/go-autorest/autorest v0.10.1-0.20200416184303-d4e299a3c04a

replace github.com/Azure/go-autorest/autorest/azure/auth => github.com/tombuildsstuff/go-autorest/autorest/azure/auth v0.4.3-0.20200416184303-d4e299a3c04a

go 1.13
