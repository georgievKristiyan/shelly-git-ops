package shelly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles Shelly RPC API communication
type Client struct {
	httpClient *http.Client
	auth       *AuthConfig
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Username string
	Password string
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	ID     int         `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
	Auth   *RPCAuth    `json:"auth,omitempty"`
}

// RPCAuth represents RPC authentication
type RPCAuth struct {
	Realm    string `json:"realm"`
	Username string `json:"username"`
	Nonce    int64  `json:"nonce"`
	CNonce   int64  `json:"cnonce"`
	Response string `json:"response"`
	Algorithm string `json:"algorithm"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	ID     int             `json:"id"`
	Src    string          `json:"src"`
	Dst    string          `json:"dst"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

// RPCError represents an RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewClient creates a new Shelly API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetAuth sets authentication credentials
func (c *Client) SetAuth(username, password string) {
	c.auth = &AuthConfig{
		Username: username,
		Password: password,
	}
}

// Call executes an RPC call to a Shelly device
func (c *Client) Call(ctx context.Context, deviceIP, method string, params interface{}) (json.RawMessage, error) {
	req := RPCRequest{
		ID:     1,
		Method: method,
		Params: params,
	}

	// TODO: Implement digest authentication if needed
	// if c.auth != nil {
	//     req.Auth = c.buildAuth(method)
	// }

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s/rpc", deviceIP)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// GetDeviceInfo retrieves device information
func (c *Client) GetDeviceInfo(ctx context.Context, deviceIP string) (*DeviceInfo, error) {
	result, err := c.Call(ctx, deviceIP, "Shelly.GetDeviceInfo", nil)
	if err != nil {
		return nil, err
	}

	var info DeviceInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
	}

	return &info, nil
}

// ListMethods retrieves all available RPC methods
func (c *Client) ListMethods(ctx context.Context, deviceIP string) ([]string, error) {
	result, err := c.Call(ctx, deviceIP, "Shelly.ListMethods", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Methods []string `json:"methods"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal methods: %w", err)
	}

	return response.Methods, nil
}

// GetComponentConfig retrieves configuration for a specific component
func (c *Client) GetComponentConfig(ctx context.Context, deviceIP, component string) (json.RawMessage, error) {
	method := component + ".GetConfig"
	return c.Call(ctx, deviceIP, method, nil)
}

// SetComponentConfig sets configuration for a specific component
func (c *Client) SetComponentConfig(ctx context.Context, deviceIP, component string, config interface{}) error {
	method := component + ".SetConfig"
	_, err := c.Call(ctx, deviceIP, method, config)
	return err
}

// GetConfig retrieves system-level device configuration
func (c *Client) GetConfig(ctx context.Context, deviceIP string) (json.RawMessage, error) {
	return c.Call(ctx, deviceIP, "Sys.GetConfig", nil)
}

// GetShellyConfig retrieves component-level device configuration
func (c *Client) GetShellyConfig(ctx context.Context, deviceIP string) (json.RawMessage, error) {
	return c.Call(ctx, deviceIP, "Shelly.GetConfig", nil)
}

// SetConfig sets device configuration
func (c *Client) SetConfig(ctx context.Context, deviceIP string, config map[string]interface{}) error {
	_, err := c.Call(ctx, deviceIP, "Sys.SetConfig", map[string]interface{}{"config": config})
	return err
}

// ListScripts retrieves all scripts from a device
func (c *Client) ListScripts(ctx context.Context, deviceIP string) ([]Script, error) {
	result, err := c.Call(ctx, deviceIP, "Script.List", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Scripts []Script `json:"scripts"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scripts: %w", err)
	}

	return response.Scripts, nil
}

// GetScriptCode retrieves the code for a specific script
func (c *Client) GetScriptCode(ctx context.Context, deviceIP string, scriptID int) (string, error) {
	result, err := c.Call(ctx, deviceIP, "Script.GetCode", map[string]interface{}{"id": scriptID})
	if err != nil {
		return "", err
	}

	// The API returns: {"data": "script code here", "left": remaining_bytes}
	var response struct {
		Data string `json:"data"`
		Left int    `json:"left"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal script code response: %w", err)
	}

	// If there's more data (left > 0), we need to fetch it in chunks
	fullCode := response.Data
	offset := len(response.Data)

	for response.Left > 0 {
		result, err = c.Call(ctx, deviceIP, "Script.GetCode", map[string]interface{}{
			"id":     scriptID,
			"offset": offset,
		})
		if err != nil {
			return fullCode, fmt.Errorf("failed to get remaining script data: %w", err)
		}

		if err := json.Unmarshal(result, &response); err != nil {
			return fullCode, fmt.Errorf("failed to unmarshal chunk: %w", err)
		}

		fullCode += response.Data
		offset += len(response.Data)
	}

	return fullCode, nil
}

