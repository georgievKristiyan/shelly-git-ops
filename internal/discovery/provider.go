package discovery

import "context"

// Provider defines the interface for network discovery providers
type Provider interface {
	// Authenticate with the network controller
	Authenticate(ctx context.Context, credentials map[string]string) error

	// DiscoverDevices discovers devices on the network
	// filterPattern can be used to filter devices (e.g., "shelly*")
	DiscoverDevices(ctx context.Context, filterPattern string) ([]DeviceInfo, error)

	// SetDHCPLease sets a static DHCP lease for a device
	SetDHCPLease(ctx context.Context, lease DHCPLease) error

	// GetDeviceByMAC retrieves device information by MAC address
	GetDeviceByMAC(ctx context.Context, mac string) (*DeviceInfo, error)

	// Close closes any open connections
	Close() error
}
