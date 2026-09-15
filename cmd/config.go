package cmd

import (
	"fmt"
	"strings"
)

// RunConfig processes get, set, and list operations on local CLI configuration.
func RunConfig(args []string) error {
	if len(args) == 0 || args[0] == "list" {
		return runConfigList()
	}

	subcmd := args[0]
	switch subcmd {
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("missing key name\nUsage: naagmani config get <key>")
		}
		return runConfigGet(args[1])

	case "set":
		if len(args) < 3 {
			return fmt.Errorf("missing key or value\nUsage: naagmani config set <key> <value>")
		}
		return runConfigSet(args[1], args[2])

	default:
		return fmt.Errorf("unknown config action: %q (supported: get, set, list)", subcmd)
	}
}

func runConfigList() error {
	cfg, err := LoadCLIConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Println("Naagmani CLI Configuration:")
	fmt.Printf("  cloud_url:     %s\n", cfg.CloudURL)
	fmt.Printf("  os_endpoint:   %s\n", cfg.OSEndpoint)
	if cfg.Email != "" {
		fmt.Printf("  email:         %s\n", cfg.Email)
	}
	if cfg.OrgID != "" {
		fmt.Printf("  org_id:        %s\n", cfg.OrgID)
	}
	if cfg.ProjectID != "" {
		fmt.Printf("  project_id:    %s\n", cfg.ProjectID)
	}
	if cfg.Environment != "" {
		fmt.Printf("  environment:   %s\n", cfg.Environment)
	}
	if cfg.Token != "" {
		fmt.Printf("  token:         %s\n", maskSecret(cfg.Token))
	}
	if cfg.APIKey != "" {
		fmt.Printf("  api_key:       %s\n", maskSecret(cfg.APIKey))
	}
	return nil
}

func runConfigGet(key string) error {
	cfg, err := LoadCLIConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	switch strings.ToLower(key) {
	case "cloud_url", "cloud-url":
		fmt.Println(cfg.CloudURL)
	case "os_endpoint", "os-endpoint", "os_url", "os-url":
		fmt.Println(cfg.OSEndpoint)
	case "email":
		fmt.Println(cfg.Email)
	case "org_id", "org", "org-id":
		fmt.Println(cfg.OrgID)
	case "project_id", "project", "project-id":
		fmt.Println(cfg.ProjectID)
	case "environment", "env":
		fmt.Println(cfg.Environment)
	case "token":
		fmt.Println(maskSecret(cfg.Token))
	case "api_key", "api-key":
		fmt.Println(maskSecret(cfg.APIKey))
	default:
		return fmt.Errorf("unknown configuration key: %q", key)
	}
	return nil
}

func runConfigSet(key, value string) error {
	cfg, err := LoadCLIConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	switch strings.ToLower(key) {
	case "cloud_url", "cloud-url":
		cfg.CloudURL = value
	case "os_endpoint", "os-endpoint", "os_url", "os-url":
		cfg.OSEndpoint = value
	case "org_id", "org", "org-id":
		cfg.OrgID = value
	case "project_id", "project", "project-id":
		cfg.ProjectID = value
	case "environment", "env":
		cfg.Environment = value
	case "token":
		cfg.Token = value
	case "api_key", "api-key":
		cfg.APIKey = value
	case "email":
		cfg.Email = value
	default:
		return fmt.Errorf("unknown configuration key: %q", key)
	}

	if err := SaveCLIConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("✓ Set %s = %s\n", key, value)
	return nil
}