// PutScriptCode creates or updates a script
func (c *Client) PutScriptCode(ctx context.Context, deviceIP string, scriptID int, code string, append bool) error {
	params := map[string]interface{}{
		"id":     scriptID,
		"code":   code,
		"append": append,
	}
	_, err := c.Call(ctx, deviceIP, "Script.PutCode", params)
	return err
}

// CreateScript creates a new script
func (c *Client) CreateScript(ctx context.Context, deviceIP, name string) (int, error) {
	result, err := c.Call(ctx, deviceIP, "Script.Create", map[string]interface{}{"name": name})
	if err != nil {
		return 0, err
	}

	var response struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return 0, fmt.Errorf("failed to unmarshal create response: %w", err)
	}

	return response.ID, nil
}

// DeleteScript deletes a script
func (c *Client) DeleteScript(ctx context.Context, deviceIP string, scriptID int) error {
	_, err := c.Call(ctx, deviceIP, "Script.Delete", map[string]interface{}{"id": scriptID})
	return err
}

// SetScriptConfig sets script configuration (name, enable)
func (c *Client) SetScriptConfig(ctx context.Context, deviceIP string, scriptID int, name string, enable bool) error {
	params := map[string]interface{}{
		"id": scriptID,
		"config": map[string]interface{}{
			"name":   name,
			"enable": enable,
		},
	}
	_, err := c.Call(ctx, deviceIP, "Script.SetConfig", params)
	return err
}

// StartScript starts a script
func (c *Client) StartScript(ctx context.Context, deviceIP string, scriptID int) error {
	_, err := c.Call(ctx, deviceIP, "Script.Start", map[string]interface{}{"id": scriptID})
	return err
}

// StopScript stops a script
func (c *Client) StopScript(ctx context.Context, deviceIP string, scriptID int) error {
	_, err := c.Call(ctx, deviceIP, "Script.Stop", map[string]interface{}{"id": scriptID})
	return err
}

// GetStatus retrieves device status
func (c *Client) GetStatus(ctx context.Context, deviceIP string) (json.RawMessage, error) {
	return c.Call(ctx, deviceIP, "Shelly.GetStatus", nil)
}

// Reboot reboots the device
func (c *Client) Reboot(ctx context.Context, deviceIP string) error {
	_, err := c.Call(ctx, deviceIP, "Shelly.Reboot", nil)
	return err
}

// ListSchedules retrieves all schedules from a device
func (c *Client) ListSchedules(ctx context.Context, deviceIP string) ([]Schedule, error) {
	result, err := c.Call(ctx, deviceIP, "Schedule.List", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Jobs []Schedule `json:"jobs"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schedules: %w", err)
	}

	return response.Jobs, nil
}

// CreateSchedule creates a new schedule
func (c *Client) CreateSchedule(ctx context.Context, deviceIP string, schedule Schedule) (int, error) {
	params := map[string]interface{}{
		"enable":   schedule.Enable,
		"timespec": schedule.Timespec,
		"calls":    schedule.Calls,
	}

	result, err := c.Call(ctx, deviceIP, "Schedule.Create", params)
	if err != nil {
		return 0, err
	}

	var response struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return 0, fmt.Errorf("failed to unmarshal create response: %w", err)
	}

	return response.ID, nil
}

// UpdateSchedule updates an existing schedule
func (c *Client) UpdateSchedule(ctx context.Context, deviceIP string, schedule Schedule) error {
	params := map[string]interface{}{
		"id":       schedule.ID,
		"enable":   schedule.Enable,
		"timespec": schedule.Timespec,
		"calls":    schedule.Calls,
	}
	_, err := c.Call(ctx, deviceIP, "Schedule.Update", params)
	return err
}

// DeleteSchedule deletes a schedule
func (c *Client) DeleteSchedule(ctx context.Context, deviceIP string, scheduleID int) error {
	_, err := c.Call(ctx, deviceIP, "Schedule.Delete", map[string]interface{}{"id": scheduleID})
	return err
}

// ListWebhooks retrieves all webhooks from a device
func (c *Client) ListWebhooks(ctx context.Context, deviceIP string) ([]Webhook, error) {
	result, err := c.Call(ctx, deviceIP, "Webhook.List", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Hooks []Webhook `json:"hooks"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhooks: %w", err)
	}

	return response.Hooks, nil
}

// CreateWebhook creates a new webhook
func (c *Client) CreateWebhook(ctx context.Context, deviceIP string, webhook Webhook) (int, error) {
	params := map[string]interface{}{
		"cid":    webhook.CID,
		"enable": webhook.Enable,
		"event":  webhook.Event,
	}

	if webhook.Name != "" {
		params["name"] = webhook.Name
	}
	if len(webhook.URLs) > 0 {
		params["urls"] = webhook.URLs
	}
	if len(webhook.Actions) > 0 {
		params["actions"] = webhook.Actions
	}

	result, err := c.Call(ctx, deviceIP, "Webhook.Create", params)
	if err != nil {
		return 0, err
	}

	var response struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return 0, fmt.Errorf("failed to unmarshal create response: %w", err)
	}

	return response.ID, nil
}

