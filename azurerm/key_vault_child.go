package azurerm

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

func parseKeyVaultChildID(id string) (*KeyVaultChildID, error) {
	// example: https://tharvey-keyvault.vault.azure.net/type/bird/fdf067c93bbb4b22bff4d8b7a9a56217
	idURL, err := url.ParseRequestURI(id)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse Azure KeyVault Child Id: %s", err)
	}

	path := idURL.Path

	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	components := strings.Split(path, "/")

	if len(components) != 3 {
		return nil, fmt.Errorf("Azure KeyVault Child Id should have 3 segments, got %d: '%s'", len(components), path)
	}

	childId := KeyVaultChildID{
		KeyVaultBaseUrl: fmt.Sprintf("%s://%s/", idURL.Scheme, idURL.Host),
		Name:            components[1],
		Version:         components[2],
	}

	return &childId, nil
}

type KeyVaultChildID struct {
	KeyVaultBaseUrl string
	Name            string
	Version         string
}

func validateKeyVaultChildName(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)

	if matched := regexp.MustCompile(`^[0-9a-zA-Z-]+$`).Match([]byte(value)); !matched {
		es = append(es, fmt.Errorf("%q may only contain alphanumeric characters and dashes", k))
	}

	return
}
