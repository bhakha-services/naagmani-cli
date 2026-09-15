package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// VersionsOptions parameters for listing versions of a plugin.
type VersionsOptions struct {
	PluginName string
	CloudURL   string
	Token      string
	OrgID      string
}

type PluginVersionListItem struct {
	ID              string `json:"id"`
	PluginID        string `json:"plugin_id"`
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocol_version"`
	Status          string `json:"status"`
	PublishedAt     string `json:"published_at"`
}

// RunPluginVersions lists published versions for a plugin.
func RunPluginVersions(opts VersionsOptions) error {
	if opts.PluginName == "" {
		return fmt.Errorf("plugin name is required")
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

	url := fmt.Sprintf("%s/v1/plugins/%s/versions", cloudURL, opts.PluginName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if orgID != "" {
		req.Header.Set("X-Organization-ID", orgID)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("requesting versions from Cloud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Cloud returned error (status %d)", resp.StatusCode)
	}

	var versions []PluginVersionListItem
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	fmt.Printf("%s\n\n", opts.PluginName)
	if len(versions) == 0 {
		fmt.Printf("No published versions found.\n")
		return nil
	}

	for _, v := range versions {
		fmt.Printf("%-8s %s\n", v.Version, v.Status)
	}

	return nil
}
