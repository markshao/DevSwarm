package notification

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestBuildLarkCardPayloadIncludesNodeAndTitle(t *testing.T) {
	payload := buildLarkCardPayload("boss, 我想干活", "node-123", "review", "approval required")

	header, ok := payload["header"].(map[string]interface{})
	if !ok {
		t.Fatalf("header missing in payload")
	}
	title, ok := header["title"].(map[string]interface{})
	if !ok {
		t.Fatalf("title missing in payload header")
	}
	if got := title["content"]; got != "boss, 我想干活" {
		t.Fatalf("expected card title %q, got %v", "boss, 我想干活", got)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "Node:** node-123") {
		t.Fatalf("expected node info in card body, got %s", body)
	}
	if !strings.Contains(body, "Reason:** approval required") {
		t.Fatalf("expected reason in card body, got %s", body)
	}
}

func TestResolveEnvReference(t *testing.T) {
	if err := os.Setenv("ORION_LARK_TEST_VALUE", "token-123"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("ORION_LARK_TEST_VALUE") })

	got := resolveEnvReference("${ORION_LARK_TEST_VALUE}")
	if got != "token-123" {
		t.Fatalf("expected resolved env value, got %q", got)
	}
}
