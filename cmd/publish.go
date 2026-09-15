package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PublishOptions parameters for publishing a plugin to Naagmani Cloud.
type PublishOptions struct {
	Dir          string
	CloudURL     string
	Token        string
	OrgID        string
	TargetOS     string
	TargetArch   string
	ReleaseNotes string
}

// PublishResult response from Naagmani Cloud publish API.
type PublishResult struct {
	PluginID        string `json:"plugin_id"`
	PluginName      string `json:"plugin_name"`
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocol_version"`
	ArtifactURL     string `json:"artifact_url"`
	Checksum        string `json:"checksum"`
	Status          string `json:"status"`
}

// RunPluginPublish builds, packages, and publishes a plugin to Naagmani Cloud.
func RunPluginPublish(opts PublishOptions) (*PublishResult, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	if opts.TargetOS == "" {
		opts.TargetOS = "linux"
	}
	if opts.TargetArch == "" {
		opts.TargetArch = "amd64"
	}

	absDir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("resolving absolute directory: %w", err)
	}

	manifestPath := filepath.Join(absDir, "plugin.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	// 1. Resolve Cloud URL and Auth Token
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
		return nil, fmt.Errorf("authentication required to publish: run 'naagmani login' or pass --token flag / NAAGMANI_CLOUD_TOKEN environment variable")
	}

	fmt.Printf("Naagmani Plugin Publish\n\n")
	fmt.Printf("Plugin:      %s (v%s)\n", manifest.Name, manifest.Version)
	fmt.Printf("Target OS:   %s/%s\n", opts.TargetOS, opts.TargetArch)
	fmt.Printf("Cloud:       %s\n\n", cloudURL)

	// 2. Package the plugin (automatically builds if needed)
	pkgOpts := PackageOptions{
		Dir:        absDir,
		TargetOS:   opts.TargetOS,
		TargetArch: opts.TargetArch,
		OutputDir:  "dist",
	}
	pkgResult, err := RunPluginPackage(pkgOpts)
	if err != nil {
		return nil, fmt.Errorf("packaging failed: %w", err)
	}

	// Read manifest content for metadata
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading plugin.json: %w", err)
	}

	// 3. Prepare multipart request
	bodyBuf := &bytes.Buffer{}
	mpWriter := multipart.NewWriter(bodyBuf)

	// Form fields
	_ = mpWriter.WriteField("name", manifest.Name)
	_ = mpWriter.WriteField("display_name", manifest.Name)
	if manifest.Description != "" {
		_ = mpWriter.WriteField("description", manifest.Description)
	}
	if manifest.Author != "" {
		_ = mpWriter.WriteField("publisher", manifest.Author)
	}
	_ = mpWriter.WriteField("version", manifest.Version)
	_ = mpWriter.WriteField("protocol_version", "naagmani.plugin/v1")
	_ = mpWriter.WriteField("os", opts.TargetOS)
	_ = mpWriter.WriteField("architecture", opts.TargetArch)
	_ = mpWriter.WriteField("checksum", "sha256:"+pkgResult.SHA256)
	_ = mpWriter.WriteField("manifest", string(manifestRaw))
	if opts.ReleaseNotes != "" {
		_ = mpWriter.WriteField("release_notes", opts.ReleaseNotes)
	}

	// File field
	filePart, err := mpWriter.CreateFormFile("artifact", pkgResult.Artifact)
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}

	f, err := os.Open(pkgResult.ArtifactPath)
	if err != nil {
		return nil, fmt.Errorf("opening artifact file %s: %w", pkgResult.ArtifactPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(filePart, f); err != nil {
		return nil, fmt.Errorf("copying artifact to form body: %w", err)
	}

	if err := mpWriter.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	// 4. Send upload request to Cloud POST /v1/plugins/publish
	publishURL := cloudURL + "/v1/plugins/publish"
	httpReq, err := http.NewRequest(http.MethodPost, publishURL, bodyBuf)
	if err != nil {
		return nil, fmt.Errorf("creating publish request: %w", err)
	}

	httpReq.Header.Set("Content-Type", mpWriter.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+token)
	if orgID != "" {
		httpReq.Header.Set("X-Organization-ID", orgID)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling Naagmani Cloud publish API at %s: %w", publishURL, err)
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
		return nil, fmt.Errorf("publish failed (%d): %s", resp.StatusCode, msg)
	}

	var pubResult PublishResult
	if err := json.NewDecoder(resp.Body).Decode(&pubResult); err != nil {
		return nil, fmt.Errorf("decoding publish response: %w", err)
	}

	fmt.Printf("✓ Plugin published successfully!\n")
	fmt.Printf("✓ Plugin ID:        %s\n", pubResult.PluginID)
	fmt.Printf("✓ Version:          %s\n", pubResult.Version)
	fmt.Printf("✓ Protocol:         %s\n", pubResult.ProtocolVersion)
	fmt.Printf("✓ Checksum:         %s\n", pubResult.Checksum)
	fmt.Printf("✓ Artifact URL:     %s\n", pubResult.ArtifactURL)
	fmt.Printf("✓ Marketplace status: %s\n\n", pubResult.Status)

	return &pubResult, nil
}
