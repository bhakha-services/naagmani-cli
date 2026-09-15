package cmd

import (
	"fmt"
)

// RunLogout clears locally stored authentication credentials.
func RunLogout() error {
	cfg, err := LoadCLIConfig()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	cfg.Token = ""
	cfg.APIKey = ""
	cfg.Email = ""

	if err := SaveCLIConfig(cfg); err != nil {
		return fmt.Errorf("clearing credentials: %w", err)
	}

	fmt.Println("✓ Successfully logged out.")
	fmt.Println("Local session credentials removed from ~/.naagmani/config.json")
	return nil
}
