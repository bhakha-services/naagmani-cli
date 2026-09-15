package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RunWhoami inspects the currently active authentication state and identity.
func RunWhoami() error {
	cfg, err := LoadCLIConfig()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	token := cfg.Token
	if token == "" && cfg.APIKey != "" {
		token = cfg.APIKey
	}

	if token == "" {
		fmt.Println("Status: Unauthenticated")
		fmt.Println("Run 'naagmani login' or set NAAGMANI_TOKEN to authenticate.")
		return fmt.Errorf("not logged in")
	}

	// Verify token against Naagmani Cloud
	cloudURL := strings.TrimRight(cfg.CloudURL, "/")
	meURL := cloudURL + "/v1/auth/me"

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, meURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		// Network connectivity failure or offline mode: show local cached context
		fmt.Println("Status: Authenticated (Offline / Cached Context)")
		if cfg.Email != "" {
			fmt.Printf("Email:            %s\n", cfg.Email)
		}
		if cfg.OrgID != "" {
			fmt.Printf("Organization ID:  %s\n", cfg.OrgID)
		}
		if cfg.ProjectID != "" {
			fmt.Printf("Project ID:       %s\n", cfg.ProjectID)
		}
		if cfg.Environment != "" {
			fmt.Printf("Environment:      %s\n", cfg.Environment)
		}
		fmt.Printf("Cloud URL:        %s\n", cfg.CloudURL)
		fmt.Printf("Token:            %s\n", maskSecret(token))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		fmt.Println("Status: Invalid or Expired Token")
		fmt.Println("Please run 'naagmani login' to re-authenticate.")
		return fmt.Errorf("session expired or token invalid")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloud returned status %d", resp.StatusCode)
	}

	var userResp struct {
		ID             string `json:"id"`
		Email          string `json:"email"`
		Name           string `json:"name"`
		Role           string `json:"role"`
		OrganizationID string `json:"organization_id"`
	}

	_ = json.NewDecoder(resp.Body).Decode(&userResp)

	fmt.Println("✓ Authenticated Identity:")
	if userResp.Name != "" {
		fmt.Printf("  User:           %s (%s)\n", userResp.Name, userResp.Email)
	} else if userResp.Email != "" {
		fmt.Printf("  Email:          %s\n", userResp.Email)
	} else if cfg.Email != "" {
		fmt.Printf("  Email:          %s\n", cfg.Email)
	}

	if userResp.ID != "" {
		fmt.Printf("  User ID:        %s\n", userResp.ID)
	}
	if userResp.Role != "" {
		fmt.Printf("  Role:           %s\n", userResp.Role)
	}

	orgID := userResp.OrganizationID
	if orgID == "" {
		orgID = cfg.OrgID
	}
	if orgID != "" {
		fmt.Printf("  Organization:   %s\n", orgID)
	}
	if cfg.ProjectID != "" {
		fmt.Printf("  Project:        %s\n", cfg.ProjectID)
	}
	if cfg.Environment != "" {
		fmt.Printf("  Environment:    %s\n", cfg.Environment)
	}

	fmt.Printf("  Cloud URL:      %s\n", cfg.CloudURL)
	fmt.Printf("  Active Token:   %s\n", maskSecret(token))
	return nil
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		return "********"
	}
	return s[:4] + "..." + s[len(s)-4:]
}
