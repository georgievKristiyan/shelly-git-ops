package shelly

import "encoding/json"

// DeviceInfo represents Shelly device information
type DeviceInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Model      string `json:"model"`
	Gen        int    `json:"gen"`
	FW         string `json:"fw_id"`
	App        string `json:"app"`
	Auth       bool   `json:"auth_en"`
	AuthDomain string `json:"auth_domain"`
}

// Script represents a Shelly script
type Script struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Enable  bool   `json:"enable"`
	Running bool   `json:"running"`
}

// ScriptCode represents script code
type ScriptCode struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// Component represents a virtual component
type Component struct {
	Key    string          `json:"key"`
	Status json.RawMessage `json:"status"`
	Config json.RawMessage `json:"config"`
}

// ComponentInfo represents component information from Shelly.GetComponents
type ComponentInfo struct {
	Key    string          `json:"key"`
	Status json.RawMessage `json:"status,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

// BooleanComponent represents a boolean virtual component
type BooleanComponent struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Value  bool   `json:"value"`
	Meta   ComponentMeta `json:"meta"`
}

// NumberComponent represents a number virtual component
type NumberComponent struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Meta   ComponentMeta `json:"meta"`
}

// TextComponent represents a text virtual component
type TextComponent struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Value  string `json:"value"`
	Meta   ComponentMeta `json:"meta"`
}

// ComponentMeta represents component metadata
type ComponentMeta struct {
	UI struct {
		View string `json:"view"`
	} `json:"ui"`
}

// Group represents a Shelly group
type Group struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// KVSData represents key-value store data
type KVSData map[string]interface{}

// Schedule represents a Shelly schedule
type Schedule struct {
	ID      int             `json:"id"`
	Enable  bool            `json:"enable"`
	Timespec string         `json:"timespec"`
	Calls   []ScheduleCall  `json:"calls"`
}

// ScheduleCall represents a call within a schedule
type ScheduleCall struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// Webhook represents a Shelly webhook/action
type Webhook struct {
	ID      int      `json:"id"`
	CID     int      `json:"cid"`
	Enable  bool     `json:"enable"`
	Event   string   `json:"event"`
	Name    string   `json:"name,omitempty"`
	URLs    []string `json:"urls,omitempty"`
	Actions []Action `json:"actions,omitempty"`
}

// Action represents an action within a webhook
type Action struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// DeviceConfig represents complete device configuration
type DeviceConfig struct {
	DeviceInfo DeviceInfo      `json:"device_info"`
	Config     json.RawMessage `json:"config"`
	Scripts    []ScriptCode    `json:"scripts"`
	Components []Component     `json:"components"`
	KVS        KVSData         `json:"kvs"`
}
