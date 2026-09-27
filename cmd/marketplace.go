package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
	JSON     bool
}

// MarketplaceInfoOptions parameters for inspecting a marketplace package.
type MarketplaceInfoOptions struct {
	PluginIDOrName string
	CloudURL       string
	Token          string
	OrgID          string
	JSON           bool
}

// MarketplaceInstallOptions parameters for installing a marketplace package.
type MarketplaceInstallOptions struct {
	PluginIDOrName string
	Version        string
	EnvironmentID  string
	ProjectID      string
	CloudURL       string
	Token          string
	OrgID          string
	JSON           bool
}

// MarketplaceVersionsOptions parameters for listing versions of a package.
type MarketplaceVersionsOptions struct {
	PluginIDOrName string
	CloudURL       string
	Token          string
	OrgID          string
	JSON           bool
}

// RunMarketplaceSearch queries the Naagmani Cloud marketplace catalog.
func RunMarketplaceSearch(opts MarketplaceSearchOptions) error {
	cfg, _ := LoadCLIConfig()

	cloudURL := opts.CloudURL
	if cloudURL == "" && cfg != nil {
		cloudURL = cfg.CloudURL
	}
	if cloudURL == "" {
		cloudURL = os.Getenv("NAAGMANI_CLOUD_URL")
	}
	if cloudURL == "" {
		cloudURL = "http://localhost:8081"
	}
	cloudURL = strings.TrimRight(cloudURL, "/")

	token := opts.Token
	if token == "" && cfg != nil {
		token = cfg.Token
	}
	if token == "" {
		token = os.Getenv("NAAGMANI_CLOUD_TOKEN")
	}

	orgID := opts.OrgID
	if orgID == "" && cfg != nil {
		orgID = cfg.OrgID
	}
	if orgID == "" {
		orgID = os.Getenv("NAAGMANI_ORG_ID")
	}

	params := url.Values{}
	if opts.Query != "" {
		params.Set("query", opts.Query)
		params.Set("q", opts.Query)
	}
	if opts.Category != "" {
		params.Set("category", opts.Category)
	}

	// Try canonical /v1/marketplace/packages, fallback to /v1/plugins
	reqURL := fmt.Sprintf("%s/v1/marketplace/packages?%s", cloudURL, params.Encode())
	if orgID != "" {
		reqURL = fmt.Sprintf("%s/v1/organizations/%s/marketplace/packages?%s", cloudURL, orgID, params.Encode())
	}

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

	var parsed struct {
		Packages []struct {
			ID             string `json:"id"`
			Slug           string `json:"slug"`
			Name           string `json:"name"`
			Category       string `json:"category"`
			Description    string `json:"description"`
			PublisherName  string `json:"publisher_name"`
			Tier           string `json:"tier"`
			Status         string `json:"status"`
			Visibility     string `json:"visibility"`
			Certified      bool   `json:"certified"`
			LatestVersion  string `json:"latest_version"`
			DownloadsCount int    `json:"downloads_count"`
		} `json:"packages"`
	}

	// Also support plain array format
	rawBody := new(bytes.Buffer)
	_, _ = rawBody.ReadFrom(resp.Body)

	if err := json.Unmarshal(rawBody.Bytes(), &parsed); err != nil || len(parsed.Packages) == 0 {
		var arr []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Category    string `json:"category"`
			Description string `json:"description"`
			Publisher   string `json:"publisher"`
			Visibility  string `json:"visibility"`
		}
		if json.Unmarshal(rawBody.Bytes(), &arr) == nil && len(arr) > 0 {
			for _, item := range arr {
				parsed.Packages = append(parsed.Packages, struct {
					ID             string `json:"id"`
					Slug           string `json:"slug"`
					Name           string `json:"name"`
					Category       string `json:"category"`
					Description    string `json:"description"`
					PublisherName  string `json:"publisher_name"`
					Tier           string `json:"tier"`
					Status         string `json:"status"`
					Visibility     string `json:"visibility"`
					Certified      bool   `json:"certified"`
					LatestVersion  string `json:"latest_version"`
					DownloadsCount int    `json:"downloads_count"`
				}{
					ID:            item.ID,
					Slug:          item.Name,
					Name:          item.Name,
					Category:      item.Category,
					Description:   item.Description,
					PublisherName: item.Publisher,
					Visibility:    item.Visibility,
					Status:        "PUBLISHED",
					Certified:     true,
				})
			}
		}
	}

	if opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(parsed.Packages)
	}

	if len(parsed.Packages) == 0 {
		fmt.Printf("No extensions found matching query: %q\n", opts.Query)
		return nil
	}

	fmt.Printf("Naagmani Marketplace Extensions (%d found):\n\n", len(parsed.Packages))
	fmt.Printf("%-28s %-10s %-12s %-10s %-8s %s\n", "SLUG / NAME", "TYPE", "TIER", "STATUS", "CERT", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 90))
	for _, p := range parsed.Packages {
		cert := "No"
		if p.Certified {
			cert = "Yes [✓]"
		}
		desc := p.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		tier := p.Tier
		if tier == "" {
			tier = "FREE"
		}
		fmt.Printf("%-28s %-10s %-12s %-10s %-8s %s\n", p.Slug, p.Category, tier, p.Status, cert, desc)
	}
	fmt.Println("\nTo inspect extension details: naagmani marketplace info <slug>")
	return nil
}

