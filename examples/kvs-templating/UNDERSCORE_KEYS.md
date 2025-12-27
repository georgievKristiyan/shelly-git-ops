# Working with Underscore Keys in Templates

## The Problem

When using keys with underscores in Go templates, the simple dot notation may not work reliably:

**values.yaml:**
```yaml
test_key: "test_value3"
```

**kvs/data.json (PROBLEMATIC):**
```json
{
  "test1": "{{ .test_key }}"
}
```

**Result:** `<no value>` or template execution error

## The Solution

Use the `index` function to access map keys with underscores, hyphens, or other special characters:

**kvs/data.json (WORKING):**
```json
{
  "test1": "{{ index . \"test_key\" }}"
}
```

**Result:** `"test_value3"` ✓

## Why This Happens

Go's `text/template` package uses dot notation (`.key`) for accessing struct fields. When working with maps:

- **CamelCase keys** work fine: `{{ .testKey }}`
- **Keys with underscores** may fail: `{{ .test_key }}`
- **Keys with hyphens** will fail: `{{ .my-key }}`
- **Keys with dots** will fail: `{{ .my.key }}`

The `index` function provides a reliable way to access any map key regardless of its name.

## Complete Examples

### Example 1: Simple Keys

**values.yaml:**
```yaml
simple_key: "value1"
another_key: "value2"
normalKey: "value3"
```

**kvs/data.json:**
```json
{
  "val1": "{{ index . \"simple_key\" }}",
  "val2": "{{ index . \"another_key\" }}",
  "val3": "{{ .normalKey }}"
}
```

**Pushed to device:**
```json
{
  "val1": "value1",
  "val2": "value2",
  "val3": "value3"
}
```

### Example 2: Nested Values

**values.yaml:**
```yaml
api:
  endpoint_url: "https://api.example.com"
  api_key: "secret123"
database:
  host: "db.example.com"
```

**kvs/data.json:**
```json
{
  "api_url": "{{ index .api \"endpoint_url\" }}",
  "api_secret": "{{ index .api \"api_key\" }}",
  "db_host": "{{ index .database \"host\" }}"
}
```

**Pushed to device:**
```json
{
  "api_url": "https://api.example.com",
  "api_secret": "secret123",
  "db_host": "db.example.com"
}
```

### Example 3: Mixed Notation

**values.yaml:**
```yaml
environment: production
api:
  endpoint: "https://api.example.com"
  timeout: 30
feature_flags:
  enable_debug: true
```

**kvs/data.json:**
```json
{
  "env": "{{ .environment }}",
  "api_url": "{{ .api.endpoint }}",
  "timeout": "{{ .api.timeout }}",
  "debug": "{{ index .feature_flags \"enable_debug\" }}"
}
```

## Quick Reference

| Key Name | Dot Notation | Index Function |
|----------|--------------|----------------|
| `simpleKey` | `{{ .simpleKey }}` ✓ | `{{ index . "simpleKey" }}` ✓ |
| `simple_key` | `{{ .simple_key }}` ⚠️ | `{{ index . "simple_key" }}` ✓ |
| `simple-key` | `{{ .simple-key }}` ✗ | `{{ index . "simple-key" }}` ✓ |
| `api.endpoint` | `{{ .api.endpoint }}` ✓ | `{{ index .api "endpoint" }}` ✓ |
| `api.my_key` | `{{ .api.my_key }}` ⚠️ | `{{ index .api "my_key" }}` ✓ |

Legend:
- ✓ Works reliably
- ⚠️ May have issues
- ✗ Will not work

## Recommendation

**Always use the `index` function for consistency and reliability:**

```json
{
  "value": "{{ index . \"key_name\" }}"
}
```

This works for all key names and avoids surprises with special characters.

## Testing Your Templates

To test if your templates render correctly:

1. Run with `--dry-run` first:
   ```bash
   ./shelly-gitops push --values values.yaml --dry-run
   ```

2. Check the info messages:
   ```
   Info: Rendered template for KVS key test1
   ```

3. If you see errors, check:
   - Is the key name spelled correctly in values.yaml?
   - Are you using `index` for keys with underscores/special chars?
   - Is the template syntax correct?

4. With the improved error messages, you'll see:
   ```
   failed to render template for KVS key test1: ...
   Available keys: [test_key, api, settings]
   ```

This helps you identify if the key exists or if there's a syntax issue.
