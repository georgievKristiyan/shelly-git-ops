# KVS Templating Quick Reference

## Correct Template Syntax

### Current Device Fields
```json
{
  "device_ip": "{{ .device.ip_address }}",
  "device_name": "{{ .device.name }}",
  "device_id": "{{ .device.device_id }}",
  "device_mac": "{{ .device.mac_address }}",
  "device_model": "{{ .device.model }}",
  "device_folder": "{{ .device.folder }}"
}
```

### Other Devices - NESTED INDEX REQUIRED
```json
{
  "other_device_ip": "{{ index (index .devices \"device-id\") \"ip_address\" }}",
  "other_device_name": "{{ index (index .devices \"device-id\") \"name\" }}"
}
```

### Values from values.yaml
```json
{
  "simple_value": "{{ .my_value }}",
  "nested_value": "{{ .config.setting }}",
  "underscore_key": "{{ index . \"my_key\" }}"
}
```

## Common Mistakes ❌

### WRONG - Single index with two arguments
```json
{
  "ip": "{{ index .devices \"device-id\" \"ip_address\" }}"
}
```

### WRONG - Dot notation with device ID
```json
{
  "ip": "{{ .devices.device-id.ip_address }}"
}
```

### WRONG - Mixed dot and index
```json
{
  "ip": "{{ index .devices.device-id \"ip_address\" }}"
}
```

### CORRECT ✓ - Nested index
```json
{
  "ip": "{{ index (index .devices \"device-id\") \"ip_address\" }}"
}
```

## How Nested Index Works

The `index` function takes a map and a key, returning the value:

```
{{ index MAP KEY }}
```

For `.devices`:
1. `.devices` is a map where keys are device IDs
2. Each value is another map with device fields

So to get a device's IP:
```
Step 1: {{ index .devices "device-id" }}           → returns device object (a map)
Step 2: {{ index DEVICE_OBJECT "ip_address" }}     → returns IP string
Combined: {{ index (index .devices "device-id") "ip_address" }}
```

## Real Example

**manifest.json has these devices:**
- Device ID: `shellypro4pm-841fe89605fc`
- Device ID: `ShellyWallDisplay-000822901732`

**kvs/data.json:**
```json
{
  "my_ip": "{{ .device.ip_address }}",
  "display_ip": "{{ index (index .devices \"ShellyWallDisplay-000822901732\") \"ip_address\" }}",
  "pro4pm_name": "{{ index (index .devices \"shellypro4pm-841fe89605fc\") \"name\" }}"
}
```

**If you see errors**, check:
1. Device ID is spelled exactly as it appears in `manifest.json`
2. You're using nested `index` calls: `{{ index (index .devices "id") "field" }}`
3. All quotes are escaped properly: `\"`
4. Template errors now show in stderr when you run `push`