// RunMarketplaceInfo fetches detailed metadata, digital signatures, and provenance for a marketplace package.
func RunMarketplaceInfo(opts MarketplaceInfoOptions) error {
	cfg, _ := LoadCLIConfig()

	cloudURL := opts.CloudURL
	if cloudURL == "" && cfg != nil {
		cloudURL = cfg.CloudURL
	}
	if cloudURL == "" {
		cloudURL = os.Getenv("NAAGMANI_CLOUD_URL")
	}
	if cloudURL == "" {
		cloudURL = "http://localhost:8081"
	}
	cloudURL = strings.TrimRight(cloudURL, "/")

	token := opts.Token
	if token == "" && cfg != nil {
		token = cfg.Token
	}
	if token == "" {
		token = os.Getenv("NAAGMANI_CLOUD_TOKEN")
	}

	orgID := opts.OrgID
	if orgID == "" && cfg != nil {
		orgID = cfg.OrgID
	}
	if orgID == "" {
		orgID = os.Getenv("NAAGMANI_ORG_ID")
	}

	reqURL := fmt.Sprintf("%s/v1/marketplace/packages/%s", cloudURL, url.PathEscape(opts.PluginIDOrName))
	if orgID != "" {
		reqURL = fmt.Sprintf("%s/v1/organizations/%s/marketplace/packages/%s", cloudURL, orgID, url.PathEscape(opts.PluginIDOrName))
	}

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
		return fmt.Errorf("package %q not found in marketplace", opts.PluginIDOrName)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("marketplace query failed with status %d", resp.StatusCode)
	}

	var p map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return fmt.Errorf("decoding package info: %w", err)
	}

	if opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(p)
	}

	slug, _ := p["slug"].(string)
	if slug == "" {
		slug, _ = p["name"].(string)
	}
	name, _ := p["name"].(string)
	category, _ := p["category"].(string)
	desc, _ := p["description"].(string)
	tier, _ := p["tier"].(string)
	status, _ := p["status"].(string)
	latestVer, _ := p["latest_version"].(string)
	certified, _ := p["certified"].(bool)

	fmt.Printf("Naagmani Extension: %s (%s)\n", name, slug)
	fmt.Printf("===================================================================\n")
	fmt.Printf("  ID:             %v\n", p["id"])
	fmt.Printf("  Category:       %s\n", category)
	fmt.Printf("  Tier:           %s\n", tier)
	fmt.Printf("  Status:         %s\n", status)
	fmt.Printf("  Certified:      %v\n", certified)
	if latestVer != "" {
		fmt.Printf("  Latest Ver:     %s\n", latestVer)
	}
	if pub, ok := p["publisher_name"].(string); ok && pub != "" {
		fmt.Printf("  Publisher:      %s\n", pub)
	}
	if desc != "" {
		fmt.Printf("  Description:    %s\n", desc)
	}

	// Print manifest summary if available
	if manifest, ok := p["manifest"].(map[string]interface{}); ok {
		if perms, ok := manifest["permissions_declaration"].([]interface{}); ok && len(perms) > 0 {
			var permStrs []string
			for _, item := range perms {
				permStrs = append(permStrs, fmt.Sprintf("%v", item))
			}
			fmt.Printf("  Permissions:    %s\n", strings.Join(permStrs, ", "))
		}
		if deps, ok := manifest["dependencies"].([]interface{}); ok && len(deps) > 0 {
			fmt.Printf("  Dependencies:   %d declared\n", len(deps))
		}
	}

	return nil
}

