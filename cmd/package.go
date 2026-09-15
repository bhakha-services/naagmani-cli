package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PackageOptions parameters for creating a plugin distribution archive.
type PackageOptions struct {
	Dir        string
	TargetOS   string
	TargetArch string
	OutputDir  string
}

// PackageResult represents the packaged archive and its cryptographic checksum.
type PackageResult struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Protocol     string `json:"protocol"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	SHA256       string `json:"sha256"`
	Artifact     string `json:"artifact"`
	ArtifactPath string `json:"artifact_path"`
}

// RunPluginPackage creates a deterministic .tar.gz archive containing the plugin binary and plugin.json.
func RunPluginPackage(opts PackageOptions) (*PackageResult, error) {
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
		return nil, fmt.Errorf("resolving absolute directory: %w", err)
	}

	manifestPath := filepath.Join(absDir, "plugin.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	// 1. Build if not already built or ensure dist binary exists
	targetDistDir := filepath.Join(absDir, opts.OutputDir, manifest.Name)
	distManifestPath := filepath.Join(targetDistDir, "plugin.json")
	if _, err := os.Stat(distManifestPath); os.IsNotExist(err) {
		fmt.Printf("Dist manifest not found in %s, building first...\n", targetDistDir)
		buildOpts := BuildOptions{
			Dir:        absDir,
			TargetOS:   opts.TargetOS,
			TargetArch: opts.TargetArch,
			OutputDir:  opts.OutputDir,
		}
		if _, err := RunPluginBuild(buildOpts); err != nil {
			return nil, fmt.Errorf("pre-package build failed: %w", err)
		}
	}

	// 2. Prepare target package archive name
	// Format: <name>-<version>-<os>-<arch>.tar.gz
	tarGzName := fmt.Sprintf("%s-%s-%s-%s.tar.gz", manifest.Name, manifest.Version, opts.TargetOS, opts.TargetArch)
	outputDir := filepath.Join(absDir, opts.OutputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}
	tarGzPath := filepath.Join(outputDir, tarGzName)

	fmt.Printf("Naagmani Plugin Package\n\n")
	fmt.Printf("Package:      %s\n", tarGzName)
	fmt.Printf("Destination:  %s\n", tarGzPath)

	// 3. Create .tar.gz archive
	outFile, err := os.Create(tarGzPath)
	if err != nil {
		return nil, fmt.Errorf("creating archive file: %w", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Deterministic timestamp for reproducible builds
	fixedTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	// Walk all files in targetDistDir and add them to tar archive
	err = filepath.Walk(targetDistDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(targetDistDir, path)
		if err != nil {
			return err
		}
		tarName := filepath.ToSlash(relPath)

		mode := int64(0644)
		if info.Mode()&0111 != 0 || strings.HasSuffix(tarName, ".exe") || tarName == manifest.Name {
			mode = 0755
		}

		header := &tar.Header{
			Name:     tarName,
			Mode:     mode,
			Size:     info.Size(),
			ModTime:  fixedTime,
			Typeflag: tar.TypeReg,
			Uid:      0,
			Gid:      0,
			Uname:    "naagmani",
			Gname:    "naagmani",
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("writing tar header for %s: %w", tarName, err)
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening file %s: %w", path, err)
		}
		defer f.Close()

		if _, err := io.Copy(tarWriter, f); err != nil {
			return fmt.Errorf("copying %s into tar: %w", tarName, err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking dist files to package: %w", err)
	}

	// Explicitly close tarWriter and gzWriter to flush all bytes to disk before computing SHA-256
	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("closing tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip writer: %w", err)
	}
	if err := outFile.Close(); err != nil {
		return nil, fmt.Errorf("closing output file: %w", err)
	}

	// 4. Compute SHA-256 of the package archive
	pkgHash, err := computeFileSHA256(tarGzPath)
	if err != nil {
		return nil, fmt.Errorf("calculating package checksum: %w", err)
	}

	// 5. Generate package metadata JSON
	result := &PackageResult{
		Name:         manifest.Name,
		Version:      manifest.Version,
		Protocol:     "naagmani.plugin/v1",
		OS:           opts.TargetOS,
		Architecture: opts.TargetArch,
		SHA256:       pkgHash,
		Artifact:     tarGzName,
		ArtifactPath: tarGzPath,
	}

	metaData, _ := json.MarshalIndent(result, "", "  ")
	metaName := strings.TrimSuffix(tarGzName, ".tar.gz") + ".json"
	metaPath := filepath.Join(outputDir, metaName)
	_ = os.WriteFile(metaPath, metaData, 0644)

	fmt.Printf("✓ Archive created: %s\n", tarGzName)
	fmt.Printf("✓ Checksum (SHA-256): %s\n", pkgHash)
	fmt.Printf("✓ Package metadata: %s\n\n", metaPath)

	return result, nil
}
