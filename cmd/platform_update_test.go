package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type MockDeploymentBackend struct {
	DeployShouldFail bool
	HealthShouldFail bool
	DeploysCount     int
	RollbacksCount   int
	LastDeployedVer  string
	LastRollbackVer  string
}

func (m *MockDeploymentBackend) Name() string {
	return "mock-backend"
}

func (m *MockDeploymentBackend) Preflight(ctx context.Context, dir string) error {
	return nil
}

func (m *MockDeploymentBackend) PullRelease(ctx context.Context, dir string, manifest *CanonicalReleaseManifest) error {
	return nil
}

func (m *MockDeploymentBackend) Deploy(ctx context.Context, dir string, manifest *CanonicalReleaseManifest) error {
	m.DeploysCount++
	m.LastDeployedVer = manifest.Release.Version
	if m.DeployShouldFail {
		return errors.New("simulated container deployment failure")
	}
	return nil
}

func (m *MockDeploymentBackend) Restart(ctx context.Context, dir string) error {
	return nil
}

func (m *MockDeploymentBackend) Rollback(ctx context.Context, dir string, version string) error {
	m.RollbacksCount++
	m.LastRollbackVer = version
	return nil
}

func (m *MockDeploymentBackend) VerifyHealth(ctx context.Context, dir string, timeout time.Duration) error {
	if m.HealthShouldFail {
		return errors.New("simulated container health probe failure")
	}
	return nil
}

