package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

// InitOptions parameters for initializing a new Naagmani project.
type InitOptions struct {
	ProjectName string
	Environment string
	Force       bool
}

// RunInit initializes a canonical naagmani.yaml configuration file in the working directory.
func RunInit(opts InitOptions) error {
	configPath := "naagmani.yaml"
	if _, err := os.Stat(configPath); err == nil && !opts.Force {
		return fmt.Errorf("naagmani.yaml already exists in current directory (use --force to overwrite)")
	}

	projectName := opts.ProjectName
	if projectName == "" {
		cwd, err := os.Getwd()
		if err == nil {
			projectName = filepath.Base(cwd)
		} else {
			projectName = "my-naagmani-app"
		}
	}

	env := opts.Environment
	if env == "" {
		env = "development"
	}

	content := fmt.Sprintf(`# Naagmani Project Configuration
version: "1.0"
name: "%s"
environment: "%s"

# Local Development Settings
os:
  endpoint: "http://localhost:8080"
  log_level: "info"

# Plugin Runtime Configuration
plugins:
  directory: "./plugins"
  auto_reload: true
  default_failure_policy: "fail_close"

# AI Routing Defaults
models:
  default: "gpt-4o"
  fallback: "claude-3-5-sonnet"
`, projectName, env)

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing naagmani.yaml: %w", err)
	}

	fmt.Printf("✓ Initialized Naagmani project configuration in %s\n", configPath)
	fmt.Printf("  Project:     %s\n", projectName)
	fmt.Printf("  Environment: %s\n", env)
	return nil
}
