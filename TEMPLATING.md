# KVS Templating

Shelly-gitops supports Go template syntax in KVS (Key-Value Store) values, similar to Helm values files.

## Overview

- Templates are only supported in KVS values (not in configs, scripts, etc.)
- Templates use Go template syntax with `{{ }}` delimiters
- Values are provided via a YAML file using the `--values` flag
- When pulling, templated values are preserved (not overwritten with actual device values)

## Usage

### 1. Create a values file

Create a `values.yaml` file with your configuration:

```yaml
environment: production
api:
  endpoint: https://api.example.com
  timeout: 30
thresholds:
  temperature: 22.5
```

### 2. Use templates in KVS

In your device's `kvs/data.json` file, use template syntax:

```json
{
  "api_endpoint": "{{ .api.endpoint }}",
  "update_interval": "{{ .api.timeout }}",
  "temp_threshold": "{{ .thresholds.temperature }}",
  "env": "{{ .environment }}"
}
```

### 3. Push with values file

Push configuration with template rendering:

```bash
# Push to all devices
shelly-gitops push --values values.yaml

# Push to specific devices
shelly-gitops push --values values.yaml --devices device1,device2

# Dry run to see what would be pushed
shelly-gitops push --values values.yaml --dry-run
```

## Template Syntax

Templates use Go's `text/template` syntax:

### Basic value substitution
```
{{ .variableName }}
{{ .test_key }}
```

**Important:** For keys with underscores or special characters, use the `index` function:
```
{{ index . "test_key" }}
{{ index . "my-key" }}
{{ index . "key.with.dots" }}
```

### Nested values
```
{{ .api.endpoint }}
{{ .thresholds.temperature }}
```

Or with index for complex paths:
```
{{ index .api "endpoint" }}
```

### Conditionals
```
{{ if .features.debugMode }}debug{{ else }}production{{ end }}
```

### Default values
```
{{ .timeout | default 30 }}
```

### Working with underscores

While `{{ .test_key }}` *should* work for map keys with underscores, if you encounter issues, use the `index` function:

**Recommended (always works):**
```json
{
  "my_value": "{{ index . \"my_key\" }}"
}
```

**Alternative (may work):**
```json
{
  "my_value": "{{ .my_key }}"
}
```

## Device Context

Templates have access to device information, allowing you to reference the current device or other devices in your manifest.

### Current Device Information

Access the current device being processed using the `.device` context:

```json
{
  "my_ip": "{{ .device.ip_address }}",
  "my_name": "{{ .device.name }}",
  "my_id": "{{ .device.device_id }}",
  "my_mac": "{{ .device.mac_address }}",
  "my_model": "{{ .device.model }}",
  "my_folder": "{{ .device.folder }}"
}
```

### Other Devices

Access information about other devices using the `.devices` map. Use **nested** `index` functions to access devices by their device ID:

```json
{
  "gateway_ip": "{{ index (index .devices \"gateway-device\") \"ip_address\" }}",
  "sensor_name": "{{ index (index .devices \"sensor-01\") \"name\" }}"
}
```

**Important:** You must use nested `index` calls because `.devices` is a map of maps:
- First `index`: gets the device object from `.devices` map using the device ID
- Second `index`: gets the specific field (like "ip_address") from that device object

### Use Cases

**1. Device identification in logs or API calls:**
```json
{
  "device_identifier": "{{ .device.name }} ({{ .device.ip_address }})",
  "mqtt_topic": "home/shelly/{{ .device.device_id }}/status"
}
```

**2. Cross-device communication:**
```json
{
  "gateway_url": "http://{{ index (index .devices \"gateway\") \"ip_address\" }}:8080",
  "coordinator_ip": "{{ index (index .devices \"coordinator\") \"ip_address\" }}"
}
```

**3. Environment-specific + device-specific configuration:**
```json
{
  "api_endpoint": "{{ .api.endpoint }}/devices/{{ .device.device_id }}",
  "backup_server": "{{ .backup.host }}:{{ .backup.port }}",
  "device_name": "{{ .environment }}-{{ .device.name }}"
}
```

**Example with multiple values:**

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
  "mqtt_topic": "{{ .environment }}/{{ .device.name }}/status",
  "coordinator_ip": "{{ index (index .devices \"main-coordinator\") \"ip_address\" }}"
}
```

**Pushed to device (example):**
```json
{
  "mqtt_server": "mqtt.example.com:1883",
  "mqtt_client_id": "production-shelly1pm-abc123",
  "mqtt_topic": "production/Living Room Light/status",
  "coordinator_ip": "192.168.1.100"
}
```

## How It Works

### During Push

1. The tool reads your values file
2. For each KVS value that contains `{{ }}`, it renders the template with your values
3. The rendered value is pushed to the device

Example:
- Template in `kvs/data.json`: `"{{ .api.endpoint }}"`
- Value in `values.yaml`: `api.endpoint: https://api.example.com`
- Pushed to device: `"https://api.example.com"`

### During Pull

1. The tool pulls current KVS values from the device
2. For each key, it checks if the local version is templated
3. If templated, it preserves the template (doesn't overwrite)
4. If not templated, it updates with the device value

This allows you to:
- Keep templates in version control
- Pull actual values without losing templates
- Update templates locally without overwriting them from devices

## Example Workflow

1. **Initial setup**: Pull current values from devices
   ```bash
   shelly-gitops pull
   ```

2. **Add templates**: Edit `<device>/kvs/data.json` to add templates
   ```json
   {
     "api_url": "{{ .api.endpoint }}",
     "static_value": "this stays as-is"
   }
   ```

3. **Create values file**: Create `values.yaml`
   ```yaml
   api:
     endpoint: https://api.example.com
   ```

4. **Push with values**: Deploy rendered values
   ```bash
   shelly-gitops push --values values.yaml
   ```

5. **Future pulls**: Templates are preserved
   ```bash
   shelly-gitops pull  # Templates remain, non-templated values update
   ```

## Best Practices

1. **Use templates for environment-specific values**: API URLs, endpoints, thresholds
2. **Keep static values as-is**: Don't template everything, only what varies
3. **Version control**: Keep both `kvs/data.json` (with templates) and `values.yaml` in git
4. **Multiple environments**: Use different values files (e.g., `values-prod.yaml`, `values-dev.yaml`)
5. **Document templates**: Comment your values file to explain what each value is for

## Notes

- Only string values in KVS can be templated
- Non-string values (numbers, booleans, objects) are used as-is
- Template errors will be reported and skip that specific KVS key
- The `--values` flag is optional; without it, templates remain as-is