func TestPlatformUpdate_OrchestratedUpgradeLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	metaDir := filepath.Join(tmpDir, ".naagmani")
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		t.Fatalf("failed creating meta dir: %v", err)
	}

	// Write initial platform state (v2.0.0)
	initState := PlatformState{
		InstallationID: "inst_phase7_test",
		Version:        "v2.0.0",
		Channel:        "stable",
		InstalledAt:    time.Now().UTC().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().UTC().Add(-24 * time.Hour),
		Status:         "healthy",
		PortalPort:     3000,
		CloudPort:      8081,
	}
	sBytes, _ := json.MarshalIndent(initState, "", "  ")
	if err := os.WriteFile(filepath.Join(metaDir, "platform_state.json"), sBytes, 0644); err != nil {
		t.Fatalf("failed writing platform_state.json: %v", err)
	}

	// Write release manifest
	manifestContent := `schema_version: "naagmani.release/v1"
release:
  version: "v2.1.0"
  channel: "stable"
  released_at: "2026-10-15T12:00:00Z"
  min_cli_version: "1.0.0"
components:
  os:
    version: "v2.1.0"
    image: "ghcr.io/bhakha-services/naagmani-os:v2.1.0"
    digest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"
    migrations:
      required: true
      schema_version: "20261015_01"
  cloud:
    version: "v2.1.0"
    image: "ghcr.io/bhakha-services/naagmani-cloud:v2.1.0"
    digest: "sha256:2222222222222222222222222222222222222222222222222222222222222222"
    migrations:
      required: true
      schema_version: "20261015_01"
  developer:
    version: "v2.1.0"
    image: "ghcr.io/bhakha-services/naagmani-developer:v2.1.0"
    digest: "sha256:3333333333333333333333333333333333333333333333333333333333333333"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "release.yaml"), []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed writing release.yaml: %v", err)
	}

	t.Run("1. Successful Platform Upgrade (v2.0.0 -> v2.1.0)", func(t *testing.T) {
		mockBackend := &MockDeploymentBackend{}

		err := ExecuteOrchestratedUpgrade(tmpDir, "v2.1.0", "stable", "", mockBackend, nil)
		if err != nil {
			t.Fatalf("expected upgrade to succeed, got: %v", err)
		}

		if mockBackend.DeploysCount != 1 {
			t.Errorf("expected 1 deployment, got %d", mockBackend.DeploysCount)
		}
		if mockBackend.LastDeployedVer != "v2.1.0" {
			t.Errorf("expected deploy version v2.1.0, got %s", mockBackend.LastDeployedVer)
		}
		if mockBackend.RollbacksCount != 0 {
			t.Errorf("expected 0 rollbacks on success, got %d", mockBackend.RollbacksCount)
		}

		// Verify updated platform state
		var updatedState PlatformState
		stData, err := os.ReadFile(filepath.Join(metaDir, "platform_state.json"))
		if err != nil {
			t.Fatalf("reading state file: %v", err)
		}
		if err := json.Unmarshal(stData, &updatedState); err != nil {
			t.Fatalf("parsing state JSON: %v", err)
		}

		if updatedState.Version != "v2.1.0" {
			t.Errorf("expected state version v2.1.0, got %s", updatedState.Version)
		}
		if updatedState.PreviousVersion != "v2.0.0" {
			t.Errorf("expected state previous version v2.0.0, got %s", updatedState.PreviousVersion)
		}
		if updatedState.Status != "healthy" {
			t.Errorf("expected state status 'healthy', got %s", updatedState.Status)
		}

		// Verify migration state recorded
		var migrations []MigrationRecord
		migData, err := os.ReadFile(filepath.Join(metaDir, "migration_state.json"))
		if err != nil {
			t.Fatalf("reading migration state: %v", err)
		}
		if err := json.Unmarshal(migData, &migrations); err != nil {
			t.Fatalf("parsing migration JSON: %v", err)
		}
		if len(migrations) == 0 || migrations[len(migrations)-1].SchemaVersion != "20261015_01" {
			t.Errorf("expected migration 20261015_01 to be recorded, got %+v", migrations)
		}

		// Verify upgrade history logged
		var history []UpgradeHistoryRecord
		histData, err := os.ReadFile(filepath.Join(metaDir, "update_history.json"))
		if err != nil {
			t.Fatalf("reading history file: %v", err)
		}
		if err := json.Unmarshal(histData, &history); err != nil {
			t.Fatalf("parsing history JSON: %v", err)
		}
		if len(history) == 0 || history[0].Status != UpgradeStateCompleted {
			t.Errorf("expected history record with status COMPLETED, got %+v", history)
		}
	})

	t.Run("2. Failed Deployment Triggers Automatic Rollback", func(t *testing.T) {
		mockBackend := &MockDeploymentBackend{
			HealthShouldFail: true, // fails health checks post-deployment
		}

		err := ExecuteOrchestratedUpgrade(tmpDir, "v2.2.0", "stable", "", mockBackend, nil)
		if err == nil {
			t.Fatal("expected upgrade to fail and return error")
		}
		if !strings.Contains(err.Error(), "rolled back to v2.1.0") {
			t.Errorf("expected rollback message in error, got %v", err)
		}

		if mockBackend.RollbacksCount != 1 {
			t.Errorf("expected 1 rollback invocation, got %d", mockBackend.RollbacksCount)
		}
		if mockBackend.LastRollbackVer != "v2.1.0" {
			t.Errorf("expected rollback to v2.1.0, got %s", mockBackend.LastRollbackVer)
		}

		// Verify history logged rollback
		var history []UpgradeHistoryRecord
		histData, err := os.ReadFile(filepath.Join(metaDir, "update_history.json"))
		if err != nil {
			t.Fatalf("reading history: %v", err)
		}
		if err := json.Unmarshal(histData, &history); err != nil {
			t.Fatalf("parsing history: %v", err)
		}
		if history[0].Status != UpgradeStateRolledBack {
			t.Errorf("expected latest history status ROLLED_BACK, got %s", history[0].Status)
		}
	})

	t.Run("3. Interrupted Upgrade Recovery Engine", func(t *testing.T) {
		// Simulate crash during MIGRATING
		interruptedTxn := UpgradeTransaction{
			UpgradeID:      "upg_crash_test",
			InstallationID: "inst_phase7_test",
			FromVersion:    "v2.1.0",
			TargetVersion:  "v2.2.0",
			Channel:        "stable",
			State:          UpgradeStateMigrating,
			StartedAt:      time.Now().UTC().Add(-10 * time.Minute),
			InitiatedBy:    "operator",
		}
		saveUpgradeTransaction(tmpDir, &interruptedTxn)

		// Run recovery with force-rollback
		err := runPlatformRecovery([]string{"--dir", tmpDir, "--force-rollback"})
		if err != nil {
			t.Fatalf("expected recovery to succeed, got: %v", err)
		}

		// Active transaction file should be cleared
		_, err = os.ReadFile(filepath.Join(metaDir, "upgrade_transaction.json"))
		if !os.IsNotExist(err) {
			t.Errorf("expected upgrade_transaction.json to be deleted after recovery")
		}

		// Platform state must be intact at v2.1.0
		var recoveredState PlatformState
		stData, err := os.ReadFile(filepath.Join(metaDir, "platform_state.json"))
		if err != nil {
			t.Fatalf("reading state: %v", err)
		}
		if err := json.Unmarshal(stData, &recoveredState); err != nil {
			t.Fatalf("parsing state JSON: %v", err)
		}
		if recoveredState.Version != "v2.1.0" {
			t.Errorf("expected recovered version v2.1.0, got %s", recoveredState.Version)
		}
	})
}
