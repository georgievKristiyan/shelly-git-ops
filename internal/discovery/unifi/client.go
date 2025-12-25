package unifi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// Client handles UniFi Controller API communication
type Client struct {
	baseURL    string
	httpClient *http.Client
	site       string
	apiVersion string // "legacy" or "network-app"
}

// LoginRequest represents the login credentials
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// DeviceResponse represents the UniFi API device list response
type DeviceResponse struct {
	Meta struct {
		RC string `json:"rc"`
	} `json:"meta"`
	Data []UniFiDevice `json:"data"`
}

// UniFiDevice represents a device from UniFi API
type UniFiDevice struct {
	MAC        string `json:"mac"`
	IP         string `json:"ip"`
	Hostname   string `json:"hostname"`
	Name       string `json:"name"`
	LastSeen   int64  `json:"last_seen"`
	IsWired    bool   `json:"is_wired"`
	UseFixedIP bool   `json:"use_fixedip"`
	FixedIP    string `json:"fixed_ip"`
	NetworkID  string `json:"network_id"`
}

// NewClient creates a new UniFi API client
func NewClient(baseURL, username, password string, verifySSL bool) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	httpClient := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !verifySSL,
			},
		},
	}

	client := &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		site:       "default", // Default site name
		apiVersion: "unknown",
	}

	// Authenticate and detect API version
	if err := client.login(context.Background(), username, password); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return client, nil
}

// login authenticates with the UniFi controller
// Tries multiple API endpoints to detect controller version
func (c *Client) login(ctx context.Context, username, password string) error {
	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	body, err := json.Marshal(loginReq)
	if err != nil {
		return err
	}

	// Try different login endpoints (newer first since user has newer version)
	endpoints := []struct {
		path    string
		version string
	}{
		{"/api/auth/login", "network-app"},  // Newer UniFi Network Application
		{"/api/login", "legacy"},            // Older UniFi Controller
		{"/api/auth", "network-app-alt"},    // Alternative newer endpoint
	}

	var lastErr error
	for _, endpoint := range endpoints {
		url := fmt.Sprintf("%s%s", c.baseURL, endpoint.path)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			c.apiVersion = endpoint.version
			return nil
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		lastErr = fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return fmt.Errorf("all login attempts failed: %w", lastErr)
}

// GetClients retrieves all clients from the UniFi controller
func (c *Client) GetClients(ctx context.Context) ([]UniFiDevice, error) {
	// Try different API paths based on detected version
	paths := []string{}

	if c.apiVersion == "network-app" || c.apiVersion == "network-app-alt" {
		// Newer UniFi Network Application paths
		paths = []string{
			fmt.Sprintf("/proxy/network/api/s/%s/stat/sta", c.site),
			fmt.Sprintf("/proxy/network/v2/api/site/%s/clients/active", c.site),
			fmt.Sprintf("/v2/api/site/%s/clients/active", c.site),
			fmt.Sprintf("/api/s/%s/stat/sta", c.site),
			fmt.Sprintf("/api/s/%s/rest/user", c.site),
		}
	} else {
		// Legacy UniFi Controller paths
		paths = []string{
			fmt.Sprintf("/api/s/%s/stat/sta", c.site),
			fmt.Sprintf("/api/s/%s/rest/user", c.site),
			fmt.Sprintf("/proxy/network/api/s/%s/stat/sta", c.site),
		}
	}

	var lastErr error
	for _, path := range paths {
		url := fmt.Sprintf("%s%s", c.baseURL, path)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		// Read response body first
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("GET %s failed with status %d: %s", path, resp.StatusCode, string(bodyBytes))
			continue
		}

		var deviceResp DeviceResponse
		if err := json.Unmarshal(bodyBytes, &deviceResp); err != nil {
			lastErr = fmt.Errorf("failed to parse response from %s: %w. Body: %s", path, err, string(bodyBytes))
			continue
		}

		// Some endpoints might not have meta.rc field, check if we got data
		if deviceResp.Meta.RC != "" && deviceResp.Meta.RC != "ok" {
			lastErr = fmt.Errorf("API returned error: %s", deviceResp.Meta.RC)
			continue
		}

		// Successfully got devices
		if len(deviceResp.Data) >= 0 {
			// Success! Could log which path worked here for debugging
			// fmt.Printf("DEBUG: Successfully fetched clients from: %s\n", path)
			return deviceResp.Data, nil
		}

		lastErr = fmt.Errorf("no data in response from %s", path)
	}

	return nil, fmt.Errorf("all GetClients attempts failed. Last error: %w. Tried paths: %v", lastErr, paths)
}

// SetStaticIP sets a static IP for a device via DHCP reservation
func (c *Client) SetStaticIP(ctx context.Context, mac, ip, hostname string) error {
	url := fmt.Sprintf("%s/api/s/%s/rest/user", c.baseURL, c.site)

	payload := map[string]interface{}{
		"mac":         mac,
		"use_fixedip": true,
		"fixed_ip":    ip,
		"name":        hostname,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set static IP failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Close closes the client connection
func (c *Client) Close() error {
	// Logout
	url := fmt.Sprintf("%s/api/logout", c.baseURL)
	req, _ := http.NewRequest("POST", url, nil)
	c.httpClient.Do(req)
	return nil
}
