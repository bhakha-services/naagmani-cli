package cmd

import (
	"strings"
	"testing"
)

func TestPlatformComposeForwardsOptionalSMTPOnlyToLocalAPI(t *testing.T) {
	compose := getProductionComposeTemplate()
	cloud := strings.Split(strings.Split(compose, "  naagmani-cloud:")[1], "  naagmani-developer:")[0]
	portal := strings.Split(compose, "  naagmani-developer:")[1]
	for _, key := range []string{"EMAIL_ENABLED", "EMAIL_FROM", "SMTP_HOST", "SMTP_PORT", "SMTP_SECURITY", "SMTP_USERNAME", "SMTP_PASSWORD"} {
		if !strings.Contains(cloud, key+":") {
			t.Fatalf("missing %s", key)
		}
		if strings.Contains(portal, key+":") {
			t.Fatalf("SMTP configuration leaked to portal: %s", key)
		}
	}
	if !strings.Contains(cloud, "EMAIL_ENABLED: ${EMAIL_ENABLED:-false}") {
		t.Fatal("email must remain optional")
	}
}
