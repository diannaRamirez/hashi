package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/appconfiguration/2023-03-01/configurationstores"
)

var (
	ConfigurationStoreCache = map[string]ConfigurationStoreDetails{}
	keysmith                = &sync.RWMutex{}
	lock                    = map[string]*sync.RWMutex{}
)

type ConfigurationStoreDetails struct {
	configurationStoreId string
	dataPlaneEndpoint    string
}

func (c Client) AddToCache(configurationStoreId configurationstores.ConfigurationStoreId, dataPlaneEndpoint string) {
	cacheKey := c.cacheKeyForConfigurationStore(configurationStoreId.ConfigurationStoreName)
	keysmith.Lock()
	ConfigurationStoreCache[cacheKey] = ConfigurationStoreDetails{
		configurationStoreId: configurationStoreId.ID(),
		dataPlaneEndpoint:    dataPlaneEndpoint,
	}
	keysmith.Unlock()
}

func (c Client) ConfigurationStoreIDFromEndpoint(ctx context.Context, configurationStoresClient *configurationstores.ConfigurationStoresClient, configurationStoreEndpoint string, subscriptionId string) (*string, error) {
	configurationStoreName, err := c.parseNameFromEndpoint(configurationStoreEndpoint)
	if err != nil {
		return nil, err
	}

	cacheKey := c.cacheKeyForConfigurationStore(*configurationStoreName)
	keysmith.Lock()
	if lock[cacheKey] == nil {
		lock[cacheKey] = &sync.RWMutex{}
	}
	keysmith.Unlock()
	lock[cacheKey].Lock()
	defer lock[cacheKey].Unlock()

	if v, ok := ConfigurationStoreCache[cacheKey]; ok {
		return &v.configurationStoreId, nil
	}

	subscriptionIdStruct := commonids.NewSubscriptionID(subscriptionId)
	predicate := configurationstores.ConfigurationStoreOperationPredicate{
		Name: configurationStoreName,
	}
	result, err := configurationStoresClient.ListCompleteMatchingPredicate(ctx, subscriptionIdStruct, predicate)
	if err != nil {
		return nil, fmt.Errorf("listing Configuration Stores: %+v", err)
	}

	if len(result.Items) != 0 {
		configurationStoreId, err := configurationstores.ParseConfigurationStoreID(*result.Items[0].Id)
		if err != nil {
			return nil, fmt.Errorf("parsing Configuration Store ID: %+v", err)
		}
		c.AddToCache(*configurationStoreId, configurationStoreEndpoint)

		return pointer.To(configurationStoreId.ID()), nil
	}

	// we haven't found it, but Data Sources and Resources need to handle this error separately
	return nil, nil
}

func (c Client) EndpointForConfigurationStore(ctx context.Context, configurationStoreId configurationstores.ConfigurationStoreId) (*string, error) {
	cacheKey := c.cacheKeyForConfigurationStore(configurationStoreId.ConfigurationStoreName)
	keysmith.Lock()
	if lock[cacheKey] == nil {
		lock[cacheKey] = &sync.RWMutex{}
	}
	keysmith.Unlock()
	lock[cacheKey].Lock()
	defer lock[cacheKey].Unlock()

	if v, ok := ConfigurationStoreCache[cacheKey]; ok {
		return &v.dataPlaneEndpoint, nil
	}

	resp, err := c.ConfigurationStoresClient.Get(ctx, configurationStoreId)
	if err != nil {
		return nil, fmt.Errorf("retrieving %s:%+v", configurationStoreId, err)
	}

	if resp.Model == nil || resp.Model.Properties == nil || resp.Model.Properties.Endpoint == nil {
		return nil, fmt.Errorf("retrieving %s: `model.properties.Endpoint` was nil", configurationStoreId)
	}

	c.AddToCache(configurationStoreId, *resp.Model.Properties.Endpoint)

	return resp.Model.Properties.Endpoint, nil
}

func (c Client) Exists(ctx context.Context, configurationStoreId configurationstores.ConfigurationStoreId) (bool, error) {
	cacheKey := c.cacheKeyForConfigurationStore(configurationStoreId.ConfigurationStoreName)
	keysmith.Lock()
	if lock[cacheKey] == nil {
		lock[cacheKey] = &sync.RWMutex{}
	}
	keysmith.Unlock()
	lock[cacheKey].Lock()
	defer lock[cacheKey].Unlock()

	if _, ok := ConfigurationStoreCache[cacheKey]; ok {
		return true, nil
	}

	resp, err := c.ConfigurationStoresClient.Get(ctx, configurationStoreId)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			return false, nil
		}
		return false, fmt.Errorf("retrieving %s: %+v", configurationStoreId, err)
	}

	if resp.Model == nil || resp.Model.Properties == nil || resp.Model.Properties.Endpoint == nil {
		return false, fmt.Errorf("retrieving %s: `model.properties.Endpoint` was nil", configurationStoreId)
	}

	c.AddToCache(configurationStoreId, *resp.Model.Properties.Endpoint)

	return true, nil
}

func (c Client) RemoveFromCache(configurationStoreId configurationstores.ConfigurationStoreId) {
	cacheKey := c.cacheKeyForConfigurationStore(configurationStoreId.ConfigurationStoreName)
	keysmith.Lock()
	if lock[cacheKey] == nil {
		lock[cacheKey] = &sync.RWMutex{}
	}
	keysmith.Unlock()
	lock[cacheKey].Lock()
	delete(ConfigurationStoreCache, cacheKey)
	lock[cacheKey].Unlock()
}

func (c Client) cacheKeyForConfigurationStore(name string) string {
	return strings.ToLower(name)
}

func (c Client) parseNameFromEndpoint(input string) (*string, error) {
	uri, err := url.ParseRequestURI(input)
	if err != nil {
		return nil, err
	}

	// https://the-appconfiguration.azconfig.io

	segments := strings.Split(uri.Host, ".")
	if len(segments) < 3 || segments[1] != "azconfig" || segments[2] != "io" {
		return nil, fmt.Errorf("expected a URI in the format `https://the-appconfiguration.azconfig.io` but got %q", uri.Host)
	}
	return &segments[0], nil
}
