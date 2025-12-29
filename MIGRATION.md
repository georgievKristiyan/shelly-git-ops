# Migration Notes

## Cleanup Old Config Files

If you pulled configurations before this fix, you may have invalid config files that should be removed:

### Remove Cloud Config (Read-Only)

Cloud configuration is managed by Shelly Cloud and cannot be updated locally. Remove it from your device folders:

```bash
find . -name "cloud.json" -path "*/configs/*" -delete
```

### Remove Script Metadata from Configs

Script metadata should only exist in the `scripts/` folder, not in `configs/`. Remove script configs:

```bash
find . -name "script-*.json" -path "*/configs/*" -delete
```

### Verify Cleanup

After cleanup, your device folder structure should look like:

```
device-folder/
├── configs/
│   ├── sys.json
│   ├── wifi.json
│   ├── switch-0.json
│   └── ... (other components, but NOT cloud.json or script-*.json)
├── scripts/
│   ├── script-1.js
│   ├── script-1.meta.json
│   └── ...
└── ...
```

## Why These Changes?

1. **Cloud config**: The Shelly API returns error `-107: Permission denied: Only cloud can update!` when trying to push cloud configuration. Cloud settings are managed through the Shelly Cloud service and should not be modified locally.

2. **Script metadata**: Script configurations appear in the device config response as `"script:1": {...}`, but these are managed through the Script API (`Script.Create`, `Script.SetConfig`, etc.), not through component config API. Storing them in `configs/` causes errors during push because:
   - Configs are pushed before scripts are created
   - The script doesn't exist yet, causing error `-105: Argument 'id', value 1 not found!`

The tool now automatically skips these during both pull and push operations.
