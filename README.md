# Shelly Git-Ops

Git-based infrastructure-as-code for Shelly smart home devices. Manage your Shelly device configurations, scripts, and virtual components using familiar Git workflows.

## Features

- **Git-based workflow**: Treat device configurations as code with version control
- **Network discovery**: Automatically discover Shelly devices on your network
- **Provider architecture**: Pluggable discovery providers (UniFi supported)
- **Bidirectional sync**: Pull configurations from devices and push changes back
- **Simple workflow**: Push/pull operations work directly with your local files
- **Script management**: Version control your Shelly scripts
- **Parallel operations**: Concurrent device operations for speed

## Architecture

```
┌─────────────────┐
│   Git Repo      │
│  ┌───────────┐  │
│  │   Files   │  │ <- Device configurations
│  └─────┬─────┘  │
│        │        │
└────────┼────────┘
         │
         ▼
┌─────────────────┐
│  Shelly Devices │
│  ┌───┐ ┌───┐   │
│  │ 1 │ │ 2 │...│
│  └───┘ └───┘   │
└─────────────────┘
```

## Installation

### From Source

```bash
git clone https://github.com/darkermage/shelly-git-ops
cd shelly-git-ops
go build -o shelly-gitops ./cmd/shelly-gitops
sudo mv shelly-gitops /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/darkermage/shelly-git-ops/cmd/shelly-gitops@latest
```

## Quick Start

### 1. Initialize Repository

```bash
mkdir my-shelly-devices
cd my-shelly-devices
shelly-gitops init
```

This creates:
- Git repository
- `manifest.yaml` - Device registry

### 2. Discover Devices

Using UniFi Controller:

```bash
shelly-gitops discover scan \
  --provider unifi \
  --controller-url https://unifi.local:8443 \
  --username admin \
  --filter "shelly*"
```

This will:
- Authenticate with UniFi
- Discover all devices matching "shelly*"
- Query each device via Shelly API
- Create device folders
- Pull initial configurations

### 3. View Status

```bash
shelly-gitops status
```

Example output:
```
Current branch: main

Working tree clean

Devices (3):
  - shelly-plus-1pm-living (Shelly Plus 1PM)
    IP: 192.168.1.100
    Last sync: 2025-11-28 10:30:00
  - shelly-pro4pm-garage (Shelly Pro 4PM)
    IP: 192.168.1.101
    Last sync: 2025-11-28 10:30:00
```

### Dynamic Configuration Discovery

The tool automatically discovers all available configuration methods on each device:

1. **Calls `Shelly.ListMethods`** - Gets all RPC methods the device supports
2. **Finds `*.GetConfig` methods** - Identifies all configuration endpoints
3. **Retrieves each config** - Calls every `GetConfig` method
4. **Stores separately** - Saves each component config to `configs/<component>.json`

This means:
- New device features are automatically supported
- Different device models work seamlessly
- Firmware updates adding new components are captured automatically

### 4. Pull Latest State

```bash
shelly-gitops pull
```

Fetches current configuration from all devices and overwrites local files.

Output example:
```
Pulling configuration from devices...
✓ shellyplus1pm-abc123: saved 12 config(s), 2 script(s)
✓ shellypro4pm-def456: saved 15 config(s), 1 script(s)

Successfully synced 2/2 devices
```

**Note on Device Names**: The pull command automatically syncs device names from `Shelly.GetDeviceInfo`. If you rename a device in the Shelly app or web interface:
- The next `pull` will update the name in `manifest.yaml`
- The device folder will be renamed to match (e.g., `old-name-abc123` → `new-name-abc123`)
- Spaces and special characters are converted to dashes for filesystem safety

### 5. Make Changes

Create a feature branch:

```bash
git checkout -b feature/auto-off-timer
```

Edit configurations or scripts:

```bash
# Edit a component config
vim living-room-light-abc123/configs/switch.json

# Edit a script
vim living-room-light-abc123/scripts/script-1.js
```

Example config change (`configs/switch.json`):

```json
{
  "id": 0,
  "auto_off": true,
  "auto_off_delay": 300.0,
  "initial_state": "restore_last",
  "in_mode": "follow",
  "name": "Living Room Light"
}
```

Example script change:

