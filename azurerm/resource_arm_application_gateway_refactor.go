package azurerm

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func expandApplicationGatewayWafConfig(d *schema.ResourceData) *network.ApplicationGatewayWebApplicationFirewallConfiguration {
	wafSet := d.Get("waf_configuration").(*schema.Set).List()
	waf := wafSet[0].(map[string]interface{})

	enabled := waf["enabled"].(bool)
	mode := waf["firewall_mode"].(string)
	rulesettype := waf["rule_set_type"].(string)
	rulesetversion := waf["rule_set_version"].(string)

	return &network.ApplicationGatewayWebApplicationFirewallConfiguration{
		Enabled:        &enabled,
		FirewallMode:   network.ApplicationGatewayFirewallMode(mode),
		RuleSetType:    &rulesettype,
		RuleSetVersion: &rulesetversion,
	}
}

func expandApplicationGatewayURLPathMaps(d *schema.ResourceData, gatewayID string) *[]network.ApplicationGatewayURLPathMap {
	configs := d.Get("url_path_map").([]interface{})
	pathMaps := make([]network.ApplicationGatewayURLPathMap, 0)

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

func flattenApplicationGatewayWafConfig(waf *network.ApplicationGatewayWebApplicationFirewallConfiguration) []interface{} {
	result := make(map[string]interface{})

	result["enabled"] = *waf.Enabled
	result["firewall_mode"] = string(waf.FirewallMode)
	result["rule_set_type"] = waf.RuleSetType
	result["rule_set_version"] = waf.RuleSetVersion

	return []interface{}{result}
}

func flattenApplicationGatewayURLPathMaps(input *[]network.ApplicationGatewayURLPathMap) ([]interface{}, error) {
	result := make([]interface{}, 0)

	if pathMaps := input; pathMaps != nil {
		for _, config := range *pathMaps {
			pathMap := map[string]interface{}{
				"id":   *config.ID,
				"name": *config.Name,
			}

			if props := config.ApplicationGatewayURLPathMapPropertiesFormat; props != nil {
				if backendPool := props.DefaultBackendAddressPool; backendPool != nil {
					backendAddressPoolName := strings.Split(*backendPool.ID, "/")[len(strings.Split(*backendPool.ID, "/"))-1]
					pathMap["default_backend_address_pool_name"] = backendAddressPoolName
					pathMap["default_backend_address_pool_id"] = *backendPool.ID
				}

				if settings := props.DefaultBackendHTTPSettings; settings != nil {
					backendHTTPSettingsName := strings.Split(*settings.ID, "/")[len(strings.Split(*settings.ID, "/"))-1]
					pathMap["default_backend_http_settings_name"] = backendHTTPSettingsName
					pathMap["default_backend_http_settings_id"] = *settings.ID
				}

				pathRules := make([]interface{}, 0)
				if rules := props.PathRules; rules != nil {
					for _, pathRuleConfig := range *rules {
						rule := map[string]interface{}{
							"id":   *pathRuleConfig.ID,
							"name": *pathRuleConfig.Name,
						}

						if ruleProps := pathRuleConfig.ApplicationGatewayPathRulePropertiesFormat; props != nil {
							if pool := ruleProps.BackendAddressPool; pool != nil {
								backendAddressPoolName2 := strings.Split(*pool.ID, "/")[len(strings.Split(*pool.ID, "/"))-1]
								rule["backend_address_pool_name"] = backendAddressPoolName2
								rule["backend_address_pool_id"] = *pool.ID
							}

							if backend := ruleProps.BackendHTTPSettings; backend != nil {
								backendHTTPSettingsName2 := strings.Split(*backend.ID, "/")[len(strings.Split(*backend.ID, "/"))-1]
								rule["backend_http_settings_name"] = backendHTTPSettingsName2
								rule["backend_http_settings_id"] = *backend.ID
							}

							pathOutputs := make([]interface{}, 0)
							if paths := ruleProps.Paths; paths != nil {
								for _, rulePath := range *paths {
									pathOutputs = append(pathOutputs, rulePath)
								}
							}
							rule["paths"] = pathOutputs
						}

						pathRules = append(pathRules, rule)
					}
					pathMap["path_rule"] = pathRules
				}
			}

			result = append(result, pathMap)
		}
	}

	return result, nil
}

// TODO: can this be removed?
func hashApplicationGatewayWafConfig(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%t-", m["enabled"].(bool)))
	buf.WriteString(fmt.Sprintf("%s-", m["firewall_mode"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", *m["rule_set_type"].(*string)))
	buf.WriteString(fmt.Sprintf("%s-", *m["rule_set_version"].(*string)))

	return hashcode.String(buf.String())
}
