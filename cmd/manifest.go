package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Manifest mirrors the Naagmani OS Manifest specification.
type Manifest struct {
	Name          string           `json:"name"`
	Version       string           `json:"version"`
	APIVersion    string           `json:"api_version"`
	Description   string           `json:"description,omitempty"`
	Author        string           `json:"author,omitempty"`
	Runtime       RuntimeConfig    `json:"runtime"`
	Permissions   []string         `json:"permissions"`
	Hooks         []string         `json:"hooks"`
	Capabilities  []string         `json:"capabilities,omitempty"`
	Priority      int              `json:"priority,omitempty"`
	FailurePolicy string           `json:"failure_policy,omitempty"`
	TimeoutMs     int              `json:"timeout_ms,omitempty"`
	ConfigSchema  json.RawMessage  `json:"config_schema,omitempty"`
}

// RuntimeConfig defines the execution properties for the plugin.
type RuntimeConfig struct {
	Language string            `json:"language"`
	Command  string            `json:"command"`
	Args     []string          `json:"args,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	WorkDir  string            `json:"work_dir,omitempty"`
}

// LoadManifest reads and parses a plugin.json file.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest file: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest JSON: %w", err)
	}

	if err := m.Validate(); err != nil {
		return nil, err
	}

	return &m, nil
}

// Validate applies the strict validation rules of Naagmani OS.
func (m *Manifest) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("version is required")
	}
	if strings.TrimSpace(m.APIVersion) == "" {
		return fmt.Errorf("api_version is required")
	}
	if m.APIVersion != "v1" {
		return fmt.Errorf("unsupported api_version: %q (supported: v1)", m.APIVersion)
	}
	if strings.TrimSpace(m.Runtime.Command) == "" {
		return fmt.Errorf("runtime.command is required")
	}
	if strings.Contains(m.Runtime.Command, "../") || strings.Contains(m.Runtime.Command, "..\\") {
		return fmt.Errorf("runtime.command contains invalid path traversal: %s", m.Runtime.Command)
	}
	if strings.TrimSpace(m.Runtime.Language) == "" {
		return fmt.Errorf("runtime.language is required")
	}
	if len(m.Permissions) == 0 {
		return fmt.Errorf("at least one permission must be declared")
	}

	validHooks := map[string]bool{
		"request.before":  true,
		"request.after":   true,
		"response.before": true,
		"response.after":  true,
	}
	for _, hook := range m.Hooks {
		if !validHooks[hook] {
			return fmt.Errorf("unknown hook: %q (valid: request.before, request.after, response.before, response.after)", hook)
		}
	}

	if m.Priority <= 0 {
		m.Priority = 100
	}
	if m.TimeoutMs <= 0 {
		m.TimeoutMs = 500
	}
	if m.FailurePolicy == "" {
		m.FailurePolicy = "fail_close"
	} else if m.FailurePolicy != "fail_close" && m.FailurePolicy != "fail_open" {
		return fmt.Errorf("invalid failure_policy: %q (supported: fail_close, fail_open)", m.FailurePolicy)
	}

	return nil
}
