package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPlatformHelp(t *testing.T) {
	err := RunPlatform([]string{"help"})
	if err != nil {
		t.Fatalf("expected RunPlatform('help') to succeed, got %v", err)
	}
}

func TestPlatformSecretGeneration(t *testing.T) {
	key, err := generateRandomHex(32)
	if err != nil {
		t.Fatalf("failed generating random hex: %v", err)
	}
	if len(key) != 64 {
		t.Errorf("expected 64 hex characters (32 bytes), got %d", len(key))
	}
}

func TestPlatformInstallDryRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "naagmani-install-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = RunPlatform([]string{"install", "--dir", tmpDir, "--dry-run"})
	if err != nil {
		t.Fatalf("expected dry-run install to succeed, got %v", err)
	}
}

func TestComprehensivePreflightEngine(t *testing.T) {
	result := runComprehensivePreflight(3000, 8081)
	if len(result.Requirements) == 0 {
		t.Errorf("expected preflight to evaluate multiple requirements, got 0")
	}

	foundOS := false
	for _, req := range result.Requirements {
		if req.Name == "OS / Arch" {
			foundOS = true
			if !req.Passed {
				t.Errorf("expected OS check to pass")
			}
		}
	}
	if !foundOS {
		t.Errorf("OS / Arch check not found in preflight requirements")
	}
}

func TestInstallationLockLifecycle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "naagmani-lock-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	unlock, err := acquireInstallLock(tmpDir, false)
	if err != nil {
		t.Fatalf("failed acquiring initial install lock: %v", err)
	}

	lockFile := filepath.Join(tmpDir, ".naagmani", "install.lock")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Errorf("expected lock file to exist at %s", lockFile)
	}

	// Re-acquiring without force should fail due to active PID
	_, err2 := acquireInstallLock(tmpDir, false)
	if err2 == nil {
		t.Errorf("expected concurrent lock acquisition to fail, but succeeded")
	}

	// Release lock
	unlock()
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		t.Errorf("expected lock file to be removed after unlock")
	}
}

func TestInstallationProgressTracking(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "naagmani-progress-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	saveInstallProgress(tmpDir, StageConfigReady, "v2.0.0", "stable", 3000, 8081, nil)

	progressFile := filepath.Join(tmpDir, ".naagmani", "install_state.json")
	data, err := os.ReadFile(progressFile)
	if err != nil {
		t.Fatalf("failed reading install progress file: %v", err)
	}

	var progress InstallProgressState
	if err := json.Unmarshal(data, &progress); err != nil {
		t.Fatalf("failed parsing install progress: %v", err)
	}

	if progress.CurrentStage != StageConfigReady || progress.Version != "v2.0.0" {
		t.Errorf("unexpected progress state: %+v", progress)
	}
}

func TestInstallationLogRecording(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "naagmani-log-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	recordInstallLog(tmpDir, StageInit, "Starting installation test", nil)

	logFile := filepath.Join(tmpDir, ".naagmani", "install.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed reading install log: %v", err)
	}

	if len(data) == 0 {
		t.Errorf("expected install log to contain entries, got empty file")
	}
}

func TestPlatformInstallationIDPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "naagmani-inst-id-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.MkdirAll(filepath.Join(tmpDir, ".naagmani"), 0755)

	id1 := getOrCreateInstallationID(tmpDir)
	if len(id1) < 10 {
		t.Errorf("expected valid installation ID format, got %s", id1)
	}

	id2 := getOrCreateInstallationID(tmpDir)
	if id1 != id2 {
		t.Errorf("expected installation ID to be persistent, got %s vs %s", id1, id2)
	}
}

func TestEnvFileOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "naagmani-env-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envPath := filepath.Join(tmpDir, ".env")
	initialContent := "NAAGMANI_VERSION=v1.0.0\nPORTAL_PORT=3000\n"
	_ = os.WriteFile(envPath, []byte(initialContent), 0600)

	envMap := loadEnvMap(envPath)
	if envMap["NAAGMANI_VERSION"] != "v1.0.0" || envMap["PORTAL_PORT"] != "3000" {
		t.Errorf("unexpected env map: %v", envMap)
	}

	updateEnvVar(envPath, "NAAGMANI_VERSION", "v2.0.0")
	updatedMap := loadEnvMap(envPath)
	if updatedMap["NAAGMANI_VERSION"] != "v2.0.0" {
		t.Errorf("expected NAAGMANI_VERSION to be v2.0.0, got %s", updatedMap["NAAGMANI_VERSION"])
	}
}

func TestPlatformStateSerialization(t *testing.T) {
	state := PlatformState{
		InstallationID: "inst_12345",
		Version:        "v2.0.0",
		Channel:        "stable",
		Status:         "healthy",
		PortalPort:     3000,
		CloudPort:      8081,
		ComposeFile:    "docker-compose.production.yml",
		InstalledAt:    time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed marshaling state: %v", err)
	}

	var parsed PlatformState
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed unmarshaling state: %v", err)
	}

	if parsed.InstallationID != state.InstallationID || parsed.Version != state.Version {
		t.Errorf("mismatch in deserialized state: %v", parsed)
	}
}

func TestReleaseManifestResolution(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "naagmani-manifest-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifestPath := filepath.Join(tmpDir, "release.yaml")
	manifestContent := `schema_version: "naagmani.release/v1"
release:
  version: "v2.0.0"
  channel: "stable"
`
	_ = os.WriteFile(manifestPath, []byte(manifestContent), 0644)

	manifest, err := resolveAndVerifyReleaseManifest(tmpDir, "", "v2.0.0")
	if err != nil {
		t.Fatalf("failed resolving manifest: %v", err)
	}

	if manifest.SchemaVersion != "naagmani.release/v1" {
		t.Errorf("expected schema_version naagmani.release/v1, got %s", manifest.SchemaVersion)
	}
}

func TestLicenseLifecycleCLI(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "naagmani-lic-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Test status without license (Default Free fallback)
	if err := runLicenseStatus([]string{"--dir", tmpDir}); err != nil {
		t.Fatalf("failed running license status: %v", err)
	}

	// 2. Create mock signed license artifact
	licPayload := LicensePayload{
		Version:        2,
		Type:           "naagmani-license",
		LicenseID:      "lic_test_12345",
		InstallationID: "inst_test_abcde",
		Edition:        "enterprise",
		Features:       []string{"router", "byok", "advanced_routing", "enterprise_rbac"},
		IssuedAt:       time.Now().UTC(),
	}
	signed := SignedLicense{
		Payload:   licPayload,
		Signature: "dGVzdC1zaWduYXR1cmUtZWRlbmNvZGVkLXZhbGlk",
	}
	licBytes, _ := json.MarshalIndent(signed, "", "  ")
	licFile := filepath.Join(tmpDir, "test-license.json")
	if err := os.WriteFile(licFile, licBytes, 0644); err != nil {
		t.Fatalf("failed writing test license: %v", err)
	}

	// 3. Test license verify
	if err := runLicenseVerify([]string{licFile}); err != nil {
		t.Fatalf("failed verifying license: %v", err)
	}

	// 4. Test license activate
	if err := runLicenseActivate([]string{"--dir", tmpDir, licFile}); err != nil {
		t.Fatalf("failed activating license: %v", err)
	}

	// 5. Test status with activated license
	if err := runLicenseStatus([]string{"--dir", tmpDir}); err != nil {
		t.Fatalf("failed running license status after activation: %v", err)
	}
}
