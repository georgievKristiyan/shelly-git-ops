# Device Context in Templates

This document explains how to use device information in KVS templates.

## Overview

Templates have access to device information through the `.device` and `.devices` context variables:

- `.device` - Information about the **current device** being pushed to
- `.devices` - Map of **all devices** in your manifest (keyed by device ID)

## Available Device Fields

Each device context has the following fields:

| Field | Description | Example |
|-------|-------------|---------|
| `device_id` | Unique device identifier | `"shelly1pm-abc123"` |
| `name` | Human-readable device name | `"Living Room Light"` |
| `model` | Device model | `"SHELLY1PM"` |
| `ip_address` | Device IP address | `"192.168.1.100"` |
| `mac_address` | Device MAC address | `"AA:BB:CC:DD:EE:FF"` |
| `folder` | Local folder name | `"living-room-light"` |

## Usage Examples

### Current Device Information

Access the current device's information using `.device`:

```json
{
  "my_ip": "{{ .device.ip_address }}",
  "my_name": "{{ .device.name }}",
  "device_info": "{{ .device.name }} ({{ .device.model }})"
}
```

**Result on device "Living Room Light" (192.168.1.100):**
```json
{
  "my_ip": "192.168.1.100",
  "my_name": "Living Room Light",
  "device_info": "Living Room Light (SHELLY1PM)"
}
```

### Other Devices

Access other devices using **nested** `index` functions with `.devices`:

```json
{
  "gateway_ip": "{{ index (index .devices \"gateway\") \"ip_address\" }}",
  "coordinator_name": "{{ index (index .devices \"coordinator\") \"name\" }}"
}
```

**Important:** You must use nested `index` calls:
- First `index` gets the device from the `.devices` map
- Second `index` gets the field from that device

**Common mistakes to avoid:**
```json
{
  "wrong1": "{{ index .devices \"gateway\" \"ip_address\" }}",
  "wrong2": "{{ .devices.gateway.ip_address }}",
  "wrong3": "{{ index .devices.gateway \"ip_address\" }}",
  "correct": "{{ index (index .devices \"gateway\") \"ip_address\" }}"
}
```

**Assuming manifest has devices with IDs "gateway" and "coordinator":**
```json
{
  "gateway_ip": "192.168.1.1",
  "coordinator_name": "Main Coordinator"
}
```

### Combined with Values

Combine device context with values from your values.yaml:

**values.yaml:**
```yaml
environment: production
mqtt:
  broker: mqtt.example.com
  port: 1883
```

**kvs/data.json:**
```json
{
  "mqtt_server": "{{ .mqtt.broker }}:{{ .mqtt.port }}",
  "mqtt_client_id": "{{ .environment }}-{{ .device.device_id }}",
  "mqtt_topic": "{{ .environment }}/{{ .device.name }}/status"
}
```

**Result:**
```json
{
  "mqtt_server": "mqtt.example.com:1883",
  "mqtt_client_id": "production-shelly1pm-abc123",
  "mqtt_topic": "production/Living Room Light/status"
}
```

## Practical Use Cases

### 1. MQTT Topic Configuration

```json
{
  "mqtt_client_id": "{{ .device.device_id }}",
  "mqtt_topic_status": "home/{{ .device.name }}/status",
  "mqtt_topic_command": "home/{{ .device.name }}/command"
}
```

### 2. Cross-Device Communication

```json
{
  "gateway_url": "http://{{ index (index .devices \"gateway\") \"ip_address\" }}:8080",
  "backup_url": "http://{{ index (index .devices \"backup-server\") \"ip_address\" }}:9090"
}
```

### 3. API Integration

```json
{
  "api_endpoint": "{{ .api.base_url }}/devices/{{ .device.device_id }}",
  "webhook_url": "{{ .webhook.base_url }}/notify/{{ .device.device_id }}"
}
```

### 4. Device Identification

```json
{
  "device_label": "{{ .device.name }} @ {{ .device.ip_address }}",
  "system_id": "{{ .environment }}-{{ .device.model }}-{{ .device.device_id }}"
}
```

## Testing

To test device context templates:

1. **Create a test values file** (see `device-context-values.yaml`)

2. **Create a KVS file with device templates** (see `device-context-example.json`)

3. **Run a dry-run push:**
   ```bash
   shelly-gitops push --values device-context-values.yaml --dry-run
   ```

4. **Check the rendered output** in the logs

5. **Push to specific device:**
   ```bash
   shelly-gitops push --values device-context-values.yaml --devices your-device-id
   ```

## Notes

- Device IDs must match exactly (case-sensitive) when using `{{ index .devices "device-id" }}`
- If a referenced device doesn't exist, the template will fail with an error
- Device context is available even without a values file (just omit `--values` flag)
- All device fields are always available (empty string if not set)

## Examples in This Directory

- `device-context-example.json` - Complete KVS example using device context
- `device-context-values.yaml` - Sample values file with comments
- This file (`DEVICE_CONTEXT.md`) - Complete documentation
