package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CLIConfig holds persistent local CLI configuration and auth token.
type CLIConfig struct {
	CloudURL    string `json:"cloud_url"`
	Token       string `json:"token,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
	Email       string `json:"email,omitempty"`
	OrgID       string `json:"org_id,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	Environment string `json:"environment,omitempty"`
	OSEndpoint  string `json:"os_endpoint,omitempty"`
}

// LoginOptions parameters for developer authentication.
type LoginOptions struct {
	CloudURL string
	Email    string
	Password string
}

func getCLIConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".naagmani")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadCLIConfig loads persistent CLI configuration if present, overlaid with environment variable overrides.
func LoadCLIConfig() (*CLIConfig, error) {
	cfg := &CLIConfig{
		CloudURL:   "http://localhost:8081",
		OSEndpoint: "http://localhost:8080",
	}

	cfgPath, err := getCLIConfigPath()
	if err == nil {
		if data, err := os.ReadFile(cfgPath); err == nil {
			_ = json.Unmarshal(data, cfg)
		}
	}

	// Environment variable overrides (Highest Precedence)
	if envCloud := os.Getenv("NAAGMANI_CLOUD_URL"); envCloud != "" {
		cfg.CloudURL = envCloud
	}
	if envToken := os.Getenv("NAAGMANI_TOKEN"); envToken != "" {
		cfg.Token = envToken
	}
	if envKey := os.Getenv("NAAGMANI_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
		if cfg.Token == "" {
			cfg.Token = envKey
		}
	}
	if envOrg := os.Getenv("NAAGMANI_ORG_ID"); envOrg != "" {
		cfg.OrgID = envOrg
	}
	if envProj := os.Getenv("NAAGMANI_PROJECT_ID"); envProj != "" {
		cfg.ProjectID = envProj
	}
	if envEnv := os.Getenv("NAAGMANI_ENVIRONMENT"); envEnv != "" {
		cfg.Environment = envEnv
	}
	if envOS := os.Getenv("NAAGMANI_OS_URL"); envOS != "" {
		cfg.OSEndpoint = envOS
	}

	if cfg.CloudURL == "" {
		cfg.CloudURL = "http://localhost:8081"
	}
	if cfg.OSEndpoint == "" {
		cfg.OSEndpoint = "http://localhost:8080"
	}
	return cfg, nil
}

// SaveCLIConfig writes persistent CLI configuration.
func SaveCLIConfig(cfg *CLIConfig) error {
	cfgPath, err := getCLIConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0600)
}

// RunLogin logs a developer into Naagmani Cloud.
func RunLogin(opts LoginOptions) error {
	cloudURL := strings.TrimRight(opts.CloudURL, "/")
	if cloudURL == "" {
		cloudURL = "http://localhost:8081"
	}

	if opts.Email == "" || opts.Password == "" {
		return fmt.Errorf("email and password are required")
	}

	loginPayload, _ := json.Marshal(map[string]string{
		"email":    opts.Email,
		"password": opts.Password,
	})

	loginURL := cloudURL + "/v1/auth/login"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(loginURL, "application/json", bytes.NewReader(loginPayload))
	if err != nil {
		return fmt.Errorf("connecting to Naagmani Cloud at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg, _ := errResp["error"].(string)
		if msg == "" {
			msg, _ = errResp["message"].(string)
		}
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("login failed: %s", msg)
	}

	var authResp struct {
		Token string `json:"token"`
		User  struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("parsing auth response: %w", err)
	}

	cfg := &CLIConfig{
		CloudURL: cloudURL,
		Token:    authResp.Token,
		Email:    authResp.User.Email,
	}

	if err := SaveCLIConfig(cfg); err != nil {
		return fmt.Errorf("saving login credentials: %w", err)
	}

	fmt.Printf("✓ Successfully authenticated as %s (%s)\n", authResp.User.Name, authResp.User.Email)
	fmt.Printf("✓ Cloud control plane: %s\n", cloudURL)
	return nil
}