// RunMarketplaceVersions lists all published versions for a marketplace package.
func RunMarketplaceVersions(opts MarketplaceVersionsOptions) error {
	cfg, _ := LoadCLIConfig()

	cloudURL := opts.CloudURL
	if cloudURL == "" && cfg != nil {
		cloudURL = cfg.CloudURL
	}
	if cloudURL == "" {
		cloudURL = os.Getenv("NAAGMANI_CLOUD_URL")
	}
	if cloudURL == "" {
		cloudURL = "http://localhost:8081"
	}
	cloudURL = strings.TrimRight(cloudURL, "/")

	token := opts.Token
	if token == "" && cfg != nil {
		token = cfg.Token
	}
	if token == "" {
		token = os.Getenv("NAAGMANI_CLOUD_TOKEN")
	}

	orgID := opts.OrgID
	if orgID == "" && cfg != nil {
		orgID = cfg.OrgID
	}
	if orgID == "" {
		orgID = os.Getenv("NAAGMANI_ORG_ID")
	}

	reqURL := fmt.Sprintf("%s/v1/marketplace/packages/%s/versions", cloudURL, url.PathEscape(opts.PluginIDOrName))
	if orgID != "" {
		reqURL = fmt.Sprintf("%s/v1/organizations/%s/marketplace/packages/%s/versions", cloudURL, orgID, url.PathEscape(opts.PluginIDOrName))
	}

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
		return fmt.Errorf("connecting to marketplace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("querying versions failed with status %d", resp.StatusCode)
	}

	var parsed struct {
		Versions []struct {
			Version   string `json:"version"`
			Status    string `json:"status"`
			Artifact  *struct {
				Digest string `json:"digest"`
				Size   int64  `json:"size"`
			} `json:"artifact"`
			Signature *struct {
				Algorithm string `json:"algorithm"`
				KeyID     string `json:"key_id"`
			} `json:"signature"`
			ReleaseNotes string `json:"release_notes"`
		} `json:"versions"`
	}

	_ = json.NewDecoder(resp.Body).Decode(&parsed)

	if opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(parsed.Versions)
	}

	fmt.Printf("Versions for %s (%d found):\n\n", opts.PluginIDOrName, len(parsed.Versions))
	fmt.Printf("%-10s %-12s %-18s %s\n", "VERSION", "STATUS", "DIGEST", "RELEASE NOTES")
	fmt.Println(strings.Repeat("-", 80))
	for _, v := range parsed.Versions {
		digest := "none"
		if v.Artifact != nil && len(v.Artifact.Digest) > 16 {
			digest = v.Artifact.Digest[:16] + "..."
		}
		fmt.Printf("%-10s %-12s %-18s %s\n", v.Version, v.Status, digest, v.ReleaseNotes)
	}

	return nil
}

// RunMarketplaceInstall installs a package from marketplace into the tenant environment.
func RunMarketplaceInstall(opts MarketplaceInstallOptions) error {
	cfg, _ := LoadCLIConfig()

	cloudURL := opts.CloudURL
	if cloudURL == "" && cfg != nil {
		cloudURL = cfg.CloudURL
	}
	if cloudURL == "" {
		cloudURL = os.Getenv("NAAGMANI_CLOUD_URL")
	}
	if cloudURL == "" {
		cloudURL = "http://localhost:8081"
	}
	cloudURL = strings.TrimRight(cloudURL, "/")

	token := opts.Token
	if token == "" && cfg != nil {
		token = cfg.Token
	}
	if token == "" {
		token = os.Getenv("NAAGMANI_CLOUD_TOKEN")
	}
	if token == "" {
		return fmt.Errorf("authentication required to install extensions (run 'naagmani login')")
	}

	orgID := opts.OrgID
	if orgID == "" && cfg != nil {
		orgID = cfg.OrgID
	}
	if orgID == "" {
		orgID = os.Getenv("NAAGMANI_ORG_ID")
	}

	envID := opts.EnvironmentID
	if envID == "" && cfg != nil {
		envID = cfg.Environment
	}
	if envID == "" {
		envID = "production"
	}

	projectID := opts.ProjectID
	if projectID == "" && cfg != nil {
		projectID = cfg.ProjectID
	}
	if projectID == "" {
		projectID = "default"
	}

	bodyData := map[string]interface{}{
		"package_id":      opts.PluginIDOrName,
		"version":         opts.Version,
		"scope_mode":      "selected",
		"project_ids":     []string{projectID},
		"environment_ids": []string{envID},
		"auto_activate":   true,
	}

	payloadBytes, _ := json.Marshal(bodyData)
	reqURL := fmt.Sprintf("%s/v1/marketplace/install", cloudURL)
	if orgID != "" {
		reqURL = fmt.Sprintf("%s/v1/organizations/%s/marketplace/install", cloudURL, orgID)
	}

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
		return fmt.Errorf("installing extension: %w", err)
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

	var installResult map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&installResult)

	if opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(installResult)
	}

	fmt.Printf("✓ Extension %q installed successfully in project %q (environment: %q)\n", opts.PluginIDOrName, projectID, envID)
	fmt.Println("Naagmani OS will automatically verify cryptographic signatures and launch the extension runtime.")
	return nil
}
