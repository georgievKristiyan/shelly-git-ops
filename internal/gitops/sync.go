package gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/darkermage/shelly-git-ops/internal/discovery"
	"github.com/darkermage/shelly-git-ops/internal/shelly"
	"github.com/darkermage/shelly-git-ops/internal/storage"
	"golang.org/x/sync/errgroup"
)

// SyncManager orchestrates Git-ops synchronization
type SyncManager struct {
	repo          *Repository
	repoPath      string
	manifest      *storage.Manifest
	shellyClient  *shelly.Client
	deviceStorage *storage.DeviceStorage
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	DeviceID string
	Success  bool
	Error    error
	Message  string
}

// NewSyncManager creates a new sync manager
func NewSyncManager(repoPath string) (*SyncManager, error) {
	repo, err := OpenRepository(repoPath)
	if err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(repoPath, "manifest.yaml")
	manifest, err := storage.LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	return &SyncManager{
		repo:          repo,
		repoPath:      repoPath,
		manifest:      manifest,
		shellyClient:  shelly.NewClient(),
		deviceStorage: storage.NewDeviceStorage(repoPath),
	}, nil
}

// PullFromDevices fetches current state from all devices and overwrites local files
func (sm *SyncManager) PullFromDevices(ctx context.Context) ([]SyncResult, error) {
	// Safety check: ensure there are no uncommitted changes
	hasChanges, err := sm.repo.HasChanges()
	if err != nil {
		return nil, fmt.Errorf("failed to check repository status: %w", err)
	}
	if hasChanges {
		return nil, fmt.Errorf("cannot pull: working tree has uncommitted changes. Please commit or stash your changes first")
	}

	// Pull from all devices in parallel
	g, ctx := errgroup.WithContext(ctx)
	results := make([]SyncResult, len(sm.manifest.Devices))

	for i, device := range sm.manifest.Devices {
		i, device := i, device // Capture loop variables
		g.Go(func() error {
			result := sm.pullDeviceConfig(ctx, device)
			results[i] = result
			return nil // Don't fail entire operation if one device fails
		})
	}

	if err := g.Wait(); err != nil {
		return results, err
	}

	return results, nil
}

