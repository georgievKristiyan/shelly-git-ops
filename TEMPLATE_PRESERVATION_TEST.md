# Template Preservation Test

This document explains how template preservation works during pull operations.

## Scenario

You have a local KVS file with templates:

```json
{
  "api_endpoint": "{{ .api.endpoint }}",
  "temperature": "{{ .thresholds.temp }}",
  "device_name": "My Device",
  "update_interval": "300"
}
```

On the device, the actual values are:
```json
{
  "api_endpoint": "https://api.example.com",
  "temperature": "22.5",
  "device_name": "My Device",
  "update_interval": "600",
  "new_key": "some value"
}
```

## After Pull

The local file will be:

```json
{
  "api_endpoint": "{{ .api.endpoint }}",     ← PRESERVED (templated)
  "temperature": "{{ .thresholds.temp }}",   ← PRESERVED (templated)
  "device_name": "My Device",                ← KEPT (not templated, same value)
  "update_interval": "600",                  ← UPDATED (not templated, changed)
  "new_key": "some value"                    ← ADDED (new key from device)
}
```

## How It Works

1. **Templated values are NEVER overwritten**
   - If a local value contains `{{ }}`, it's preserved as-is
   - The actual value from the device is ignored for that key

2. **Non-templated values are updated**
   - If a local value is NOT templated, it gets updated from the device
   - This allows you to pull configuration changes

3. **New keys are added**
   - If a key exists on the device but not locally, it's added
   - This captures new KVS keys created on the device

4. **Local-only templated keys are preserved**
   - If a templated key exists locally but not on device, it's kept
   - This is useful for planned deployments

## Testing

To test template preservation:

1. Create a device KVS file with templates:
   ```bash
   mkdir -p test-device/kvs
   cat > test-device/kvs/data.json <<EOF
   {
     "templated_key": "{{ .myValue }}",
     "normal_key": "original value"
   }
   EOF
   ```

2. Pull from a device:
   ```bash
   ./shelly-gitops pull
   ```

3. Verify the results:
   - `templated_key` should still be `"{{ .myValue }}"`
   - `normal_key` might be updated if it changed on the device
   - Any new keys from the device are added

## Why This Matters

Template preservation allows you to:
- **Keep configuration as code** - Templates stay in version control
- **Pull without fear** - Running pull won't destroy your templates
- **Update selectively** - Non-templated values update, templates don't
- **Manage secrets** - Keep sensitive values templated, not in the repo
