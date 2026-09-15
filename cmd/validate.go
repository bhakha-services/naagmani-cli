package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RunPluginValidate validates a plugin directory and manifest.
func RunPluginValidate(dir string) error {
	if dir == "" {
		dir = "."
	}

	manifestPath := filepath.Join(dir, "plugin.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest not found at %s", manifestPath)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Manifest invalid: %v\n", err)
		return err
	}

	fmt.Printf("Naagmani Plugin Manifest Validation\n\n")
	fmt.Printf("✓ Manifest file: %s\n", manifestPath)
	fmt.Printf("✓ Name:          %s\n", manifest.Name)
	fmt.Printf("✓ Version:       %s\n", manifest.Version)
	fmt.Printf("✓ API Version:   %s\n", manifest.APIVersion)
	fmt.Printf("✓ Language:      %s\n", manifest.Runtime.Language)
	fmt.Printf("✓ Command:       %s\n", manifest.Runtime.Command)
	fmt.Printf("✓ Permissions:   %s\n", strings.Join(manifest.Permissions, ", "))
	fmt.Printf("✓ Hooks:         %s\n", strings.Join(manifest.Hooks, ", "))
	fmt.Printf("✓ Failure Policy:%s\n", manifest.FailurePolicy)
	fmt.Printf("✓ Timeout:       %dms\n", manifest.TimeoutMs)
	fmt.Printf("\nPlugin manifest is valid and ready for Naagmani OS.\n")

	return nil
}
