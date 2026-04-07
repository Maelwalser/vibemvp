package provider

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const credConfigFile = "providers.json"

// providerConfig is the persisted record for one provider.
type providerConfig struct {
	Auth       string `json:"auth"`
	Credential string `json:"credential"`
}

// credConfigPath returns the path to the provider config file.
func credConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vibemenu", credConfigFile), nil
}

// loadProviderConfigs reads the config file, returning an empty map on first run.
func loadProviderConfigs() (map[string]providerConfig, error) {
	path, err := credConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]providerConfig), nil
	}
	if err != nil {
		return nil, err
	}
	var configs map[string]providerConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, err
	}
	return configs, nil
}

// saveProviderConfigs writes the config map to disk with 0600 permissions.
func saveProviderConfigs(configs map[string]providerConfig) error {
	path, err := credConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// SaveProviderCredential persists the auth method and credential for a provider.
func SaveProviderCredential(provider, auth, credential string) error {
	configs, err := loadProviderConfigs()
	if err != nil {
		configs = make(map[string]providerConfig)
	}
	newConfigs := make(map[string]providerConfig, len(configs)+1)
	for k, v := range configs {
		newConfigs[k] = v
	}
	newConfigs[provider] = providerConfig{Auth: auth, Credential: credential}
	return saveProviderConfigs(newConfigs)
}

// LoadAllProviderCredentials returns all previously saved provider selections.
func LoadAllProviderCredentials() map[string]ProviderSelection {
	configs, err := loadProviderConfigs()
	if err != nil || len(configs) == 0 {
		return nil
	}
	result := make(map[string]ProviderSelection, len(configs))
	for prov, cfg := range configs {
		if cfg.Credential == "" {
			continue
		}
		result[prov] = ProviderSelection{
			Provider:   prov,
			Auth:       cfg.Auth,
			Credential: cfg.Credential,
		}
	}
	return result
}

// DeleteProviderCredential removes all stored data for the given provider.
func DeleteProviderCredential(provider string) {
	configs, err := loadProviderConfigs()
	if err != nil {
		return
	}
	newConfigs := make(map[string]providerConfig, len(configs))
	for k, v := range configs {
		if k != provider {
			newConfigs[k] = v
		}
	}
	_ = saveProviderConfigs(newConfigs)
}
