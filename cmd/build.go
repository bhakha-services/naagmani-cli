package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BuildOptions parameters for building a plugin artifact.
type BuildOptions struct {
	Dir        string
	TargetOS   string
	TargetArch string
	OutputDir  string
}

// BuildResult represents the outcome and metadata of a plugin build.
type BuildResult struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Protocol     string `json:"protocol"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	SHA256       string `json:"sha256"`
	BinaryPath   string `json:"binary_path"`
	OutputDir    string `json:"output_dir"`
}

// RunPluginBuild compiles the plugin for a target OS and architecture.
func RunPluginBuild(opts BuildOptions) (*BuildResult, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	if opts.TargetOS == "" {
		opts.TargetOS = "linux"
	}
	if opts.TargetArch == "" {
		opts.TargetArch = "amd64"
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "dist"
	}

	absDir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("resolving absolute plugin dir: %w", err)
	}

	manifestPath := filepath.Join(absDir, "plugin.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	fmt.Printf("Naagmani Plugin Build\n\n")
	fmt.Printf("Plugin:       %s\n", manifest.Name)
	fmt.Printf("Version:      %s\n", manifest.Version)
	fmt.Printf("Target OS:    %s\n", opts.TargetOS)
	fmt.Printf("Target Arch:  %s\n", opts.TargetArch)

	// Output directory: dist/<plugin-name>/
	targetDistDir := filepath.Join(absDir, opts.OutputDir, manifest.Name)
	if err := os.MkdirAll(targetDistDir, 0755); err != nil {
		return nil, fmt.Errorf("creating dist directory: %w", err)
	}

	var binHash string
	var primaryPath string
	distManifest := *manifest

	lang := strings.ToLower(manifest.Runtime.Language)
	switch lang {
	case "go", "golang":
		binName := manifest.Name
		if opts.TargetOS == "windows" {
			binName += ".exe"
		}
		targetBinPath := filepath.Join(targetDistDir, binName)

		cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", targetBinPath, ".")
		cmd.Dir = absDir
		cmd.Env = append(os.Environ(),
			"GOOS="+opts.TargetOS,
			"GOARCH="+opts.TargetArch,
			"CGO_ENABLED=0",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("compiling Go plugin binary failed: %w", err)
		}

		hash, err := computeFileSHA256(targetBinPath)
		if err != nil {
			return nil, fmt.Errorf("calculating binary checksum: %w", err)
		}
		binHash = hash
		primaryPath = targetBinPath

		cmdPrefix := "./"
		if opts.TargetOS == "windows" {
			cmdPrefix = ".\\"
		}
		distManifest.Runtime.Command = cmdPrefix + binName

	case "node", "nodejs", "typescript", "ts":
		// Build TypeScript if package.json and tsconfig.json exist
		if _, err := os.Stat(filepath.Join(absDir, "package.json")); err == nil {
			if _, err := os.Stat(filepath.Join(absDir, "tsconfig.json")); err == nil {
				// Run tsc / npm run build
				buildCmd := exec.Command("npm", "run", "build")
				if runtime.GOOS == "windows" {
					buildCmd = exec.Command("cmd", "/c", "npm run build")
				}
				buildCmd.Dir = absDir
				buildCmd.Stdout = os.Stdout
				buildCmd.Stderr = os.Stderr
				if err := buildCmd.Run(); err != nil {
					// Fallback to npx tsc
					tscCmd := exec.Command("npx", "tsc")
					if runtime.GOOS == "windows" {
						tscCmd = exec.Command("cmd", "/c", "npx tsc")
					}
					tscCmd.Dir = absDir
					_ = tscCmd.Run()
				}
			}

			// Copy dist directory if it exists
			srcDist := filepath.Join(absDir, "dist")
			if info, err := os.Stat(srcDist); err == nil && info.IsDir() {
				_ = copyDir(srcDist, filepath.Join(targetDistDir, "dist"))
			}
			// Copy package.json
			_ = copyFile(filepath.Join(absDir, "package.json"), filepath.Join(targetDistDir, "package.json"))
			// Copy node_modules if present
			srcNodeModules := filepath.Join(absDir, "node_modules")
			if info, err := os.Stat(srcNodeModules); err == nil && info.IsDir() {
				_ = copyDir(srcNodeModules, filepath.Join(targetDistDir, "node_modules"))
			}
			// Copy root js files if present
			if entries, err := os.ReadDir(absDir); err == nil {
				for _, e := range entries {
					if !e.IsDir() && (strings.HasSuffix(e.Name(), ".js") || strings.HasSuffix(e.Name(), ".mjs") || strings.HasSuffix(e.Name(), ".cjs")) {
						_ = copyFile(filepath.Join(absDir, e.Name()), filepath.Join(targetDistDir, e.Name()))
					}
				}
			}
		}

		distManifest.Runtime.Command = manifest.Runtime.Command
		distManifest.Runtime.Args = manifest.Runtime.Args

		// Compute hash of the primary entrypoint
		entryName := "index.js"
		if len(manifest.Runtime.Args) > 0 {
			entryName = manifest.Runtime.Args[0]
		}
		candidateEntry := filepath.Join(targetDistDir, entryName)
		if _, err := os.Stat(candidateEntry); err == nil {
			primaryPath = candidateEntry
			binHash, _ = computeFileSHA256(candidateEntry)
		} else {
			primaryPath = filepath.Join(targetDistDir, "plugin.json")
			binHash = "0000000000000000000000000000000000000000000000000000000000000000"
		}

	case "python", "py":
		// Copy all python source files and modules
		if entries, err := os.ReadDir(absDir); err == nil {
			for _, e := range entries {
				if e.Name() == "dist" || e.Name() == "__pycache__" || strings.HasPrefix(e.Name(), ".") {
					continue
				}
				srcPath := filepath.Join(absDir, e.Name())
				dstPath := filepath.Join(targetDistDir, e.Name())
				if e.IsDir() {
					_ = copyDir(srcPath, dstPath)
				} else {
					_ = copyFile(srcPath, dstPath)
				}
			}
		}

		distManifest.Runtime.Command = manifest.Runtime.Command
		distManifest.Runtime.Args = manifest.Runtime.Args

		entryName := "main.py"
		if len(manifest.Runtime.Args) > 0 {
			entryName = manifest.Runtime.Args[0]
		}
		candidateEntry := filepath.Join(targetDistDir, entryName)
		if _, err := os.Stat(candidateEntry); err == nil {
			primaryPath = candidateEntry
			binHash, _ = computeFileSHA256(candidateEntry)
		} else {
			primaryPath = filepath.Join(targetDistDir, "plugin.json")
			binHash = "0000000000000000000000000000000000000000000000000000000000000000"
		}

	default:
		return nil, fmt.Errorf("unsupported plugin language: %s", manifest.Runtime.Language)
	}

	distManifestData, err := json.MarshalIndent(distManifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling dist manifest: %w", err)
	}

	distManifestPath := filepath.Join(targetDistDir, "plugin.json")
	if err := os.WriteFile(distManifestPath, distManifestData, 0644); err != nil {
		return nil, fmt.Errorf("writing dist plugin.json: %w", err)
	}

	// Write build metadata
	result := &BuildResult{
		Name:         manifest.Name,
		Version:      manifest.Version,
		Protocol:     "naagmani.plugin/v1",
		OS:           opts.TargetOS,
		Architecture: opts.TargetArch,
		SHA256:       binHash,
		BinaryPath:   primaryPath,
		OutputDir:    targetDistDir,
	}

	metaData, _ := json.MarshalIndent(result, "", "  ")
	metaPath := filepath.Join(targetDistDir, "manifest.json")
	_ = os.WriteFile(metaPath, metaData, 0644)

	fmt.Printf("✓ Build artifact ready: %s\n", primaryPath)
	fmt.Printf("✓ Checksum (SHA-256): %s\n", binHash)
	fmt.Printf("✓ Build output ready in: %s\n\n", targetDistDir)

	return result, nil
}

// computeFileSHA256 calculates the sha256 checksum of a file.
func computeFileSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HostTarget returns the host OS and architecture for local testing.
func HostTarget() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}

