package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/darkermage/shelly-git-ops/internal/shelly"
	"gopkg.in/yaml.v3"
)

// DeviceStorage handles device folder structure and file operations
type DeviceStorage struct {
	repoPath string
}

// DeviceMetadata represents device metadata stored in device.yaml
type DeviceMetadata struct {
	DeviceID   string `yaml:"device_id"`
	Name       string `yaml:"name"`
	Model      string `yaml:"model"`
	Firmware   string `yaml:"firmware"`
	IPAddress  string `yaml:"ip_address"`
	MACAddress string `yaml:"mac_address"`
}

// ScriptMetadata represents script metadata
type ScriptMetadata struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
}

// NewDeviceStorage creates a new device storage handler
func NewDeviceStorage(repoPath string) *DeviceStorage {
	return &DeviceStorage{
		repoPath: repoPath,
	}
}

// GetDevicePath returns the full path to a device folder
func (ds *DeviceStorage) GetDevicePath(folderName string) string {
	return filepath.Join(ds.repoPath, folderName)
}

// CreateDeviceFolder creates the folder structure for a device
func (ds *DeviceStorage) CreateDeviceFolder(folderName string) error {
	devicePath := ds.GetDevicePath(folderName)

	// Create main device folder
	if err := os.MkdirAll(devicePath, 0755); err != nil {
		return fmt.Errorf("failed to create device folder: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"scripts", "virtual-components", "groups", "kvs", "configs", "schedules", "webhooks"}
	for _, subdir := range subdirs {
		path := filepath.Join(devicePath, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s folder: %w", subdir, err)
		}
	}

	return nil
}

// SaveDeviceMetadata saves device metadata to device.yaml
func (ds *DeviceStorage) SaveDeviceMetadata(folderName string, metadata DeviceMetadata) error {
	devicePath := ds.GetDevicePath(folderName)
	metadataPath := filepath.Join(devicePath, "device.yaml")

	data, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// LoadDeviceMetadata loads device metadata from device.yaml
func (ds *DeviceStorage) LoadDeviceMetadata(folderName string) (*DeviceMetadata, error) {
	devicePath := ds.GetDevicePath(folderName)
	metadataPath := filepath.Join(devicePath, "device.yaml")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata DeviceMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// SaveConfig saves system-level device configuration to config.json
func (ds *DeviceStorage) SaveConfig(folderName string, config json.RawMessage) error {
	devicePath := ds.GetDevicePath(folderName)
	configPath := filepath.Join(devicePath, "config.json")

	// Pretty print JSON
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(config, &prettyJSON); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	data, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// SaveShellyConfig saves component-level device configuration to shelly-config.json
func (ds *DeviceStorage) SaveShellyConfig(folderName string, config json.RawMessage) error {
	devicePath := ds.GetDevicePath(folderName)
	configPath := filepath.Join(devicePath, "shelly-config.json")

	// Pretty print JSON
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(config, &prettyJSON); err != nil {
		return fmt.Errorf("failed to unmarshal shelly config: %w", err)
	}

	data, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal shelly config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write shelly config: %w", err)
	}

	return nil
}

// SaveComponentConfig saves a component configuration to configs/<component>.json
func (ds *DeviceStorage) SaveComponentConfig(folderName, component string, config json.RawMessage) error {
	devicePath := ds.GetDevicePath(folderName)
	configsPath := filepath.Join(devicePath, "configs")

	// Ensure configs directory exists
	if err := os.MkdirAll(configsPath, 0755); err != nil {
		return fmt.Errorf("failed to create configs directory: %w", err)
	}

	configPath := filepath.Join(configsPath, component+".json")

	// Pretty print JSON
	var prettyJSON interface{}
	if err := json.Unmarshal(config, &prettyJSON); err != nil {
		return fmt.Errorf("failed to unmarshal %s config: %w", component, err)
	}

	data, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s config: %w", component, err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s config: %w", component, err)
	}

	return nil
}

// LoadComponentConfig loads a component configuration from configs/<component>.json
func (ds *DeviceStorage) LoadComponentConfig(folderName, component string) (json.RawMessage, error) {
	devicePath := ds.GetDevicePath(folderName)
	configPath := filepath.Join(devicePath, "configs", component+".json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s config: %w", component, err)
	}

	return json.RawMessage(data), nil
}

// ListComponentConfigs lists all component config files
func (ds *DeviceStorage) ListComponentConfigs(folderName string) ([]string, error) {
	devicePath := ds.GetDevicePath(folderName)
	configsPath := filepath.Join(devicePath, "configs")

	entries, err := os.ReadDir(configsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read configs directory: %w", err)
	}

	var components []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		// Remove .json extension to get component name
		component := entry.Name()[:len(entry.Name())-5]
		components = append(components, component)
	}

	return components, nil
}

