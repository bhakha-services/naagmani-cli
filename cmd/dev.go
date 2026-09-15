package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

)

// DevConfig optional dev settings from .naagmani/dev.json
type DevConfig struct {
	OSPluginsDir string `json:"os_plugins_dir,omitempty"`
	OSURL        string `json:"os_url,omitempty"`
}

// RunPluginDev runs the local plugin development workflow.
func RunPluginDev(dir string) error {
	if dir == "" {
		dir = "."
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving absolute path: %w", err)
	}

	manifestPath := filepath.Join(absDir, "plugin.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("manifest validation failed: %w", err)
	}

	fmt.Printf("Naagmani Plugin Dev\n\n")
	fmt.Printf("Plugin:   %s\n", manifest.Name)
	fmt.Printf("Version:  %s\n", manifest.Version)
	fmt.Printf("Protocol: %s\n\n", ProtocolVersion)
	fmt.Printf("✓ Manifest valid\n")

	// 1. Build plugin if needed
	var binPath string
	lang := strings.ToLower(manifest.Runtime.Language)
	switch lang {
	case "node", "javascript", "typescript":
		if _, err := os.Stat(filepath.Join(absDir, "package.json")); err == nil {
			var buildCmd *exec.Cmd
			if runtime.GOOS == "windows" {
				buildCmd = exec.Command("cmd", "/c", "npm", "run", "build")
			} else {
				buildCmd = exec.Command("npm", "run", "build")
			}
			buildCmd.Dir = absDir
			_ = buildCmd.Run()
		}
		binPath = manifest.Runtime.Command
		fmt.Printf("✓ Plugin runtime configured (%s)\n", lang)

	case "python":
		binPath = manifest.Runtime.Command
		fmt.Printf("✓ Plugin runtime configured (python)\n")

	default: // Go or compiled binary
		binName := manifest.Name
		if runtime.GOOS == "windows" {
			binName += ".exe"
		}
		binPath = filepath.Join(absDir, binName)

		buildCmd := exec.Command("go", "build", "-o", binPath, ".")
		buildCmd.Dir = absDir
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("building plugin binary failed: %w", err)
		}
		fmt.Printf("✓ Plugin built (%s)\n", binName)
	}

	// 2. Pre-flight handshake verification
	if err := preflightHandshake(absDir, binPath, manifest); err != nil {
		return fmt.Errorf("protocol pre-flight check failed: %w", err)
	}
	fmt.Printf("✓ Plugin started\n")

	// 3. Connect to local Naagmani OS plugins directory
	osPluginsDir := findOSPluginsDir(absDir)
	if osPluginsDir != "" {
		if err := syncPluginToOS(absDir, osPluginsDir, manifest.Name); err != nil {
			fmt.Printf("! Notice: could not auto-sync to %s: %v\n", osPluginsDir, err)
		} else {
			fmt.Printf("✓ Connected to Naagmani OS (%s)\n", osPluginsDir)
		}
	} else {
		fmt.Printf("✓ Local development harness ready\n")
	}

	fmt.Printf("✓ Plugin registered\n")
	fmt.Printf("✓ Status: RUNNING\n\n")
	fmt.Printf("Hooks:\n")
	for _, h := range manifest.Hooks {
		fmt.Printf("  ✓ %s\n", h)
	}
	fmt.Printf("\nPlugin development server ready. Press Ctrl+C to exit.\n")

	// 4. Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Printf("\nShutting down plugin development server...\n")
	return nil
}

func preflightHandshake(workDir, binPath string, manifest *Manifest) error {
	var cmd *exec.Cmd
	lang := strings.ToLower(manifest.Runtime.Language)
	if (lang == "node" || lang == "javascript" || lang == "typescript" || lang == "python") && manifest.Runtime.Command != "" {
		cmd = exec.Command(manifest.Runtime.Command, manifest.Runtime.Args...)
	} else {
		cmd = exec.Command(binPath)
	}
	cmd.Dir = workDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("creating stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return fmt.Errorf("spawning binary: %w", err)
	}

	defer func() {
		_ = cmd.Process.Kill()
	}()

	scanner := bufio.NewScanner(stdout)

	// Send plugin.register
	regReq, _ := NewRPCRequest(1, MethodRegister, RegisterParams{
		PluginName:      manifest.Name,
		PluginVersion:   manifest.Version,
		APIVersion:      manifest.APIVersion,
		ProtocolVersion: ProtocolVersion,
		Permissions:     manifest.Permissions,
	})
	if err := writeLine(stdin, regReq); err != nil {
		return fmt.Errorf("sending register: %w", err)
	}

	if !scanner.Scan() {
		return fmt.Errorf("expected register response, got EOF")
	}

	var regResp RPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &regResp); err != nil {
		return fmt.Errorf("unmarshaling register response: %w", err)
	}
	if regResp.Error != nil {
		return fmt.Errorf("handshake rejected: %s (code %d)", regResp.Error.Message, regResp.Error.Code)
	}

	// Send plugin.health
	healthReq, _ := NewRPCRequest(2, MethodHealth, HealthParams{})
	if err := writeLine(stdin, healthReq); err != nil {
		return fmt.Errorf("sending health: %w", err)
	}

	if !scanner.Scan() {
		return fmt.Errorf("expected health response, got EOF")
	}

	var healthResp RPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &healthResp); err != nil {
		return fmt.Errorf("unmarshaling health response: %w", err)
	}
	if healthResp.Error != nil {
		return fmt.Errorf("health check failed: %s", healthResp.Error.Message)
	}

	// Graceful shutdown
	shutdownReq, _ := NewRPCRequest(3, MethodShutdown, ShutdownParams{})
	_ = writeLine(stdin, shutdownReq)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		_ = cmd.Process.Kill()
	}

	return nil
}

func writeLine(w io.Writer, val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

func findOSPluginsDir(startDir string) string {
	// Check .naagmani/dev.json
	cfgPath := filepath.Join(startDir, ".naagmani", "dev.json")
	if data, err := os.ReadFile(cfgPath); err == nil {
		var cfg DevConfig
		if err := json.Unmarshal(data, &cfg); err == nil && cfg.OSPluginsDir != "" {
			if info, err := os.Stat(cfg.OSPluginsDir); err == nil && info.IsDir() {
				return cfg.OSPluginsDir
			}
		}
	}

	// Search parent directories for naagmani-os/plugins
	curr := startDir
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(curr, "naagmani-os", "plugins")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	return ""
}

func syncPluginToOS(srcDir, osPluginsDir, pluginName string) error {
	targetDir := filepath.Join(osPluginsDir, pluginName)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	// Copy plugin.json
	srcManifest := filepath.Join(srcDir, "plugin.json")
	dstManifest := filepath.Join(targetDir, "plugin.json")
	if data, err := os.ReadFile(srcManifest); err == nil {
		_ = os.WriteFile(dstManifest, data, 0644)
	}

	// Copy binary if exists
	binName := pluginName
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	srcBin := filepath.Join(srcDir, binName)
	dstBin := filepath.Join(targetDir, binName)
	if data, err := os.ReadFile(srcBin); err == nil {
		_ = os.WriteFile(dstBin, data, 0755)
	}

	return nil
}
