package storage

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Manifest represents the root manifest file
type Manifest struct {
	Version   string          `yaml:"version"`
	Discovery DiscoveryConfig `yaml:"discovery"`
	Devices   []Device        `yaml:"devices"`
	filePath  string
}

// DiscoveryConfig holds discovery provider configuration
type DiscoveryConfig struct {
	Provider      string `yaml:"provider"`
	ControllerURL string `yaml:"controller_url,omitempty"`
}

// Device represents a device in the manifest
type Device struct {
	DeviceID   string    `yaml:"device_id"`
	Name       string    `yaml:"name"`
	Folder     string    `yaml:"folder"`
	IPAddress  string    `yaml:"ip_address"`
	MACAddress string    `yaml:"mac_address"`
	Model      string    `yaml:"model"`
	LastSync   time.Time `yaml:"last_sync"`
}

// LoadManifest loads a manifest from a YAML file
func LoadManifest(filePath string) (*Manifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty manifest if file doesn't exist
			return &Manifest{
				Version:  "1.0",
				Devices:  []Device{},
				filePath: filePath,
			}, nil
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	manifest.filePath = filePath
	return &manifest, nil
}

// Save saves the manifest to its file
func (m *Manifest) Save() error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// AddDevice adds a device to the manifest
func (m *Manifest) AddDevice(device Device) {
	// Check if device already exists
	for i, d := range m.Devices {
		if d.DeviceID == device.DeviceID {
			// Update existing device
			m.Devices[i] = device
			return
		}
	}

	// Add new device
	m.Devices = append(m.Devices, device)
}

// RemoveDevice removes a device from the manifest by device ID
func (m *Manifest) RemoveDevice(deviceID string) bool {
	for i, d := range m.Devices {
		if d.DeviceID == deviceID {
			m.Devices = append(m.Devices[:i], m.Devices[i+1:]...)
			return true
		}
	}
	return false
}

// GetDevice retrieves a device by ID
func (m *Manifest) GetDevice(deviceID string) *Device {
	for _, d := range m.Devices {
		if d.DeviceID == deviceID {
			return &d
		}
	}
	return nil
}

// GetDeviceByIP retrieves a device by IP address
func (m *Manifest) GetDeviceByIP(ip string) *Device {
	for _, d := range m.Devices {
		if d.IPAddress == ip {
			return &d
		}
	}
	return nil
}

// UpdateLastSync updates the last sync time for a device
func (m *Manifest) UpdateLastSync(deviceID string, syncTime time.Time) {
	for i, d := range m.Devices {
		if d.DeviceID == deviceID {
			m.Devices[i].LastSync = syncTime
			return
		}
	}
}