// LoadConfig loads device configuration from config.json
func (ds *DeviceStorage) LoadConfig(folderName string) (json.RawMessage, error) {
	devicePath := ds.GetDevicePath(folderName)
	configPath := filepath.Join(devicePath, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return json.RawMessage(data), nil
}

// SaveScript saves a script and its metadata
func (ds *DeviceStorage) SaveScript(folderName string, script *shelly.ScriptCode, enable bool) error {
	devicePath := ds.GetDevicePath(folderName)
	scriptsPath := filepath.Join(devicePath, "scripts")

	// Save script code
	scriptFile := filepath.Join(scriptsPath, fmt.Sprintf("script-%d.js", script.ID))
	if err := os.WriteFile(scriptFile, []byte(script.Code), 0644); err != nil {
		return fmt.Errorf("failed to write script code: %w", err)
	}

	// Save script metadata
	metadata := ScriptMetadata{
		ID:     script.ID,
		Name:   script.Name,
		Enable: enable,
	}

	metadataFile := filepath.Join(scriptsPath, fmt.Sprintf("script-%d.meta.json", script.ID))
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal script metadata: %w", err)
	}

	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write script metadata: %w", err)
	}

	return nil
}

// ListScripts lists all scripts in the device folder
func (ds *DeviceStorage) ListScripts(folderName string) ([]ScriptMetadata, error) {
	devicePath := ds.GetDevicePath(folderName)
	scriptsPath := filepath.Join(devicePath, "scripts")

	entries, err := os.ReadDir(scriptsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scripts directory: %w", err)
	}

	var scripts []ScriptMetadata
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(scriptsPath, entry.Name()))
		if err != nil {
			continue
		}

		var metadata ScriptMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			continue
		}

		scripts = append(scripts, metadata)
	}

	return scripts, nil
}

// LoadScript loads a script code from file
func (ds *DeviceStorage) LoadScript(folderName string, scriptID int) (string, error) {
	devicePath := ds.GetDevicePath(folderName)
	scriptFile := filepath.Join(devicePath, "scripts", fmt.Sprintf("script-%d.js", scriptID))

	data, err := os.ReadFile(scriptFile)
	if err != nil {
		return "", fmt.Errorf("failed to read script: %w", err)
	}

	return string(data), nil
}

// DeleteScript deletes a script and its metadata
func (ds *DeviceStorage) DeleteScript(folderName string, scriptID int) error {
	devicePath := ds.GetDevicePath(folderName)
	scriptsPath := filepath.Join(devicePath, "scripts")

	scriptFile := filepath.Join(scriptsPath, fmt.Sprintf("script-%d.js", scriptID))
	metadataFile := filepath.Join(scriptsPath, fmt.Sprintf("script-%d.meta.json", scriptID))

	os.Remove(scriptFile)
	os.Remove(metadataFile)

	return nil
}

// SaveVirtualComponent saves a virtual component configuration
func (ds *DeviceStorage) SaveVirtualComponent(folderName, componentType string, componentID int, data json.RawMessage) error {
	devicePath := ds.GetDevicePath(folderName)
	componentsPath := filepath.Join(devicePath, "virtual-components")

	filename := filepath.Join(componentsPath, fmt.Sprintf("%s-%d.json", componentType, componentID))

	// Pretty print JSON
	var prettyJSON interface{}
	if err := json.Unmarshal(data, &prettyJSON); err != nil {
		return fmt.Errorf("failed to unmarshal component: %w", err)
	}

	prettyData, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal component: %w", err)
	}

	if err := os.WriteFile(filename, prettyData, 0644); err != nil {
		return fmt.Errorf("failed to write component: %w", err)
	}

	return nil
}

// SaveGroup saves a group configuration
func (ds *DeviceStorage) SaveGroup(folderName string, group *shelly.Group) error {
	devicePath := ds.GetDevicePath(folderName)
	groupsPath := filepath.Join(devicePath, "groups")

	filename := filepath.Join(groupsPath, fmt.Sprintf("group-%d.json", group.ID))

	data, err := json.MarshalIndent(group, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal group: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write group: %w", err)
	}

	return nil
}

// SaveKVS saves key-value store data
func (ds *DeviceStorage) SaveKVS(folderName string, data map[string]interface{}) error {
	devicePath := ds.GetDevicePath(folderName)
	kvsPath := filepath.Join(devicePath, "kvs", "data.json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal KVS data: %w", err)
	}

	if err := os.WriteFile(kvsPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write KVS data: %w", err)
	}

	return nil
}

