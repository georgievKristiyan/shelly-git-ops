# KVS Templating Example

This example demonstrates how to use values files for templating KVS values.

## Files

- `values.yaml` - Contains the actual values to be substituted
- `device-example/kvs/data.json` - KVS file with template syntax

## Example KVS with Templates

**kvs/data.json:**
```json
{
  "api_endpoint": "{{ .api.endpoint }}",
  "api_key": "{{ .api.key }}",
  "update_interval": "{{ .settings.updateInterval }}",
  "temperature_threshold": "{{ .thresholds.temperature }}",
  "environment": "{{ .environment }}",
  "static_device_name": "My Shelly Device"
}
```

**values.yaml:**
```yaml
environment: production
api:
  endpoint: https://api.example.com/v1
  key: your-secret-api-key-here
settings:
  updateInterval: 300
thresholds:
  temperature: 22.5
```

## What Gets Pushed

When you run:
```bash
shelly-gitops push --values values.yaml
```

The rendered values pushed to the device will be:
```json
{
  "api_endpoint": "https://api.example.com/v1",
  "api_key": "your-secret-api-key-here",
  "update_interval": "300",
  "temperature_threshold": "22.5",
  "environment": "production",
  "static_device_name": "My Shelly Device"
}
```

## Pull Behavior

When you run `shelly-gitops pull`, the tool will:
- Keep `"api_endpoint": "{{ .api.endpoint }}"` (templated - preserved)
- Keep `"api_key": "{{ .api.key }}"` (templated - preserved)
- Update `"static_device_name"` if it changed on the device (not templated)

This allows you to:
1. Keep sensitive values like API keys in a separate values file
2. Use different values for different environments
3. Version control templates without exposing secrets
