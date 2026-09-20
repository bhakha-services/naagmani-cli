package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLI_CreateAndBuild(t *testing.T) {
	tempDir := t.TempDir()
	pluginName := "test-scaffold-plugin"

	err := RunPluginCreate(PluginCreateOptions{
		Name:     pluginName,
		Language: "go",
		DestDir:  filepath.Join(tempDir, pluginName),
	})
	if err != nil {
		t.Fatalf("RunPluginCreate failed: %v", err)
	}

	targetDir := filepath.Join(tempDir, pluginName)

	// Verify files created
	requiredFiles := []string{"plugin.json", "main.go", "go.mod", "README.md", ".gitignore"}
	for _, f := range requiredFiles {
		p := filepath.Join(targetDir, f)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Validate manifest
	if err := RunPluginValidate(targetDir); err != nil {
		t.Errorf("validation of generated plugin failed: %v", err)
	}

	// Build the plugin
	binName := pluginName
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	buildCmd := exec.Command("go", "build", "-o", binName, ".")
	buildCmd.Dir = targetDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\nOutput: %s", err, string(out))
	}

	binPath := filepath.Join(targetDir, binName)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Errorf("expected binary %s to exist after build", binPath)
	}

	// Preflight handshake test
	manifest, err := LoadManifest(filepath.Join(targetDir, "plugin.json"))
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	if err := preflightHandshake(targetDir, binPath, manifest); err != nil {
		t.Errorf("preflightHandshake failed on generated plugin: %v", err)
	}
}

