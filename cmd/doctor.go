package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// RunDoctor performs diagnostic checks on the developer environment.
func RunDoctor(cliVersion string) error {
	fmt.Println("Naagmani Environment Diagnostics (doctor):")
	fmt.Println("==========================================")

	// 1. Core Tooling
	fmt.Printf("✓ Naagmani CLI:        v%s (%s/%s)\n", cliVersion, runtime.GOOS, runtime.GOARCH)

	// 2. Language Runtimes
	if out, err := exec.Command("go", "version").Output(); err == nil {
		fmt.Printf("✓ Go Runtime:          %s", strings.TrimPrefix(string(out), "go version "))
	} else {
		fmt.Println("ℹ Go Runtime:          Not detected (required for compiling Go plugins)")
	}

	if out, err := exec.Command("node", "--version").Output(); err == nil {
		fmt.Printf("✓ Node.js Runtime:     %s\n", strings.TrimSpace(string(out)))
	} else {
		fmt.Println("ℹ Node.js Runtime:     Not detected (required for Node.js plugins)")
	}

	pyFound := false
	for _, pyCmd := range []string{"python", "python3"} {
		if out, err := exec.Command(pyCmd, "--version").Output(); err == nil {
			fmt.Printf("✓ Python Runtime:      %s\n", strings.TrimSpace(string(out)))
			pyFound = true
			break
		}
	}
	if !pyFound {
		fmt.Println("ℹ Python Runtime:      Not detected (required for Python plugins)")
	}

	// 3. Container Runtime
	if err := exec.Command("docker", "info").Run(); err == nil {
		fmt.Println("✓ Docker Engine:       Running and responsive")
	} else {
		fmt.Println("ℹ Docker Engine:       Not running or not installed (required for local multi-container stack)")
	}

	// 4. Configuration & Auth
	cfg, err := LoadCLIConfig()
	if err != nil {
		fmt.Printf("✖ CLI Configuration:   Error loading config: %v\n", err)
	} else {
		token := cfg.Token
		if token == "" && cfg.APIKey != "" {
			token = cfg.APIKey
		}
		if token != "" {
			fmt.Printf("✓ Authentication:      Active token configured (%s)\n", maskSecret(token))
			if cfg.Email != "" {
				fmt.Printf("  Identity:            %s\n", cfg.Email)
			}
		} else {
			fmt.Println("ℹ Authentication:      Unauthenticated (run 'naagmani login' to authenticate)")
		}

		// 5. Connectivity Probes
		client := &http.Client{Timeout: 2 * time.Second}

		// Probe OS
		osURL := strings.TrimRight(cfg.OSEndpoint, "/")
		if osURL == "" {
			osURL = "http://localhost:8080"
		}
		resp, err := client.Get(osURL + "/v1/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			fmt.Printf("✓ Naagmani OS:         Online at %s\n", osURL)
			resp.Body.Close()
		} else {
			fmt.Printf("ℹ Naagmani OS:         Offline at %s (run 'naagmani dev' to launch)\n", osURL)
		}

		// Probe Cloud
		cloudURL := strings.TrimRight(cfg.CloudURL, "/")
		if cloudURL == "" {
			cloudURL = "http://localhost:8081"
		}
		resp, err = client.Get(cloudURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			fmt.Printf("✓ Naagmani Cloud:      Online at %s\n", cloudURL)
			resp.Body.Close()
		} else {
			fmt.Printf("ℹ Naagmani Cloud:      Offline at %s\n", cloudURL)
		}
	}

	// 6. Project Context
	if _, err := os.Stat("naagmani.yaml"); err == nil {
		fmt.Println("✓ Project Context:     Found naagmani.yaml in working directory")
	} else if _, err := os.Stat("plugin.json"); err == nil {
		fmt.Println("✓ Plugin Context:      Found plugin.json in working directory")
	} else {
		fmt.Println("ℹ Working Directory:   No naagmani.yaml or plugin.json detected")
	}

	fmt.Println("\nDiagnostics complete.")
	return nil
}
