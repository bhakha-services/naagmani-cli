package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLicenseHeadlessRequestImportAndStatus(t *testing.T) {
	t.Setenv("NAAGMANI_OS_ADMIN_TOKEN", "test-local-system-token")
	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-local-system-token" {
			t.Error("missing credential")
		}
		calls[r.URL.Path]++
		switch r.URL.Path {
		case "/v1/installation/activation-request":
			b, _ := io.ReadAll(r.Body)
			if string(b) != `{"owner_email":"owner@example.com"}` {
				t.Errorf("unexpected request: %s", b)
			}
			_, _ = w.Write([]byte(`{"installation_id":"inst-headless"}`))
		case "/v1/installation/license":
			b, _ := io.ReadAll(r.Body)
			if string(b) != "NAAGMANI-LICENSE-1\ntest-container" {
				t.Error("license changed")
			}
			_, _ = w.Write([]byte(`{"active":true}`))
		case "/v1/entitlements":
			_, _ = w.Write([]byte(`{"active":true}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	out := filepath.Join(dir, "activation.nareq")
	if err := RunLicense([]string{"request", "--email", "owner@example.com", "--out", out, "--os-url", server.URL}); err != nil {
		t.Fatal(err)
	}
	if err := RunLicense([]string{"request", "--email", "owner@example.com", "--out", out, "--os-url", server.URL}); err == nil {
		t.Fatal("overwrote existing request")
	}
	lic := filepath.Join(dir, "license.lic")
	if err := os.WriteFile(lic, []byte("NAAGMANI-LICENSE-1\ntest-container"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RunLicense([]string{"import", "--file", lic, "--os-url", server.URL}); err != nil {
		t.Fatal(err)
	}
	if err := RunLicense([]string{"status", "--os-url", server.URL}); err != nil {
		t.Fatal(err)
	}
	if calls["/v1/installation/license"] != 1 || calls["/v1/entitlements"] != 1 {
		t.Fatal(calls)
	}
	if err := os.WriteFile(lic, []byte(`{"payload":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RunLicense([]string{"import", "--file", lic, "--os-url", server.URL}); err == nil {
		t.Fatal("accepted raw JSON")
	}
	if err := RunLicense([]string{"status", "--os-url", "http://example.com"}); err == nil {
		t.Fatal("allowed insecure remote credential transmission")
	}
}

func TestLicenseRefusesRedirect(t *testing.T) {
	t.Setenv("NAAGMANI_OS_ADMIN_TOKEN", "test-local-system-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "https://example.com", 302) }))
	defer server.Close()
	if err := RunLicense([]string{"status", "--os-url", server.URL}); err == nil {
		t.Fatal("followed credential redirect")
	}
}
