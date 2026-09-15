package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type PolicyListOptions struct {
	CloudURL string
	Token    string
	OrgID    string
	Scope    string
	Status   string
}

type PolicyGetOptions struct {
	PolicyID string
	CloudURL string
	Token    string
	OrgID    string
}

type PolicyCreateOptions struct {
	CloudURL      string
	Token         string
	OrgID         string
	Name          string
	Description   string
	Scope         string
	Priority      int
	Effect        string
	Models        string // comma-separated
	Providers     string // comma-separated
	Plugins       string // comma-separated
	Capabilities  string // comma-separated
	Tools         string // comma-separated
	Environments  string // comma-separated
	Principals    string // comma-separated
	MaxSteps      int
	MaxToolCalls  int
}

type PolicyUpdateOptions struct {
	PolicyID    string
	CloudURL    string
	Token       string
	OrgID       string
	Name        string
	Description string
	Priority    int
	Effect      string
}

type PolicyActionOptions struct {
	PolicyID      string
	CloudURL      string
	Token         string
	OrgID         string
	TargetVersion int
}

type PolicySimulateOptions struct {
	CloudURL      string
	Token         string
	OrgID         string
	EnvironmentID string
	PrincipalID   string
	Model         string
	Provider      string
	Plugin        string
	Capability    string
	Tool          string
	RequestType   string
}

func resolveCloudAndToken(cloudURL, token, orgID string) (string, string, string, error) {
	if cloudURL == "" {
		cloudURL = os.Getenv("NAAGMANI_CLOUD_URL")
	}
	if token == "" {
		token = os.Getenv("NAAGMANI_CLOUD_TOKEN")
	}
	if orgID == "" {
		orgID = os.Getenv("NAAGMANI_ORG_ID")
	}

	if cloudURL == "" || token == "" {
		cfg, _ := LoadCLIConfig()
		if cfg != nil {
			if cloudURL == "" && cfg.CloudURL != "" {
				cloudURL = cfg.CloudURL
			}
			if token == "" && cfg.Token != "" {
				token = cfg.Token
			}
		}
	}

	if cloudURL == "" {
		cloudURL = "http://localhost:8081"
	}
	cloudURL = strings.TrimRight(cloudURL, "/")

	if orgID == "" {
		orgID = "default"
	}

	return cloudURL, token, orgID, nil
}