// LoadKVS loads key-value store data
func (ds *DeviceStorage) LoadKVS(folderName string) (map[string]interface{}, error) {
	devicePath := ds.GetDevicePath(folderName)
	kvsPath := filepath.Join(devicePath, "kvs", "data.json")

	// If file doesn't exist, return empty map
	if _, err := os.Stat(kvsPath); os.IsNotExist(err) {
		return make(map[string]interface{}), nil
	}

	data, err := os.ReadFile(kvsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read KVS data: %w", err)
	}

	var kvsData map[string]interface{}
	if err := json.Unmarshal(data, &kvsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KVS data: %w", err)
	}

	return kvsData, nil
}

// SaveSchedule saves a schedule to file
func (ds *DeviceStorage) SaveSchedule(folderName string, schedule *shelly.Schedule) error {
	devicePath := ds.GetDevicePath(folderName)
	schedulesPath := filepath.Join(devicePath, "schedules")

	filename := filepath.Join(schedulesPath, fmt.Sprintf("schedule-%d.json", schedule.ID))

	data, err := json.MarshalIndent(schedule, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schedule: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write schedule: %w", err)
	}

	return nil
}

// ListSchedules lists all schedules in the device folder
func (ds *DeviceStorage) ListSchedules(folderName string) ([]*shelly.Schedule, error) {
	devicePath := ds.GetDevicePath(folderName)
	schedulesPath := filepath.Join(devicePath, "schedules")

	entries, err := os.ReadDir(schedulesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*shelly.Schedule{}, nil
		}
		return nil, fmt.Errorf("failed to read schedules directory: %w", err)
	}

	var schedules []*shelly.Schedule
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(schedulesPath, entry.Name()))
		if err != nil {
			continue
		}

		var schedule shelly.Schedule
		if err := json.Unmarshal(data, &schedule); err != nil {
			continue
		}

		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

// DeleteSchedule deletes a schedule file
func (ds *DeviceStorage) DeleteSchedule(folderName string, scheduleID int) error {
	devicePath := ds.GetDevicePath(folderName)
	schedulesPath := filepath.Join(devicePath, "schedules")
	filename := filepath.Join(schedulesPath, fmt.Sprintf("schedule-%d.json", scheduleID))
	return os.Remove(filename)
}

// SaveWebhook saves a webhook to file
func (ds *DeviceStorage) SaveWebhook(folderName string, webhook *shelly.Webhook) error {
	devicePath := ds.GetDevicePath(folderName)
	webhooksPath := filepath.Join(devicePath, "webhooks")

	filename := filepath.Join(webhooksPath, fmt.Sprintf("webhook-%d.json", webhook.ID))

	data, err := json.MarshalIndent(webhook, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal webhook: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write webhook: %w", err)
	}

	return nil
}

// ListWebhooks lists all webhooks in the device folder
func (ds *DeviceStorage) ListWebhooks(folderName string) ([]*shelly.Webhook, error) {
	devicePath := ds.GetDevicePath(folderName)
	webhooksPath := filepath.Join(devicePath, "webhooks")

	entries, err := os.ReadDir(webhooksPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*shelly.Webhook{}, nil
		}
		return nil, fmt.Errorf("failed to read webhooks directory: %w", err)
	}

	var webhooks []*shelly.Webhook
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(webhooksPath, entry.Name()))
		if err != nil {
			continue
		}

		var webhook shelly.Webhook
		if err := json.Unmarshal(data, &webhook); err != nil {
			continue
		}

		webhooks = append(webhooks, &webhook)
	}

	return webhooks, nil
}

// DeleteWebhook deletes a webhook file
func (ds *DeviceStorage) DeleteWebhook(folderName string, webhookID int) error {
	devicePath := ds.GetDevicePath(folderName)
	webhooksPath := filepath.Join(devicePath, "webhooks")
	filename := filepath.Join(webhooksPath, fmt.Sprintf("webhook-%d.json", webhookID))
	return os.Remove(filename)
}

// DeviceExists checks if a device folder exists
func (ds *DeviceStorage) DeviceExists(folderName string) bool {
	devicePath := ds.GetDevicePath(folderName)
	info, err := os.Stat(devicePath)
	return err == nil && info.IsDir()
}

// RemoveDevice removes a device folder completely
func (ds *DeviceStorage) RemoveDevice(folderName string) error {
	devicePath := ds.GetDevicePath(folderName)
	return os.RemoveAll(devicePath)
}