// pullDeviceConfig pulls configuration from a single device
func (sm *SyncManager) pullDeviceConfig(ctx context.Context, device storage.Device) SyncResult {
	result := SyncResult{
		DeviceID: device.DeviceID,
		Success:  false,
	}

	// Get device info
	deviceInfo, err := sm.shellyClient.GetDeviceInfo(ctx, device.IPAddress)
	if err != nil {
		result.Error = fmt.Errorf("failed to get device info: %w", err)
		return result
	}

	// Use device name from Shelly.GetDeviceInfo, fallback to manifest name if empty
	deviceName := deviceInfo.Name
	if deviceName == "" {
		deviceName = device.Name
	}

	// Check if name changed and update manifest
	if deviceName != device.Name && deviceName != "" {
		// Create new folder name based on device name (sanitize for filesystem)
		sanitizedName := strings.ToLower(deviceName)
		sanitizedName = strings.ReplaceAll(sanitizedName, " ", "-")
		sanitizedName = strings.ReplaceAll(sanitizedName, "_", "-")
		newFolderName := fmt.Sprintf("%s-%s", sanitizedName, device.DeviceID)

		// Rename folder if it exists and name changed
		if device.Folder != newFolderName && sm.deviceStorage.DeviceExists(device.Folder) {
			oldPath := sm.deviceStorage.GetDevicePath(device.Folder)
			newPath := sm.deviceStorage.GetDevicePath(newFolderName)

			if err := os.Rename(oldPath, newPath); err != nil {
				result.Error = fmt.Errorf("failed to rename device folder: %w", err)
				return result
			}

			// Update device folder in local variable
			device.Folder = newFolderName
		}

		// Update manifest
		device.Name = deviceName
		device.Folder = newFolderName
		sm.manifest.AddDevice(device)
		if err := sm.manifest.Save(); err != nil {
			result.Error = fmt.Errorf("failed to update manifest: %w", err)
			return result
		}
	}

	// Ensure device folder and all subdirectories exist
	// Call this AFTER rename logic to ensure subdirs exist in the correct location
	// Always create/ensure folder structure exists (MkdirAll is safe to call multiple times)
	if err := sm.deviceStorage.CreateDeviceFolder(device.Folder); err != nil {
		result.Error = fmt.Errorf("failed to create device folder: %w", err)
		return result
	}

	// Save device metadata
	metadata := storage.DeviceMetadata{
		DeviceID:   device.DeviceID,
		Name:       deviceName,
		Model:      deviceInfo.Model,
		Firmware:   deviceInfo.FW,
		IPAddress:  device.IPAddress,
		MACAddress: device.MACAddress,
	}
	if err := sm.deviceStorage.SaveDeviceMetadata(device.Folder, metadata); err != nil {
		result.Error = fmt.Errorf("failed to save metadata: %w", err)
		return result
	}

	// Get all component configurations using Shelly.GetConfig
	shellyConfig, err := sm.shellyClient.GetShellyConfig(ctx, device.IPAddress)
	if err != nil {
		result.Error = fmt.Errorf("failed to get shelly config: %w", err)
		return result
	}

	// Parse the config as a map to extract individual components
	var configMap map[string]json.RawMessage
	if err := json.Unmarshal(shellyConfig, &configMap); err != nil {
		result.Error = fmt.Errorf("failed to parse shelly config: %w", err)
		return result
	}

	// Save each component configuration separately
	configCount := 0
	for componentKey, componentConfig := range configMap {
		// Skip cloud config (read-only, only cloud can update)
		if componentKey == "cloud" {
			continue
		}

		// Skip script configs (managed separately in scripts folder)
		if strings.HasPrefix(componentKey, "script:") {
			continue
		}

		// componentKey format: "switch:0", "input:1", "sys", "wifi", etc.
		// Convert to filename: "switch-0.json", "input-1.json", "sys.json", "wifi.json"
		filename := strings.ReplaceAll(componentKey, ":", "-")

		// Save component config
		if err := sm.deviceStorage.SaveComponentConfig(device.Folder, filename, componentConfig); err != nil {
			result.Error = fmt.Errorf("failed to save %s config: %w", filename, err)
			return result
		}

		configCount++
	}

	// Get and save scripts
	scripts, err := sm.shellyClient.ListScripts(ctx, device.IPAddress)
	scriptCount := 0
	if err == nil {
		for _, script := range scripts {
			code, err := sm.shellyClient.GetScriptCode(ctx, device.IPAddress, script.ID)
			if err != nil {
				continue
			}

			scriptCode := &shelly.ScriptCode{
				ID:   script.ID,
				Name: script.Name,
				Code: code,
			}

			if err := sm.deviceStorage.SaveScript(device.Folder, scriptCode, script.Enable); err != nil {
				continue
			}
			scriptCount++
		}
	}

	// Get and save schedules
	schedules, err := sm.shellyClient.ListSchedules(ctx, device.IPAddress)
	scheduleCount := 0
	if err == nil {
		for _, schedule := range schedules {
			if err := sm.deviceStorage.SaveSchedule(device.Folder, &schedule); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to save schedule %d for %s: %v\n", schedule.ID, device.Name, err)
				continue
			}
			scheduleCount++
		}
	} else {
		// Log warning but don't fail - schedules might not be supported on this device
		fmt.Fprintf(os.Stderr, "Warning: Failed to list schedules for %s: %v\n", device.Name, err)
	}

	// Get and save webhooks
	webhooks, err := sm.shellyClient.ListWebhooks(ctx, device.IPAddress)
	webhookCount := 0
	if err == nil {
		for _, webhook := range webhooks {
			if err := sm.deviceStorage.SaveWebhook(device.Folder, &webhook); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to save webhook %d for %s: %v\n", webhook.ID, device.Name, err)
				continue
			}
			webhookCount++
		}
	} else {
		// Log warning but don't fail - webhooks might not be supported on this device
		fmt.Fprintf(os.Stderr, "Warning: Failed to list webhooks for %s: %v\n", device.Name, err)
	}

	// Get and save KVS (Key-Value Store) data
	kvsData, err := sm.shellyClient.GetKVS(ctx, device.IPAddress)
	kvsCount := 0
	if err == nil && len(kvsData) > 0 {
		// Load existing local KVS to preserve templates
		existingKVS, _ := sm.deviceStorage.LoadKVS(device.Folder)

		// Start with existing local KVS to preserve all keys (including templates)
		mergedKVS := make(map[string]interface{})
		for key, value := range existingKVS {
			mergedKVS[key] = value
		}

		// Update with device values, but only if local value is NOT templated
		for key, deviceValue := range kvsData {
			if existingValue, exists := existingKVS[key]; exists {
				// Check if the existing local value is templated
				if strValue, ok := existingValue.(string); ok && IsTemplated(strValue) {
					// Skip - preserve the template, don't overwrite
					continue
				}
			}
			// Not templated or doesn't exist locally, update with device value
			mergedKVS[key] = deviceValue
		}

		if err := sm.deviceStorage.SaveKVS(device.Folder, mergedKVS); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save KVS data for %s: %v\n", device.Name, err)
		} else {
			kvsCount = len(mergedKVS)
		}
	} else if err != nil {
		// Log warning but don't fail - KVS might not be supported on this device
		fmt.Fprintf(os.Stderr, "Warning: Failed to get KVS data for %s: %v\n", device.Name, err)
	}

	// Get all components (including virtual components and groups)
	components, err := sm.shellyClient.GetComponents(ctx, device.IPAddress)
	virtualComponentCount := 0
	groupCount := 0
	if err == nil {
		for _, component := range components {
			// Parse component key (e.g., "boolean:200", "number:201", "group:200")
			parts := strings.SplitN(component.Key, ":", 2)
			if len(parts) != 2 {
				// Skip components without ID (e.g., "cloud", "mqtt", "sys")
				continue
			}

			componentType := parts[0]
			componentIDStr := parts[1]

			// Check if this is a virtual component or group
			// Virtual components typically have IDs >= 200 and include:
			// boolean, number, text, enum, button, group
			isVirtualComponent := componentType == "boolean" || componentType == "number" ||
				componentType == "text" || componentType == "enum" || componentType == "button"
			isGroup := componentType == "group"

			if !isVirtualComponent && !isGroup {
				// Skip non-virtual components (like input, switch, cover, etc.)
				continue
			}

			// Parse component ID
			componentID, err := strconv.Atoi(componentIDStr)
			if err != nil {
				// Skip if ID is not a number
				fmt.Fprintf(os.Stderr, "Warning: Invalid component ID for %s on %s: %v\n", component.Key, device.Name, err)
				continue
			}

			// Marshal the entire component (including status and config)
			componentData, err := json.Marshal(component)
			if err != nil {
				// Skip if marshaling fails
				fmt.Fprintf(os.Stderr, "Warning: Failed to marshal component %s for %s: %v\n", component.Key, device.Name, err)
				continue
			}

			if isGroup {
				// Save as a group
				group := shelly.Group{
					ID:   componentID,
					Name: "", // Will be in the config
					Type: componentType,
				}
				if err := sm.deviceStorage.SaveGroup(device.Folder, &group); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to save group %s for %s: %v\n", component.Key, device.Name, err)
					continue
				}
				// Also save the full component data
				if err := sm.deviceStorage.SaveVirtualComponent(device.Folder, componentType, componentID, componentData); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to save virtual component data for group %s on %s: %v\n", component.Key, device.Name, err)
					continue
				}
				groupCount++
			} else {
				// Save as a virtual component
				if err := sm.deviceStorage.SaveVirtualComponent(device.Folder, componentType, componentID, componentData); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to save virtual component %s for %s: %v\n", component.Key, device.Name, err)
					continue
				}
				virtualComponentCount++
			}
		}
	} else {
		// Log warning but don't fail - virtual components might not be supported on this device
		fmt.Fprintf(os.Stderr, "Warning: Failed to get components for %s: %v\n", device.Name, err)
	}

	// Update last sync time
	sm.manifest.UpdateLastSync(device.DeviceID, time.Now())

	result.Success = true

	// Build success message
	var msgParts []string
	if configCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d config(s)", configCount))
	}
	if scriptCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d script(s)", scriptCount))
	}
	if scheduleCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d schedule(s)", scheduleCount))
	}
	if webhookCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d webhook(s)", webhookCount))
	}
	if kvsCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d KVS item(s)", kvsCount))
	}

	// Add virtual component and group counts
	if virtualComponentCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d virtual component(s)", virtualComponentCount))
	}
	if groupCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d group(s)", groupCount))
	}

	if len(msgParts) > 0 {
		result.Message = fmt.Sprintf("saved %s", strings.Join(msgParts, ", "))
	} else {
		result.Message = "synced device"
	}

	return result
}

