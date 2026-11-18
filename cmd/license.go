package cmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// RunLicense operates solely against the local OS, including air-gapped hosts.
func RunLicense(args []string) error {
	if len(args) == 0 || args[0] == "help" {
		fmt.Println("Usage: naagmani license request --email owner@example.com --out activation.nareq | import --file license.lic | status\nOptions: --os-url http://localhost:8080; local control-plane credential in NAAGMANI_OS_ADMIN_TOKEN")
		return nil
	}
	fs := flag.NewFlagSet("license "+args[0], flag.ContinueOnError)
	endpoint := fs.String("os-url", "http://localhost:8080", "Local OS URL")
	email := fs.String("email", "", "Configured primary admin email (request only)")
	output := fs.String("out", "activation.nareq", "Request output path")
	file := fs.String("file", "", "Encrypted .lic file")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	token := os.Getenv("NAAGMANI_OS_ADMIN_TOKEN")
	if token == "" {
		return fmt.Errorf("set NAAGMANI_OS_ADMIN_TOKEN to a local control-plane credential")
	}
	u, err := url.Parse(*endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"))) {
		return fmt.Errorf("use HTTPS or loopback HTTP for local licensing")
	}
	var body []byte
	method := http.MethodPost
	path := ""
	switch args[0] {
	case "request":
		if *email == "" {
			return fmt.Errorf("--email is required")
		}
		body, _ = json.Marshal(map[string]string{"owner_email": *email})
		path = "/v1/installation/activation-request"
	case "import":
		if !strings.HasSuffix(strings.ToLower(*file), ".lic") {
			return fmt.Errorf("--file must select an encrypted .lic")
		}
		body, err = os.ReadFile(*file)
		if err != nil {
			return err
		}
		if len(body) > 1<<20 || !bytes.HasPrefix(body, []byte("NAAGMANI-LICENSE-1\n")) {
			return fmt.Errorf("invalid encrypted .lic file")
		}
		path = "/v1/installation/license"
	case "status":
		method = http.MethodGet
		path = "/v1/entitlements"
	default:
		return fmt.Errorf("unknown license command %q", args[0])
	}
	req, err := http.NewRequest(method, strings.TrimRight(*endpoint, "/")+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("licensing redirects are not allowed") }}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil {
		return err
	}
	if len(data) > 1<<20 {
		return fmt.Errorf("response too large")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("local licensing returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if args[0] == "request" {
		f, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		ce := f.Close()
		if err == nil {
			err = ce
		}
		if err != nil {
			return err
		}
		fmt.Printf("Activation request saved to %s. Upload it to Accounts using the same verified primary admin email.\n", *output)
	} else {
		fmt.Println(string(data))
	}
	return nil
}
