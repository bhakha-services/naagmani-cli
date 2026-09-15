package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MarketplaceSearchOptions parameters for searching the plugin marketplace.
type MarketplaceSearchOptions struct {
	Query    string
	Category string
	CloudURL string
	Token    string
	OrgID    string
}

// MarketplaceInfoOptions parameters for inspecting a marketplace plugin.
type MarketplaceInfoOptions struct {
	PluginIDOrName string
	CloudURL       string
	Token          string
	OrgID          string
}

// MarketplaceInstallOptions parameters for installing a marketplace plugin.
type MarketplaceInstallOptions struct {
	PluginIDOrName string
	Version        string
	EnvironmentID  string
	ProjectID      string
	CloudURL       string
	Token          string
	OrgID          string
}

// RunMarketplaceSearch queries the Naagmani Cloud plugin catalog.
func RunMarketplaceSearch(opts MarketplaceSearchOptions) error {
	cfg, err := LoadCLIConfig()
	if err != nil {
		return err
	}

	cloudURL := opts.CloudURL
	if cloudURL == "" {
		cloudURL = cfg.CloudURL
	}
	cloudURL = strings.TrimRight(cloudURL, "/")

	token := opts.Token
	if token == "" {
		token = cfg.Token
	}

	orgID := opts.OrgID
	if orgID == "" {
		orgID = cfg.OrgID
	}

	params := url.Values{}
	if opts.Query != "" {
		params.Set("q", opts.Query)
	}
	if opts.Category != "" {
		params.Set("category", opts.Category)
	}

	reqURL := fmt.Sprintf("%s/v1/plugins?%s", cloudURL, params.Encode())
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return err
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
		return fmt.Errorf("connecting to marketplace at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("marketplace query failed with status %d", resp.StatusCode)
	}

	var results []struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Category    string   `json:"category"`
		Description string   `json:"description"`
		Author      string   `json:"author"`
		Publisher   string   `json:"publisher"`
		Visibility  string   `json:"visibility"`
		Versions    []string `json:"versions,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return fmt.Errorf("decoding marketplace results: %w", err)
	}

	if len(results) == 0 {
		fmt.Printf("No plugins found matching query: %q\n", opts.Query)
		return nil
	}

	fmt.Printf("Marketplace Plugins (%d found):\n\n", len(results))
	fmt.Printf("%-24s %-16s %-14s %s\n", "NAME", "CATEGORY", "VISIBILITY", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))
	for _, p := range results {
		desc := p.Description
		if len(desc) > 35 {
			desc = desc[:32] + "..."
		}
		fmt.Printf("%-24s %-16s %-14s %s\n", p.Name, p.Category, p.Visibility, desc)
	}
	fmt.Println("\nTo inspect details: naagmani marketplace info <name>")
	return nil
}

// RunMarketplaceInfo fetches detailed metadata for a marketplace plugin.
func RunMarketplaceInfo(opts MarketplaceInfoOptions) error {
	cfg, err := LoadCLIConfig()
	if err != nil {
		return err
	}

	cloudURL := opts.CloudURL
	if cloudURL == "" {
		cloudURL = cfg.CloudURL
	}
	cloudURL = strings.TrimRight(cloudURL, "/")

	token := opts.Token
	if token == "" {
		token = cfg.Token
	}

	orgID := opts.OrgID
	if orgID == "" {
		orgID = cfg.OrgID
	}

	reqURL := fmt.Sprintf("%s/v1/plugins/%s", cloudURL, url.PathEscape(opts.PluginIDOrName))
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return err
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
		return fmt.Errorf("connecting to marketplace at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("plugin %q not found in marketplace", opts.PluginIDOrName)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("marketplace query failed with status %d", resp.StatusCode)
	}

	var p struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Category     string   `json:"category"`
		Description  string   `json:"description"`
		Author       string   `json:"author"`
		Publisher    string   `json:"publisher"`
		Capabilities []string `json:"capabilities"`
		Permissions  []string `json:"permissions"`
		LatestVer    string   `json:"latest_version"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return fmt.Errorf("decoding plugin info: %w", err)
	}

	fmt.Printf("Plugin: %s\n", p.Name)
	fmt.Printf("========================================\n")
	fmt.Printf("  ID:           %s\n", p.ID)
	fmt.Printf("  Category:     %s\n", p.Category)
	if p.LatestVer != "" {
		fmt.Printf("  Latest Ver:   %s\n", p.LatestVer)
	}
	if p.Author != "" {
		fmt.Printf("  Author:       %s\n", p.Author)
	}
	if p.Description != "" {
		fmt.Printf("  Description:  %s\n", p.Description)
	}
	if len(p.Capabilities) > 0 {
		fmt.Printf("  Capabilities: %s\n", strings.Join(p.Capabilities, ", "))
	}
	if len(p.Permissions) > 0 {
		fmt.Printf("  Permissions:  %s\n", strings.Join(p.Permissions, ", "))
	}
	return nil
}

// RunMarketplaceInstall installs a plugin from the marketplace into the tenant environment.
func RunMarketplaceInstall(opts MarketplaceInstallOptions) error {
	cfg, err := LoadCLIConfig()
	if err != nil {
		return err
	}

	cloudURL := opts.CloudURL
	if cloudURL == "" {
		cloudURL = cfg.CloudURL
	}
	cloudURL = strings.TrimRight(cloudURL, "/")

	token := opts.Token
	if token == "" {
		token = cfg.Token
	}
	if token == "" {
		return fmt.Errorf("authentication required to install plugins (run 'naagmani login')")
	}

	orgID := opts.OrgID
	if orgID == "" {
		orgID = cfg.OrgID
	}

	envID := opts.EnvironmentID
	if envID == "" {
		envID = cfg.Environment
	}
	if envID == "" {
		envID = "production"
	}

	projectID := opts.ProjectID
	if projectID == "" {
		projectID = cfg.ProjectID
	}

	bodyData := map[string]interface{}{
		"plugin_id_or_name": opts.PluginIDOrName,
		"environment_id":    envID,
		"version":           opts.Version,
	}
	if projectID != "" {
		bodyData["project_id"] = projectID
	}

	payloadBytes, _ := json.Marshal(bodyData)
	reqURL := fmt.Sprintf("%s/v1/plugins/%s/install", cloudURL, url.PathEscape(opts.PluginIDOrName))
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if orgID != "" {
		req.Header.Set("X-Organization-ID", orgID)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("installing plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg, _ := errResp["error"].(string)
		if msg == "" {
			msg, _ = errResp["message"].(string)
		}
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("installation failed: %s", msg)
	}

	fmt.Printf("✓ Plugin %q installed successfully in environment %q\n", opts.PluginIDOrName, envID)
	fmt.Println("Naagmani OS will automatically reconcile and launch the plugin process.")
	return nil
}