// UpdateWebhook updates an existing webhook
func (c *Client) UpdateWebhook(ctx context.Context, deviceIP string, webhook Webhook) error {
	params := map[string]interface{}{
		"id":     webhook.ID,
		"cid":    webhook.CID,
		"enable": webhook.Enable,
		"event":  webhook.Event,
	}

	if webhook.Name != "" {
		params["name"] = webhook.Name
	}
	if len(webhook.URLs) > 0 {
		params["urls"] = webhook.URLs
	}
	if len(webhook.Actions) > 0 {
		params["actions"] = webhook.Actions
	}

	_, err := c.Call(ctx, deviceIP, "Webhook.Update", params)
	return err
}

// DeleteWebhook deletes a webhook
func (c *Client) DeleteWebhook(ctx context.Context, deviceIP string, webhookID int) error {
	_, err := c.Call(ctx, deviceIP, "Webhook.Delete", map[string]interface{}{"id": webhookID})
	return err
}

// SetKVS sets a key-value pair in the device's KVS
func (c *Client) SetKVS(ctx context.Context, deviceIP, key string, value interface{}) error {
	params := map[string]interface{}{
		"key":   key,
		"value": value,
	}
	_, err := c.Call(ctx, deviceIP, "KVS.Set", params)
	return err
}

// DeleteKVS deletes a key from the device's KVS
func (c *Client) DeleteKVS(ctx context.Context, deviceIP, key string) error {
	params := map[string]interface{}{
		"key": key,
	}
	_, err := c.Call(ctx, deviceIP, "KVS.Delete", params)
	return err
}

// GetKVS retrieves all key-value store data from a device
func (c *Client) GetKVS(ctx context.Context, deviceIP string) (map[string]interface{}, error) {
	// First, list all KVS keys
	result, err := c.Call(ctx, deviceIP, "KVS.List", nil)
	if err != nil {
		return nil, err
	}

	// KVS.List returns: {"keys": {"key1": {"etag": "..."}, "key2": {...}}, "rev": 35}
	var listResponse struct {
		Keys map[string]interface{} `json:"keys"`
		Rev  int                    `json:"rev"`
	}
	if err := json.Unmarshal(result, &listResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KVS keys: %w", err)
	}

	// If no keys, return empty map
	if len(listResponse.Keys) == 0 {
		return make(map[string]interface{}), nil
	}

	// Extract key names from the map
	keyNames := make([]string, 0, len(listResponse.Keys))
	for key := range listResponse.Keys {
		keyNames = append(keyNames, key)
	}

	// Get all values using KVS.GetMany
	result, err = c.Call(ctx, deviceIP, "KVS.GetMany", map[string]interface{}{
		"keys": keyNames,
	})
	if err != nil {
		return nil, err
	}

	// KVS.GetMany returns: {"items": [{"key": "k1", "etag": "...", "value": ...}], "offset": 0, "total": N}
	var getManyResponse struct {
		Items []struct {
			Key   string      `json:"key"`
			Etag  string      `json:"etag"`
			Value interface{} `json:"value"`
		} `json:"items"`
		Offset int `json:"offset"`
		Total  int `json:"total"`
	}
	if err := json.Unmarshal(result, &getManyResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KVS values: %w", err)
	}

	// Convert array to map
	kvsData := make(map[string]interface{})
	for _, item := range getManyResponse.Items {
		kvsData[item.Key] = item.Value
	}

	return kvsData, nil
}

// GetComponents retrieves all components including virtual components and groups
// Handles pagination automatically to fetch all components
func (c *Client) GetComponents(ctx context.Context, deviceIP string) ([]ComponentInfo, error) {
	var allComponents []ComponentInfo
	offset := 0

	for {
		// Call with current offset
		params := map[string]interface{}{
			"offset": offset,
		}
		result, err := c.Call(ctx, deviceIP, "Shelly.GetComponents", params)
		if err != nil {
			return nil, err
		}

		var response struct {
			Components []ComponentInfo `json:"components"`
			Offset     int             `json:"offset"`
			Total      int             `json:"total"`
		}
		if err := json.Unmarshal(result, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal components: %w", err)
		}

		// Add components from this page
		allComponents = append(allComponents, response.Components...)

		// Check if we have all components
		if len(allComponents) >= response.Total {
			break
		}

		// Move to next page
		offset += len(response.Components)

		// Safety check to prevent infinite loops
		if len(response.Components) == 0 {
			break
		}
	}

	return allComponents, nil
}
