package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type UpdateOptions struct {
	PluginName          string
	TargetVersion       string
	ApprovedPermissions []string
	CloudURL            string
	Token               string
	OrgID               string
}

type UpdateResponse struct {
	ID             string `json:"id"`
	PluginID       string `json:"plugin_id"`
	PluginName     string `json:"plugin_name,omitempty"`
	DesiredVersion string `json:"desired_version"`
	DesiredState   string `json:"desired_state"`
	ObservedState  string `json:"observed_state"`
}

func RunPluginUpdate(opts UpdateOptions) error {
	if opts.PluginName == "" {
		return fmt.Errorf("plugin name is required")
	}
	if opts.TargetVersion == "" {
		return fmt.Errorf("--version is required")
	}

	cloudURL := opts.CloudURL
	if cloudURL == "" {
		cloudURL = os.Getenv("NAAGMANI_CLOUD_URL")
	}
	token := opts.Token
	if token == "" {
		token = os.Getenv("NAAGMANI_CLOUD_TOKEN")
	}
	orgID := opts.OrgID
	if orgID == "" {
		orgID = os.Getenv("NAAGMANI_ORG_ID")
	}

	if cloudURL == "" || token == "" {
		cfg, _ := LoadCLIConfig()
		if cfg != nil {
			if cloudURL == "" && cfg.CloudURL != "" {
				cloudURL = cfg.CloudURL
			}
			if token == "" && cfg.Token != "" {
				token = cfg.Token
			}
		}
	}

	if cloudURL == "" {
		cloudURL = "http://localhost:8081"
	}
	cloudURL = strings.TrimRight(cloudURL, "/")

	if token == "" {
		return fmt.Errorf("authentication required: run 'naagmani login' or pass --token")
	}

	payload, err := json.Marshal(map[string]interface{}{
		"version":              opts.TargetVersion,
		"approved_permissions": opts.ApprovedPermissions,
	})
	if err != nil {
		return fmt.Errorf("marshaling update payload: %w", err)
	}

	url := fmt.Sprintf("%s/v1/organizations/%s/plugins/%s/update", cloudURL, orgID, opts.PluginName)
	if orgID == "" {
		url = fmt.Sprintf("%s/v1/plugins/%s/update", cloudURL, opts.PluginName)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating update request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if orgID != "" {
		req.Header.Set("X-Organization-ID", orgID)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending update request to Cloud: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res UpdateResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	fmt.Printf("Plugin update initiated successfully.\n\n")
	fmt.Printf("Plugin:          %s\n", opts.PluginName)
	fmt.Printf("Desired Version: %s\n", opts.TargetVersion)
	fmt.Printf("Desired State:   %s\n", res.DesiredState)
	fmt.Printf("Observed State:  %s\n\n", res.ObservedState)
	fmt.Printf("Naagmani OS nodes will reconcile toward version %s.\n", opts.TargetVersion)

	return nil
}