func TestCLI_Validate_InvalidManifests(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name     string
		manifest string
	}{
		{
			name:     "missing name",
			manifest: `{"version": "1.0.0", "api_version": "v1", "runtime": {"language": "go", "command": "./bin"}, "permissions": ["request.read"]}`,
		},
		{
			name:     "unsupported api_version",
			manifest: `{"name": "p", "version": "1.0.0", "api_version": "v2", "runtime": {"language": "go", "command": "./bin"}, "permissions": ["request.read"]}`,
		},
		{
			name:     "missing permissions",
			manifest: `{"name": "p", "version": "1.0.0", "api_version": "v1", "runtime": {"language": "go", "command": "./bin"}, "permissions": []}`,
		},
		{
			name:     "path traversal in command",
			manifest: `{"name": "p", "version": "1.0.0", "api_version": "v1", "runtime": {"language": "go", "command": "../../../evil"}, "permissions": ["request.read"]}`,
		},
		{
			name:     "unknown hook",
			manifest: `{"name": "p", "version": "1.0.0", "api_version": "v1", "runtime": {"language": "go", "command": "./bin"}, "permissions": ["request.read"], "hooks": ["invalid.hook"]}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pDir := filepath.Join(tempDir, tc.name)
			_ = os.MkdirAll(pDir, 0755)
			_ = os.WriteFile(filepath.Join(pDir, "plugin.json"), []byte(tc.manifest), 0644)

			err := RunPluginValidate(pDir)
			if err == nil {
				t.Errorf("expected validation error for case %q, but got success", tc.name)
			}
		})
	}
}

func TestCLI_BuildAndPackage(t *testing.T) {
	tempDir := t.TempDir()
	pluginName := "hello-dist-plugin"
	targetDir := filepath.Join(tempDir, pluginName)

	if err := RunPluginCreate(PluginCreateOptions{
		Name:     pluginName,
		Language: "go",
		DestDir:  targetDir,
	}); err != nil {
		t.Fatalf("RunPluginCreate failed: %v", err)
	}

	// 1. Test RunPluginBuild
	buildResult, err := RunPluginBuild(BuildOptions{
		Dir:        targetDir,
		TargetOS:   runtime.GOOS,
		TargetArch: runtime.GOARCH,
		OutputDir:  "dist",
	})
	if err != nil {
		t.Fatalf("RunPluginBuild failed: %v", err)
	}

	if buildResult.Name != pluginName {
		t.Errorf("expected name %s, got %s", pluginName, buildResult.Name)
	}
	if buildResult.SHA256 == "" {
		t.Errorf("expected non-empty SHA256")
	}
	if _, err := os.Stat(buildResult.BinaryPath); os.IsNotExist(err) {
		t.Errorf("binary does not exist at %s", buildResult.BinaryPath)
	}

	// 2. Test RunPluginPackage
	pkgResult, err := RunPluginPackage(PackageOptions{
		Dir:        targetDir,
		TargetOS:   runtime.GOOS,
		TargetArch: runtime.GOARCH,
		OutputDir:  "dist",
	})
	if err != nil {
		t.Fatalf("RunPluginPackage failed: %v", err)
	}

	if pkgResult.SHA256 == "" {
		t.Errorf("expected non-empty package SHA256")
	}
	if _, err := os.Stat(pkgResult.ArtifactPath); os.IsNotExist(err) {
		t.Errorf("package archive does not exist at %s", pkgResult.ArtifactPath)
	}
}

func TestCLI_CreateMultiLanguage(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Test Node plugin creation and validation
	nodeDir := filepath.Join(tempDir, "test-node-plugin")
	if err := RunPluginCreate(PluginCreateOptions{
		Name:     "test-node-plugin",
		Language: "node",
		DestDir:  nodeDir,
	}); err != nil {
		t.Fatalf("creating node plugin failed: %v", err)
	}

	for _, f := range []string{"plugin.json", "package.json", "tsconfig.json", "index.ts", "README.md"} {
		if _, err := os.Stat(filepath.Join(nodeDir, f)); os.IsNotExist(err) {
			t.Errorf("expected node file %s to exist", f)
		}
	}
	if err := RunPluginValidate(nodeDir); err != nil {
		t.Errorf("node manifest validation failed: %v", err)
	}

	// 2. Test Python plugin creation and validation
	pyDir := filepath.Join(tempDir, "test-py-plugin")
	if err := RunPluginCreate(PluginCreateOptions{
		Name:     "test-py-plugin",
		Language: "python",
		DestDir:  pyDir,
	}); err != nil {
		t.Fatalf("creating python plugin failed: %v", err)
	}

	for _, f := range []string{"plugin.json", "pyproject.toml", "main.py", "README.md"} {
		if _, err := os.Stat(filepath.Join(pyDir, f)); os.IsNotExist(err) {
			t.Errorf("expected python file %s to exist", f)
		}
	}
	if err := RunPluginValidate(pyDir); err != nil {
		t.Errorf("python manifest validation failed: %v", err)
	}

	// 3. Test building and packaging Python plugin
	buildRes, err := RunPluginBuild(BuildOptions{
		Dir:        pyDir,
		TargetOS:   runtime.GOOS,
		TargetArch: runtime.GOARCH,
		OutputDir:  "dist",
	})
	if err != nil {
		t.Fatalf("building python plugin failed: %v", err)
	}
	if buildRes.Name != "test-py-plugin" {
		t.Errorf("expected name test-py-plugin, got %s", buildRes.Name)
	}

	pkgRes, err := RunPluginPackage(PackageOptions{
		Dir:        pyDir,
		TargetOS:   runtime.GOOS,
		TargetArch: runtime.GOARCH,
		OutputDir:  "dist",
	})
	if err != nil {
		t.Fatalf("packaging python plugin failed: %v", err)
	}
	if pkgRes.SHA256 == "" {
		t.Errorf("expected non-empty sha256 for packaged python plugin")
	}
	if _, err := os.Stat(pkgRes.ArtifactPath); os.IsNotExist(err) {
		t.Errorf("expected python archive %s to exist", pkgRes.ArtifactPath)
	}
}

func TestCLI_Init(t *testing.T) {
	tempDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// 1. Initial creation
	err := RunInit(InitOptions{
		ProjectName: "my-test-ai-app",
		Environment: "test",
		Force:       false,
	})
	if err != nil {
		t.Fatalf("RunInit failed: %v", err)
	}

	content, err := os.ReadFile("naagmani.yaml")
	if err != nil {
		t.Fatalf("reading naagmani.yaml: %v", err)
	}
	if !strings.Contains(string(content), "my-test-ai-app") {
		t.Errorf("expected config to contain project name")
	}
	if !strings.Contains(string(content), "test") {
		t.Errorf("expected config to contain environment test")
	}

	// 2. Duplicate without force should error
	err = RunInit(InitOptions{
		ProjectName: "duplicate",
		Force:       false,
	})
	if err == nil {
		t.Errorf("expected error when running init without --force on existing naagmani.yaml")
	}

	// 3. Duplicate with force should succeed
	err = RunInit(InitOptions{
		ProjectName: "forced-app",
		Force:       true,
	})
	if err != nil {
		t.Errorf("expected force init to succeed: %v", err)
	}
}

func TestCLI_Config(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	// Test Set
	if err := RunConfig([]string{"set", "org_id", "org_test_123"}); err != nil {
		t.Fatalf("config set failed: %v", err)
	}
	if err := RunConfig([]string{"set", "project_id", "proj_test_456"}); err != nil {
		t.Fatalf("config set project_id failed: %v", err)
	}

	// Verify via LoadCLIConfig
	cfg, err := LoadCLIConfig()
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if cfg.OrgID != "org_test_123" {
		t.Errorf("expected org_id to be org_test_123, got %s", cfg.OrgID)
	}
	if cfg.ProjectID != "proj_test_456" {
		t.Errorf("expected project_id to be proj_test_456, got %s", cfg.ProjectID)
	}

	// Test List
	if err := RunConfig([]string{"list"}); err != nil {
		t.Errorf("config list failed: %v", err)
	}

	// Test Get
	if err := RunConfig([]string{"get", "org_id"}); err != nil {
		t.Errorf("config get failed: %v", err)
	}

	// Test Logout
	if err := RunLogout(); err != nil {
		t.Fatalf("RunLogout failed: %v", err)
	}
	cfgAfterLogout, _ := LoadCLIConfig()
	if cfgAfterLogout.Token != "" || cfgAfterLogout.APIKey != "" {
		t.Errorf("expected tokens to be cleared after logout")
	}
}

func TestCLI_Doctor(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	err := RunDoctor("1.0.0")
	if err != nil {
		t.Errorf("RunDoctor failed: %v", err)
	}
}
