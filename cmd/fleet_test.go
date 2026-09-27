package cmd

import (
	"testing"
)

func TestCLI_FleetCommands(t *testing.T) {
	t.Run("Fleet Help", func(t *testing.T) {
		err := RunFleet([]string{"help"})
		if err != nil {
			t.Fatalf("expected RunFleet('help') to succeed, got %v", err)
		}
	})

	t.Run("Fleet List Output", func(t *testing.T) {
		err := RunFleet([]string{"list"})
		if err != nil {
			t.Fatalf("expected fleet list to succeed, got %v", err)
		}
	})

	t.Run("Fleet List JSON", func(t *testing.T) {
		err := RunFleet([]string{"list", "--json"})
		if err != nil {
			t.Fatalf("expected fleet list --json to succeed, got %v", err)
		}
	})

	t.Run("Fleet Info Output", func(t *testing.T) {
		err := RunFleet([]string{"info", "inst_prod_us_east_1"})
		if err != nil {
			t.Fatalf("expected fleet info to succeed, got %v", err)
		}
	})

	t.Run("Fleet Command Dispatch", func(t *testing.T) {
		err := RunFleet([]string{"command", "inst_prod_us_east_1", "START_UPDATE", "--params", `{"target_version":"v2.2.0"}`})
		if err != nil {
			t.Fatalf("expected fleet command to succeed, got %v", err)
		}
	})

	t.Run("Fleet Rollout Dispatch", func(t *testing.T) {
		err := RunFleet([]string{"rollout", "v2.2.0", "--strategy", "CANARY", "--installations", "inst_alpha,inst_beta,inst_gamma"})
		if err != nil {
			t.Fatalf("expected fleet rollout to succeed, got %v", err)
		}
	})
}