```javascript
// Auto-off timer for light
let CONFIG = {
  timeout: 300  // 5 minutes
};

Shelly.addEventHandler(function(event) {
  if (event.name === "switch" && event.info.state === true) {
    Timer.set(CONFIG.timeout * 1000, false, function() {
      Shelly.call("Switch.Set", {id: 0, on: false});
    });
  }
});
```

Update script metadata:

```json
{
  "id": 1,
  "name": "Auto-off Timer",
  "enable": true
}
```

Commit changes:

```bash
git add .
git commit -m "Add auto-off timer to living room light"
```

### 6. Apply Changes

```bash
# Preview changes
shelly-gitops push --dry-run

# Apply to devices
shelly-gitops push
```

**Note on Script Updates**: When pushing scripts to devices:
- Running scripts are automatically stopped before upload
- Scripts are then updated with the new code
- The enabled/disabled state from `script-X.meta.json` is applied
- Scripts marked as `"enable": true` are automatically started after upload

## Repository Structure

```
my-shelly-devices/
├── manifest.yaml                      # Device registry
├── .git/                              # Git repository
└── living-room-light-abc123/          # Device folder
    ├── device.yaml                    # Device metadata
    ├── configs/                       # Component configurations
    │   ├── ble.json                   # BLE.GetConfig
    │   ├── cloud.json                 # Cloud.GetConfig
    │   ├── humidity.json              # Humidity.GetConfig (if available)
    │   ├── illuminance.json           # Illuminance.GetConfig (if available)
    │   ├── input.json                 # Input.GetConfig
    │   ├── mqtt.json                  # MQTT.GetConfig
    │   ├── switch.json                # Switch.GetConfig
    │   ├── sys.json                   # Sys.GetConfig
    │   ├── temperature.json           # Temperature.GetConfig (if available)
    │   ├── thermostat.json            # Thermostat.GetConfig (if available)
    │   ├── ui.json                    # UI.GetConfig (if available)
    │   └── wifi.json                  # WiFi.GetConfig
    ├── scripts/
    │   ├── script-1.js                # Script code
    │   ├── script-1.meta.json         # Script metadata
    │   ├── script-2.js
    │   └── script-2.meta.json
    ├── virtual-components/
    │   ├── boolean-0.json
    │   ├── number-0.json
    │   └── text-0.json
    └── kvs/
        └── data.json                  # Key-Value Store
```

**Note**: The exact configs available depend on the device model and firmware. The tool automatically discovers all `*.GetConfig` methods and retrieves their configurations.

## Workflow Examples

### Adding a New Device Manually

If a device isn't discovered automatically:

1. Add to `manifest.yaml`:

```yaml
devices:
  - device_id: "shellypro4pm-abc123"
    name: "new-device"
    folder: "new-device-abc123"
    ip_address: "192.168.1.150"
    mac_address: "AA:BB:CC:DD:EE:FF"
    model: "Shelly Pro 4PM"
    last_sync: "2025-11-28T10:00:00Z"
```

2. Pull configuration:

```bash
shelly-gitops pull
```

### Setting Static DHCP (Future Feature)

```bash
shelly-gitops discover set-dhcp \
  --device-id shellypro4pm-abc123 \
  --ip 192.168.1.150 \
  --provider unifi
```

### Rollback Changes

Using Git:

```bash
# Rollback to previous commit
git revert HEAD

# Push old configuration back to devices
shelly-gitops push
```

### Branching Strategy

```bash
# Create branches for different environments
git checkout -b production
git checkout -b staging

# Different device sets per branch
# Modify manifest.yaml accordingly
```

## Commands Reference

### Global Flags

- `--repo <path>` - Repository path (default: current directory)

### Commands

#### `init`

Initialize a new repository.

```bash
shelly-gitops init
```

#### `discover scan`

Discover and add devices.

```bash
shelly-gitops discover scan \
  --provider unifi \
  --controller-url https://unifi.local:8443 \
  --username admin \
  [--password secret] \
  [--filter "shelly*"] \
  [--verify-ssl]
```

Flags:
- `--provider` - Discovery provider (currently: `unifi`)
- `--controller-url` - Controller URL (required)
- `--username` - Username (required)
- `--password` - Password (prompts if not provided)
- `--filter` - Hostname filter pattern (default: `shelly*`)
- `--verify-ssl` - Verify SSL certificates (default: false)

#### `pull`

Pull configurations from all devices and overwrite local files.

