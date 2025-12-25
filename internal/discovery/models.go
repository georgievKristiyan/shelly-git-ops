package discovery

import "time"

// DeviceInfo represents discovered device information
type DeviceInfo struct {
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	Hostname   string `json:"hostname,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
	LastSeen   time.Time `json:"last_seen,omitempty"`
}

// DHCPLease represents DHCP lease configuration
type DHCPLease struct {
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	Hostname   string `json:"hostname"`
}

// IsShelly checks if the device is a Shelly device based on hostname
func (d *DeviceInfo) IsShelly() bool {
	if len(d.Hostname) < 6 {
		return false
	}
	return d.Hostname[:6] == "shelly" || d.Hostname[:6] == "Shelly"
}
