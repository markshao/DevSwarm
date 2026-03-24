package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromOrionYAML(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("MOONSHOT_API_KEY", "k-test")

	content := `llm:
  api_key: "${MOONSHOT_API_KEY}"
  base_url: "https://api.moonshot.cn/v1"
  model: "kimi-k2-turbo-preview"
`
	if err := os.WriteFile(filepath.Join(homeDir, ".orion.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.APIKey != "k-test" {
		t.Fatalf("expected expanded api key, got %q", cfg.APIKey)
	}
	if cfg.BaseURL != "https://api.moonshot.cn/v1" {
		t.Fatalf("unexpected base_url: %q", cfg.BaseURL)
	}
	if cfg.Model != "kimi-k2-turbo-preview" {
		t.Fatalf("unexpected model: %q", cfg.Model)
	}
}