```bash
shelly-gitops pull
```

#### `push`

Push all local configurations to devices.

```bash
shelly-gitops push [--dry-run]
```

Flags:
- `--dry-run` - Preview changes without applying

#### `status`

Show repository and device status.

```bash
shelly-gitops status
```

## Configuration

### Manifest File

`manifest.yaml` structure:

```yaml
version: "1.0"
discovery:
  provider: "unifi"
  controller_url: "https://unifi.local:8443"
devices:
  - device_id: "shellypro4pm-abc123"
    name: "garage-switches"
    folder: "garage-switches-abc123"
    ip_address: "192.168.1.101"
    mac_address: "A8:03:2A:B6:78:90"
    model: "Shelly Pro 4PM"
    last_sync: "2025-11-28T10:30:00Z"
```

### Device Folder

Each device has:
- `device.yaml` - Metadata (model, firmware, IPs)
- `configs/` - All component configurations (auto-discovered via `Shelly.ListMethods`)
  - Each `*.GetConfig` method gets its own JSON file
  - Examples: `switch.json`, `wifi.json`, `thermostat.json`, `sys.json`, etc.
  - Automatically adapts to device capabilities
- `scripts/` - Script files and metadata
- `virtual-components/` - Virtual component configurations
- `kvs/` - Key-Value Store data

## Supported Providers

### UniFi

Requires:
- UniFi Network Controller (version 6.x or later)
- Controller URL (e.g., `https://unifi.local:8443`)
- Admin credentials
- Network access to controller

### Future Providers

Planned:
- mDNS/Bonjour discovery
- IP range scanning
- Manual CSV import
- Home Assistant integration

## Development

### Project Structure

```
shelly-git-ops/
├── cmd/shelly-gitops/        # CLI entry point
├── internal/
│   ├── discovery/           # Discovery provider interface
│   │   └── unifi/          # UniFi provider
│   ├── gitops/             # Git operations & sync
│   ├── shelly/             # Shelly API client
│   ├── storage/            # Manifest & device storage
│   └── config/             # Configuration management
├── pkg/                    # Public APIs
└── go.mod
```

### Building

```bash
go build -o shelly-gitops ./cmd/shelly-gitops
```

### Testing

```bash
go test ./...
```

### Adding a New Discovery Provider

1. Implement `discovery.Provider` interface
2. Add to provider factory in CLI
3. Document in README

Example:

```go
type MyProvider struct {}

func (p *MyProvider) Authenticate(ctx context.Context, creds map[string]string) error {
    // Implementation
}

func (p *MyProvider) DiscoverDevices(ctx context.Context, filter string) ([]DeviceInfo, error) {
    // Implementation
}

// ... implement other interface methods
```

## Security Considerations

### Credentials

- Passwords are prompted interactively (not stored in shell history)
- Config files use restrictive permissions (0600)
- Never commit credentials to Git
- Use environment variables or secure vaults for CI/CD

### Network Access

- Devices communicate over local network (HTTP)
- Consider using HTTPS/TLS where supported
- Use VLANs to isolate IoT devices
- Enable firewall rules

### Git Repository

- Review changes before pushing to devices
- Use branch protection in Git hosting
- Sign commits for verification
- Audit logs via Git history

## Troubleshooting

### Discovery Issues

**No devices found:**
- Verify UniFi controller URL and credentials
- Check filter pattern matches device hostnames
- Ensure devices are online and connected

**Authentication failed:**
- Verify username/password
- Check controller accessibility
- Try with `--verify-ssl=false` for self-signed certs

### Sync Issues

**Pull fails:**
- Check device IP addresses are reachable
- Verify devices are powered on
- Check firewall rules

**Push fails:**
- Ensure scripts are valid JavaScript
- Check device has enough storage
- Verify script IDs don't conflict
- Running scripts are automatically stopped before upload - this is normal behavior

**Local changes:**
- Pull will overwrite local files with device state
- Push will apply all local files to devices
- Use Git to manage your own change history and branches

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Submit a pull request

## License

MIT License - see LICENSE file

## Support

- GitHub Issues: https://github.com/darkermage/shelly-git-ops/issues
- Shelly API Docs: https://shelly-api-docs.shelly.cloud/

## Acknowledgments

- Shelly for excellent IoT devices and API
- Go-Git for pure Go git implementation
- Cobra for CLI framework