// PushToDevices applies current local configuration to devices
// If deviceFilter is empty, pushes to all devices
// If deviceFilter is provided, only pushes to devices matching the filter (by ID or name)
// If valuesFile is provided, it will be used for templating KVS values
func (sm *SyncManager) PushToDevices(ctx context.Context, dryRun bool, deviceFilter []string, valuesFile string) ([]SyncResult, error) {
	// Load values file if provided
	values, err := LoadValuesFile(valuesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load values file: %w", err)
	}

	// Build allDevices map for template context
	allDevices := make(map[string]DeviceContext)
	for _, device := range sm.manifest.Devices {
		allDevices[device.DeviceID] = DeviceContext{
			DeviceID:   device.DeviceID,
			Name:       device.Name,
			Model:      device.Model,
			IPAddress:  device.IPAddress,
			MACAddress: device.MACAddress,
			Folder:     device.Folder,
		}
	}

	// Filter devices if a filter is provided
	devicesToPush := sm.manifest.Devices
	if len(deviceFilter) > 0 {
		// Create a map for quick lookup
		filterMap := make(map[string]bool)
		for _, f := range deviceFilter {
			filterMap[strings.ToLower(f)] = true
		}

		// Filter devices
		var filtered []storage.Device
		for _, device := range sm.manifest.Devices {
			// Match by device ID or name (case-insensitive)
			if filterMap[strings.ToLower(device.DeviceID)] || filterMap[strings.ToLower(device.Name)] {
				filtered = append(filtered, device)
			}
		}
		devicesToPush = filtered
	}

	// Push to filtered devices in parallel
	g, ctx := errgroup.WithContext(ctx)
	results := make([]SyncResult, len(devicesToPush))

	for i, device := range devicesToPush {
		i, device := i, device
		g.Go(func() error {
			result := sm.pushDeviceConfig(ctx, device, dryRun, values, allDevices)
			results[i] = result
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return results, err
	}

	return results, nil
}

// pushDeviceConfig pushes configuration to a single device
func (sm *SyncManager) pushDeviceConfig(ctx context.Context, device storage.Device, dryRun bool, values Values, allDevices map[string]DeviceContext) SyncResult {
	result := SyncResult{
		DeviceID: device.DeviceID,
		Success:  false,
	}

	if !sm.deviceStorage.DeviceExists(device.Folder) {
		result.Error = fmt.Errorf("device folder does not exist")
		return result
	}

	if dryRun {
		result.Success = true
		result.Message = "dry-run: would push all configurations"
		return result
	}

	// Create template context with device information
	currentDevice := DeviceContext{
		DeviceID:   device.DeviceID,
		Name:       device.Name,
		Model:      device.Model,
		IPAddress:  device.IPAddress,
		MACAddress: device.MACAddress,
		Folder:     device.Folder,
	}
	templateContext := CreateTemplateContext(values, currentDevice, allDevices)

	// Push component configs
	componentFiles, err := sm.deviceStorage.ListComponentConfigs(device.Folder)
	if err != nil {
		result.Error = fmt.Errorf("failed to list component configs: %w", err)
		return result
	}

	configCount := 0
	for _, componentFile := range componentFiles {
		// Skip cloud config (read-only, only cloud can update)
		if componentFile == "cloud" {
			continue
		}

		// Skip script configs (managed separately in scripts folder)
		if strings.HasPrefix(componentFile, "script-") {
			continue
		}

		configData, err := sm.deviceStorage.LoadComponentConfig(device.Folder, componentFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to load config %s: %v\n", componentFile, err)
			continue
		}

		// Parse config
		var config map[string]interface{}
		if err := json.Unmarshal(configData, &config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to parse config %s: %v\n", componentFile, err)
			continue
		}

		// Parse component filename: "switch-0" -> component="Switch", id=0
		// or "sys" -> component="Sys", id=-1 (no id)
		var componentName string
		var componentID int = -1

		if strings.Contains(componentFile, "-") {
			// Has ID: "switch-0", "input-1"
			parts := strings.SplitN(componentFile, "-", 2)
			componentName = strings.Title(parts[0])
			fmt.Sscanf(parts[1], "%d", &componentID)
		} else {
			// No ID: "sys", "wifi", "cloud"
			componentName = strings.Title(componentFile)
		}

		// Build params for SetConfig
		var params map[string]interface{}
		if componentID >= 0 {
			// Component with ID: {"id": 0, "config": {...}}
			params = map[string]interface{}{
				"id":     componentID,
				"config": config,
			}
		} else {
			// Component without ID: {"config": {...}}
			params = map[string]interface{}{
				"config": config,
			}
		}

		// Apply config
		if err := sm.shellyClient.SetComponentConfig(ctx, device.IPAddress, componentName, params); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to set %s config: %v\n", componentFile, err)
			continue
		}

		configCount++
	}

	// Push scripts
	scripts, err := sm.deviceStorage.ListScripts(device.Folder)
	if err != nil {
		// If scripts directory doesn't exist, that's OK - just skip scripts
		scripts = []storage.ScriptMetadata{}
	}

	// Get device scripts once for comparison
	deviceScripts, err := sm.shellyClient.ListScripts(ctx, device.IPAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to list device scripts: %v\n", err)
		deviceScripts = []shelly.Script{} // Continue with empty list
	}

	scriptCount := 0
	for _, scriptMeta := range scripts {
		code, err := sm.deviceStorage.LoadScript(device.Folder, scriptMeta.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to load script %d: %v\n", scriptMeta.ID, err)
			continue
		}

		// Check if script exists on device
		var existingScript *shelly.Script
		scriptExists := false
		for _, ds := range deviceScripts {
			if ds.ID == scriptMeta.ID {
				scriptExists = true
				existingScript = &ds
				break
			}
		}

		if !scriptExists {
			// Create script
			id, err := sm.shellyClient.CreateScript(ctx, device.IPAddress, scriptMeta.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to create script %s: %v\n", scriptMeta.Name, err)
				continue
			}
			scriptMeta.ID = id
		} else if existingScript.Running {
			// Script is running, stop it before uploading
			if err := sm.shellyClient.StopScript(ctx, device.IPAddress, scriptMeta.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to stop running script %d: %v\n", scriptMeta.ID, err)
				continue
			}
		}

		// Upload script code
		if err := sm.shellyClient.PutScriptCode(ctx, device.IPAddress, scriptMeta.ID, code, false); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to upload script %d: %v\n", scriptMeta.ID, err)
			continue
		}

		// Set script config (name and enable state from metadata)
		if err := sm.shellyClient.SetScriptConfig(ctx, device.IPAddress, scriptMeta.ID, scriptMeta.Name, scriptMeta.Enable); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to set script %d config: %v\n", scriptMeta.ID, err)
			continue
		}

		// Start script if it should be enabled
		if scriptMeta.Enable {
			if err := sm.shellyClient.StartScript(ctx, device.IPAddress, scriptMeta.ID); err != nil {
				// Don't fail the whole operation if start fails, just log it
				fmt.Fprintf(os.Stderr, "Warning: Uploaded script %d but failed to start: %v\n", scriptMeta.ID, err)
			}
		}

		fmt.Fprintf(os.Stderr, "Info: Pushed script %d (%s)\n", scriptMeta.ID, scriptMeta.Name)
		scriptCount++
	}

	// Push schedules
	localSchedules, err := sm.deviceStorage.ListSchedules(device.Folder)
	if err != nil {
		// If schedules directory doesn't exist, that's OK - just skip schedules
		localSchedules = []*shelly.Schedule{}
	}

	// Get device schedules for comparison
	deviceSchedules, err := sm.shellyClient.ListSchedules(ctx, device.IPAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to list device schedules: %v\n", err)
		deviceSchedules = []shelly.Schedule{}
	}

	// Create maps for easier lookup
	deviceScheduleMap := make(map[int]shelly.Schedule)
	for _, ds := range deviceSchedules {
		deviceScheduleMap[ds.ID] = ds
	}

	localScheduleMap := make(map[int]*shelly.Schedule)
	for _, ls := range localSchedules {
		localScheduleMap[ls.ID] = ls
	}

	scheduleCount := 0

	// Update or create schedules from local files
	for _, localSchedule := range localSchedules {
		if _, exists := deviceScheduleMap[localSchedule.ID]; exists {
			// Update existing schedule
			if err := sm.shellyClient.UpdateSchedule(ctx, device.IPAddress, *localSchedule); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to update schedule %d: %v\n", localSchedule.ID, err)
				continue
			}
		} else {
			// Create new schedule
			if _, err := sm.shellyClient.CreateSchedule(ctx, device.IPAddress, *localSchedule); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to create schedule: %v\n", err)
				continue
			}
		}
		scheduleCount++
	}

	// Delete schedules that don't exist locally
	for _, deviceSchedule := range deviceSchedules {
		if _, exists := localScheduleMap[deviceSchedule.ID]; !exists {
			if err := sm.shellyClient.DeleteSchedule(ctx, device.IPAddress, deviceSchedule.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to delete schedule %d: %v\n", deviceSchedule.ID, err)
			}
		}
	}

	// Push webhooks
	localWebhooks, err := sm.deviceStorage.ListWebhooks(device.Folder)
	if err != nil {
		// If webhooks directory doesn't exist, that's OK - just skip webhooks
		localWebhooks = []*shelly.Webhook{}
	}

	// Get device webhooks for comparison
	deviceWebhooks, err := sm.shellyClient.ListWebhooks(ctx, device.IPAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to list device webhooks: %v\n", err)
		deviceWebhooks = []shelly.Webhook{}
	}

	// Create maps for easier lookup
	deviceWebhookMap := make(map[int]shelly.Webhook)
	for _, dw := range deviceWebhooks {
		deviceWebhookMap[dw.ID] = dw
	}

	localWebhookMap := make(map[int]*shelly.Webhook)
	for _, lw := range localWebhooks {
		localWebhookMap[lw.ID] = lw
	}

	webhookCount := 0

	// Update or create webhooks from local files
	for _, localWebhook := range localWebhooks {
		if _, exists := deviceWebhookMap[localWebhook.ID]; exists {
			// Update existing webhook
			if err := sm.shellyClient.UpdateWebhook(ctx, device.IPAddress, *localWebhook); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to update webhook %d: %v\n", localWebhook.ID, err)
				continue
			}
		} else {
			// Create new webhook
			if _, err := sm.shellyClient.CreateWebhook(ctx, device.IPAddress, *localWebhook); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to create webhook: %v\n", err)
				continue
			}
		}
		webhookCount++
	}

	// Delete webhooks that don't exist locally
	for _, deviceWebhook := range deviceWebhooks {
		if _, exists := localWebhookMap[deviceWebhook.ID]; !exists {
			if err := sm.shellyClient.DeleteWebhook(ctx, device.IPAddress, deviceWebhook.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to delete webhook %d: %v\n", deviceWebhook.ID, err)
			}
		}
	}

	// Push KVS (Key-Value Store) data
	localKVS, err := sm.deviceStorage.LoadKVS(device.Folder)
	kvsCount := 0
	if err == nil && len(localKVS) > 0 {
		// Get current KVS data from device for comparison
		deviceKVS, err := sm.shellyClient.GetKVS(ctx, device.IPAddress)
		if err != nil {
			// KVS might not be supported on this device, skip silently
			deviceKVS = make(map[string]interface{})
		}

		// Set or update keys from local KVS (with template rendering)
		for key, value := range localKVS {
			// Render template if value is templated
			renderedValue, wasTemplated, err := RenderKVSValue(value, templateContext)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to render template for KVS key %s: %v\n", key, err)
				continue
			}

			// Use rendered value for push
			if err := sm.shellyClient.SetKVS(ctx, device.IPAddress, key, renderedValue); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to set KVS key %s: %v\n", key, err)
				continue
			}

			if wasTemplated {
				fmt.Fprintf(os.Stderr, "Info: Rendered template for KVS key %s\n", key)
			}

			kvsCount++
		}

		// Delete keys that exist on device but not locally
		for key := range deviceKVS {
			if _, exists := localKVS[key]; !exists {
				if err := sm.shellyClient.DeleteKVS(ctx, device.IPAddress, key); err != nil {
					result.Message = fmt.Sprintf("failed to delete KVS key %s: %v", key, err)
				}
			}
		}
	}

	result.Success = true

	// Build success message
	var msgParts []string
	if configCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d config(s)", configCount))
	}
	if scriptCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d script(s)", scriptCount))
	}
	if scheduleCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d schedule(s)", scheduleCount))
	}
	if webhookCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d webhook(s)", webhookCount))
	}
	if kvsCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d KVS item(s)", kvsCount))
	}

	if len(msgParts) > 0 {
		result.Message = fmt.Sprintf("pushed %s", strings.Join(msgParts, ", "))
	} else {
		result.Message = "pushed to device"
	}

	return result
}

