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
