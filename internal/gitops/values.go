package gitops

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Values represents the values loaded from values file
type Values map[string]interface{}

// LoadValuesFile loads values from a YAML file
func LoadValuesFile(path string) (Values, error) {
	if path == "" {
		return make(Values), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file: %w", err)
	}

	var values Values
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values file: %w", err)
	}

	return values, nil
}

// RenderTemplate renders a Go template string with the given context
func RenderTemplate(tmplStr string, context map[string]interface{}) (string, error) {
	// Create template with option to treat missing keys as errors
	tmpl, err := template.New("kvs").Option("missingkey=error").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, context); err != nil {
		// Provide more helpful error message
		return "", fmt.Errorf("failed to execute template '%s' with context: %w\nAvailable keys: %v",
			tmplStr, err, getMapKeys(context))
	}

	return buf.String(), nil
}

// getMapKeys returns all keys from a map for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// IsTemplated checks if a string contains Go template syntax
func IsTemplated(value string) bool {
	// Check for Go template delimiters {{ }}
	templatePattern := regexp.MustCompile(`\{\{.*?\}\}`)
	return templatePattern.MatchString(value)
}

// DeviceContext represents device information available in templates
type DeviceContext struct {
	DeviceID   string `yaml:"device_id"`
	Name       string `yaml:"name"`
	Model      string `yaml:"model"`
	IPAddress  string `yaml:"ip_address"`
	MACAddress string `yaml:"mac_address"`
	Folder     string `yaml:"folder"`
}

// TemplateContext combines values and device information for template rendering
type TemplateContext struct {
	Values  Values                    // User-provided values from values.yaml
	Device  DeviceContext             // Current device being processed
	Devices map[string]DeviceContext  // All devices by device ID
}

// CreateTemplateContext creates a template context with values and device info
func CreateTemplateContext(values Values, currentDevice DeviceContext, allDevices map[string]DeviceContext) map[string]interface{} {
	context := make(map[string]interface{})

	// Add all values from values.yaml at the root level
	for k, v := range values {
		context[k] = v
	}

	// Add device-specific context
	context["device"] = map[string]interface{}{
		"device_id":   currentDevice.DeviceID,
		"name":        currentDevice.Name,
		"model":       currentDevice.Model,
		"ip_address":  currentDevice.IPAddress,
		"mac_address": currentDevice.MACAddress,
		"folder":      currentDevice.Folder,
	}

	// Add all devices map
	devicesMap := make(map[string]interface{})
	for deviceID, device := range allDevices {
		devicesMap[deviceID] = map[string]interface{}{
			"device_id":   device.DeviceID,
			"name":        device.Name,
			"model":       device.Model,
			"ip_address":  device.IPAddress,
			"mac_address": device.MACAddress,
			"folder":      device.Folder,
		}
	}
	context["devices"] = devicesMap

	return context
}

// RenderKVSValue renders a KVS value if it's a template, otherwise returns it as-is
// Returns the rendered value and whether it was templated
func RenderKVSValue(value interface{}, context map[string]interface{}) (interface{}, bool, error) {
	// Only process string values
	strValue, ok := value.(string)
	if !ok {
		return value, false, nil
	}

	// Check if it's templated
	if !IsTemplated(strValue) {
		return value, false, nil
	}

	// Render the template
	rendered, err := RenderTemplate(strValue, context)
	if err != nil {
		return nil, true, err
	}

	return rendered, true, nil
}
