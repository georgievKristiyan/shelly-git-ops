package unifi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/darkermage/shelly-git-ops/internal/discovery"
)

// Provider implements the discovery.Provider interface for UniFi
type Provider struct {
	client        *Client
	controllerURL string
	verifySSL     bool
}

// NewProvider creates a new UniFi discovery provider
func NewProvider(controllerURL string, verifySSL bool) *Provider {
	return &Provider{
		controllerURL: controllerURL,
		verifySSL:     verifySSL,
	}
}

// Authenticate authenticates with the UniFi controller
func (p *Provider) Authenticate(ctx context.Context, credentials map[string]string) error {
	username, ok := credentials["username"]
	if !ok {
		return fmt.Errorf("username not provided in credentials")
	}

	password, ok := credentials["password"]
	if !ok {
		return fmt.Errorf("password not provided in credentials")
	}

	// Get site name from credentials, default to "default"
	site := credentials["site"]
	if site == "" {
		site = "default"
	}

	client, err := NewClient(p.controllerURL, username, password, p.verifySSL)
	if err != nil {
		return err
	}

	// Set site name
	client.site = site

	p.client = client
	return nil
}

// DiscoverDevices discovers devices on the network
func (p *Provider) DiscoverDevices(ctx context.Context, filterPattern string) ([]discovery.DeviceInfo, error) {
	if p.client == nil {
		return nil, fmt.Errorf("not authenticated, call Authenticate first")
	}

	unifiDevices, err := p.client.GetClients(ctx)
	if err != nil {
		return nil, err
	}

	var devices []discovery.DeviceInfo
	for _, ud := range unifiDevices {
		// Skip devices without IP
		if ud.IP == "" {
			continue
		}

		hostname := ud.Hostname
		if hostname == "" {
			hostname = ud.Name
		}

		// Apply filter if provided
		if filterPattern != "" && !matchesPattern(hostname, filterPattern) {
			continue
		}

		device := discovery.DeviceInfo{
			MACAddress: ud.MAC,
			IPAddress:  ud.IP,
			Hostname:   hostname,
			LastSeen:   time.Unix(ud.LastSeen, 0),
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// SetDHCPLease sets a static DHCP lease for a device
func (p *Provider) SetDHCPLease(ctx context.Context, lease discovery.DHCPLease) error {
	if p.client == nil {
		return fmt.Errorf("not authenticated, call Authenticate first")
	}

	return p.client.SetStaticIP(ctx, lease.MACAddress, lease.IPAddress, lease.Hostname)
}

// GetDeviceByMAC retrieves device information by MAC address
func (p *Provider) GetDeviceByMAC(ctx context.Context, mac string) (*discovery.DeviceInfo, error) {
	if p.client == nil {
		return nil, fmt.Errorf("not authenticated, call Authenticate first")
	}

	devices, err := p.DiscoverDevices(ctx, "")
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if strings.EqualFold(device.MACAddress, mac) {
			return &device, nil
		}
	}

	return nil, fmt.Errorf("device with MAC %s not found", mac)
}

// Close closes the provider connection
func (p *Provider) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// matchesPattern checks if hostname matches the filter pattern
// Simple wildcard matching: "shelly*" matches "shelly-plus-1pm"
func matchesPattern(hostname, pattern string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	// Simple prefix matching for patterns like "shelly*"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(strings.ToLower(hostname), strings.ToLower(prefix))
	}

	// Exact match
	return strings.EqualFold(hostname, pattern)
}