func RunPolicyList(opts PolicyListOptions) error {
	cloudURL, token, orgID, err := resolveCloudAndToken(opts.CloudURL, opts.Token, opts.OrgID)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/v1/orgs/%s/policies", cloudURL, orgID)
	var query []string
	if opts.Scope != "" {
		query = append(query, "scope="+opts.Scope)
	}
	if opts.Status != "" {
		query = append(query, "status="+opts.Status)
	}
	if len(query) > 0 {
		endpoint += "?" + strings.Join(query, "&")
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed calling cloud: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloud returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Policies []map[string]interface{} `json:"policies"`
		Total    int                      `json:"total"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println(string(body))
		return nil
	}

	fmt.Printf("Policies (Total: %d):\n", result.Total)
	fmt.Printf("%-12s %-25s %-10s %-8s %-8s %-8s %s\n", "ID", "NAME", "SCOPE", "PRIORITY", "EFFECT", "STATUS", "VERSION")
	fmt.Println(strings.Repeat("-", 85))
	for _, p := range result.Policies {
		id, _ := p["id"].(string)
		name, _ := p["name"].(string)
		scope, _ := p["scope"].(string)
		priority := fmt.Sprintf("%v", p["priority"])
		effect, _ := p["effect"].(string)
		status, _ := p["status"].(string)
		version := fmt.Sprintf("v%v", p["version"])
		fmt.Printf("%-12s %-25s %-10s %-8s %-8s %-8s %s\n", id, name, scope, priority, effect, status, version)
	}
	return nil
}

func RunPolicyGet(opts PolicyGetOptions) error {
	if opts.PolicyID == "" {
		return fmt.Errorf("policy id is required")
	}
	cloudURL, token, orgID, err := resolveCloudAndToken(opts.CloudURL, opts.Token, opts.OrgID)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/v1/orgs/%s/policies/%s", cloudURL, orgID, opts.PolicyID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed calling cloud: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloud returned status %d: %s", resp.StatusCode, string(body))
	}

	var pretty bytes.Buffer
	_ = json.Indent(&pretty, body, "", "  ")
	fmt.Println(pretty.String())
	return nil
}

func RunPolicyCreate(opts PolicyCreateOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	if opts.Effect == "" {
		opts.Effect = "ALLOW"
	}
	if opts.Scope == "" {
		opts.Scope = "organization"
	}

	cloudURL, token, orgID, err := resolveCloudAndToken(opts.CloudURL, opts.Token, opts.OrgID)
	if err != nil {
		return err
	}

	conditions := map[string][]string{}
	if opts.Models != "" {
		conditions["models"] = splitAndTrim(opts.Models)
	}
	if opts.Providers != "" {
		conditions["providers"] = splitAndTrim(opts.Providers)
	}
	if opts.Plugins != "" {
		conditions["plugins"] = splitAndTrim(opts.Plugins)
	}
	if opts.Capabilities != "" {
		conditions["capabilities"] = splitAndTrim(opts.Capabilities)
	}
	if opts.Tools != "" {
		conditions["tools"] = splitAndTrim(opts.Tools)
	}
	if opts.Environments != "" {
		conditions["environments"] = splitAndTrim(opts.Environments)
	}
	if opts.Principals != "" {
		conditions["principals"] = splitAndTrim(opts.Principals)
	}

	limits := map[string]int{}
	if opts.MaxSteps > 0 {
		limits["max_steps"] = opts.MaxSteps
	}
	if opts.MaxToolCalls > 0 {
		limits["max_tool_calls"] = opts.MaxToolCalls
	}

	payload := map[string]interface{}{
		"name":        opts.Name,
		"description": opts.Description,
		"scope":       opts.Scope,
		"priority":    opts.Priority,
		"effect":      strings.ToUpper(opts.Effect),
		"conditions":  conditions,
	}
	if len(limits) > 0 {
		payload["limits"] = limits
	}

	b, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("%s/v1/orgs/%s/policies", cloudURL, orgID)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed calling cloud: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloud returned status %d: %s", resp.StatusCode, string(body))
	}

	var created map[string]interface{}
	_ = json.Unmarshal(body, &created)
	fmt.Printf("Policy created successfully: ID=%v Name=%q Scope=%v Effect=%v Version=%v\n",
		created["id"], created["name"], created["scope"], created["effect"], created["version"])
	return nil
}

func RunPolicyUpdate(opts PolicyUpdateOptions) error {
	if opts.PolicyID == "" {
		return fmt.Errorf("policy id is required")
	}
	cloudURL, token, orgID, err := resolveCloudAndToken(opts.CloudURL, opts.Token, opts.OrgID)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{}
	if opts.Name != "" {
		payload["name"] = opts.Name
	}
	if opts.Description != "" {
		payload["description"] = opts.Description
	}
	if opts.Priority != 0 {
		payload["priority"] = opts.Priority
	}
	if opts.Effect != "" {
		payload["effect"] = strings.ToUpper(opts.Effect)
	}

	b, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("%s/v1/orgs/%s/policies/%s", cloudURL, orgID, opts.PolicyID)
	req, err := http.NewRequest("PATCH", endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed calling cloud: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloud returned status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Policy %s updated successfully.\n", opts.PolicyID)
	return nil
}

func RunPolicyEnable(opts PolicyActionOptions) error {
	if opts.PolicyID == "" {
		return fmt.Errorf("policy id is required")
	}
	cloudURL, token, orgID, err := resolveCloudAndToken(opts.CloudURL, opts.Token, opts.OrgID)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/v1/orgs/%s/policies/%s/enable", cloudURL, orgID, opts.PolicyID)
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed calling cloud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloud returned status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Policy %s enabled successfully.\n", opts.PolicyID)
	return nil
}

func RunPolicyDisable(opts PolicyActionOptions) error {
	if opts.PolicyID == "" {
		return fmt.Errorf("policy id is required")
	}
	cloudURL, token, orgID, err := resolveCloudAndToken(opts.CloudURL, opts.Token, opts.OrgID)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/v1/orgs/%s/policies/%s/disable", cloudURL, orgID, opts.PolicyID)
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed calling cloud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloud returned status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Policy %s disabled successfully.\n", opts.PolicyID)
	return nil
}

func RunPolicyRollback(opts PolicyActionOptions) error {
	if opts.PolicyID == "" {
		return fmt.Errorf("policy id is required")
	}
	cloudURL, token, orgID, err := resolveCloudAndToken(opts.CloudURL, opts.Token, opts.OrgID)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"target_version": opts.TargetVersion,
	}
	b, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("%s/v1/orgs/%s/policies/%s/rollback", cloudURL, orgID, opts.PolicyID)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed calling cloud: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloud returned status %d: %s", resp.StatusCode, string(body))
	}

	var res map[string]interface{}
	_ = json.Unmarshal(body, &res)
	fmt.Printf("Policy %s rolled back successfully to version %v (now version %v).\n",
		opts.PolicyID, opts.TargetVersion, res["version"])
	return nil
}

func RunPolicySimulate(opts PolicySimulateOptions) error {
	cloudURL, token, orgID, err := resolveCloudAndToken(opts.CloudURL, opts.Token, opts.OrgID)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"org_id":         orgID,
		"environment_id": opts.EnvironmentID,
		"principal_id":   opts.PrincipalID,
		"model":          opts.Model,
		"provider":       opts.Provider,
		"plugin":         opts.Plugin,
		"capability":     opts.Capability,
		"tool":           opts.Tool,
		"request_type":   opts.RequestType,
	}

	b, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("%s/v1/policies/simulate", cloudURL)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed calling cloud: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloud returned status %d: %s", resp.StatusCode, string(body))
	}

	var dec struct {
		Effect            string   `json:"effect"`
		PolicyID          string   `json:"policy_id"`
		PolicyName        string   `json:"policy_name"`
		Reason            string   `json:"reason"`
		MatchedConditions []string `json:"matched_conditions"`
		Allowed           bool     `json:"allowed"`
	}
	_ = json.Unmarshal(body, &dec)

	fmt.Println("=== Policy Simulation Result ===")
	fmt.Printf("Decision: %s (Allowed: %v)\n", dec.Effect, dec.Allowed)
	if dec.PolicyID != "" {
		fmt.Printf("Matched Policy: %s (%s)\n", dec.PolicyName, dec.PolicyID)
	}
	fmt.Printf("Reason: %s\n", dec.Reason)
	if len(dec.MatchedConditions) > 0 {
		fmt.Printf("Matched Conditions: %s\n", strings.Join(dec.MatchedConditions, ", "))
	}
	return nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	var res []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			res = append(res, t)
		}
	}
	return res
}