// DiscoverAndAdd discovers devices and adds them to the manifest
func (sm *SyncManager) DiscoverAndAdd(ctx context.Context, provider discovery.Provider, filterPattern string) ([]storage.Device, error) {
	devices, err := provider.DiscoverDevices(ctx, filterPattern)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	var addedDevices []storage.Device

	for _, deviceInfo := range devices {
		// Only add Shelly devices
		if !deviceInfo.IsShelly() {
			continue
		}

		// Check if device already exists
		if sm.manifest.GetDeviceByIP(deviceInfo.IPAddress) != nil {
			continue
		}

		// Get device info from Shelly API
		shellyInfo, err := sm.shellyClient.GetDeviceInfo(ctx, deviceInfo.IPAddress)
		if err != nil {
			// Skip devices we can't communicate with
			continue
		}

		// Create device entry
		deviceName := strings.ToLower(deviceInfo.Hostname)
		folderName := fmt.Sprintf("%s-%s", deviceName, shellyInfo.ID)

		device := storage.Device{
			DeviceID:   shellyInfo.ID,
			Name:       deviceName,
			Folder:     folderName,
			IPAddress:  deviceInfo.IPAddress,
			MACAddress: deviceInfo.MACAddress,
			Model:      shellyInfo.Model,
			LastSync:   time.Now(),
		}

		// Add to manifest
		sm.manifest.AddDevice(device)

		// Create device folder
		sm.deviceStorage.CreateDeviceFolder(folderName)

		// Pull initial configuration
		sm.pullDeviceConfig(ctx, device)

		addedDevices = append(addedDevices, device)
	}

	// Save manifest
	if err := sm.manifest.Save(); err != nil {
		return addedDevices, fmt.Errorf("failed to save manifest: %w", err)
	}

	return addedDevices, nil
}
