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

// RenderTemplate renders a Go template string with the given values
func RenderTemplate(tmplStr string, values Values) (string, error) {
	// Create template with option to treat missing keys as errors
	tmpl, err := template.New("kvs").Option("missingkey=error").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, values); err != nil {
		// Provide more helpful error message
		return "", fmt.Errorf("failed to execute template '%s' with values: %w\nAvailable keys: %v",
			tmplStr, err, getMapKeys(values))
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

// RenderKVSValue renders a KVS value if it's a template, otherwise returns it as-is
// Returns the rendered value and whether it was templated
func RenderKVSValue(value interface{}, values Values) (interface{}, bool, error) {
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
	rendered, err := RenderTemplate(strValue, values)
	if err != nil {
		return nil, true, err
	}

	return rendered, true, nil
}
