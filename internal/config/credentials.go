package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Credentials holds credential information
type Credentials struct {
	Provider      string            `json:"provider"`
	ControllerURL string            `json:"controller_url,omitempty"`
	Username      string            `json:"username,omitempty"`
	Password      string            `json:"password,omitempty"`
	Custom        map[string]string `json:"custom,omitempty"`
}

// CredentialStore manages credential storage
type CredentialStore struct {
	configPath string
}

// NewCredentialStore creates a new credential store
func NewCredentialStore(configPath string) *CredentialStore {
	return &CredentialStore{
		configPath: configPath,
	}
}

// GetDefaultConfigPath returns the default config path
func GetDefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".shelly-gitops", "credentials.json"), nil
}

// Save saves credentials to the config file
func (cs *CredentialStore) Save(creds Credentials) error {
	// Ensure directory exists
	dir := filepath.Dir(cs.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write with restricted permissions
	if err := os.WriteFile(cs.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// Load loads credentials from the config file
func (cs *CredentialStore) Load() (*Credentials, error) {
	data, err := os.ReadFile(cs.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("credentials file not found")
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// Delete deletes the credentials file
func (cs *CredentialStore) Delete() error {
	return os.Remove(cs.configPath)
}

// Exists checks if credentials file exists
func (cs *CredentialStore) Exists() bool {
	_, err := os.Stat(cs.configPath)
	return err == nil
}
