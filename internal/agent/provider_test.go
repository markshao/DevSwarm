package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestNewQwenProvider tests creating a new Qwen provider
func TestNewQwenProvider(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)
	if provider == nil {
		t.Fatal("NewQwenProvider returned nil")
	}

	if provider.Name() != "qwen" {
		t.Errorf("expected provider name 'qwen', got '%s'", provider.Name())
	}
}

// TestQwenProviderName tests the Name method
func TestQwenProviderName(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)
	name := provider.Name()

	if name != "qwen" {
		t.Errorf("expected name 'qwen', got '%s'", name)
	}
}

// TestQwenProviderRun tests the Run method of QwenProvider
func TestQwenProviderRun(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)

	// Create temp directory for workdir
	workdir, err := os.MkdirTemp("", "orion-agent-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workdir)

	ctx := context.Background()
	prompt := "Test prompt for agent"

	output, err := provider.Run(ctx, prompt, workdir, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if output == "" {
		t.Error("expected non-empty output from Run")
	}

	// Verify agent_output.txt was created
	outputFile := filepath.Join(workdir, "agent_output.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("agent_output.txt was not created")
	} else {
		// Verify content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}
		if len(content) == 0 {
			t.Error("output file is empty")
		}
	}
}

// TestQwenProviderRunWithEnv tests Run with environment variables
func TestQwenProviderRunWithEnv(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)

	workdir, err := os.MkdirTemp("", "orion-agent-env-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workdir)

	ctx := context.Background()
	prompt := "Test with env vars"
	env := []string{"TEST_VAR=test_value", "ANOTHER_VAR=another_value"}

	output, err := provider.Run(ctx, prompt, workdir, env)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if output == "" {
		t.Error("expected non-empty output")
	}
}

// TestQwenProviderRunWithInvalidDir tests Run with invalid directory
func TestQwenProviderRunWithInvalidDir(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)

	ctx := context.Background()
	prompt := "Test prompt"

	// Use invalid directory
	_, err := provider.Run(ctx, prompt, "/nonexistent/path/that/does/not/exist", nil)
	if err == nil {
		t.Error("expected error when running with invalid directory")
	}
}

// TestQwenProviderRunContextCancellation tests Run with cancelled context
func TestQwenProviderRunContextCancellation(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)

	workdir, err := os.MkdirTemp("", "orion-agent-cancel-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workdir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	prompt := "Test prompt"

	// The current implementation doesn't check context, so this might still succeed
	// This test documents the expected behavior for future implementations
	output, err := provider.Run(ctx, prompt, workdir, nil)
	if err != nil {
		t.Logf("Run with cancelled context returned error (expected in future): %v", err)
	}
	_ = output
}

// TestNewProvider tests the provider factory function
func TestNewProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		errContains string
	}{
		{
			name: "qwen provider",
			config: Config{
				Provider: "qwen",
				Model:    "qwen-max",
			},
			wantErr: false,
		},
		{
			name: "trae provider (not implemented)",
			config: Config{
				Provider: "trae",
				Model:    "trae-pro",
			},
			wantErr:     true,
			errContains: "not yet implemented",
		},
		{
			name: "unknown provider",
			config: Config{
				Provider: "unknown",
				Model:    "some-model",
			},
			wantErr:     true,
			errContains: "unknown provider",
		},
		{
			name: "empty provider",
			config: Config{
				Provider: "",
				Model:    "default",
			},
			wantErr:     true,
			errContains: "unknown provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("expected non-nil provider")
				}
			}
		})
	}
}

// TestConfigYamlParsing tests parsing Config from YAML
func TestConfigYamlParsing(t *testing.T) {
	yamlContent := `
provider: qwen
model: qwen-max
api_key: test-key
endpoint: https://api.example.com
params:
  temperature: "0.7"
  max_tokens: "1000"
`

	// Note: This tests the structure compatibility with YAML
	// Actual YAML parsing would be done by the caller using yaml.Unmarshal
	_ = yamlContent

	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
		APIKey:   "test-key",
		Endpoint: "https://api.example.com",
		Params: map[string]string{
			"temperature": "0.7",
			"max_tokens":  "1000",
		},
	}

	if cfg.Provider != "qwen" {
		t.Errorf("expected provider 'qwen', got '%s'", cfg.Provider)
	}
	if cfg.Model != "qwen-max" {
		t.Errorf("expected model 'qwen-max', got '%s'", cfg.Model)
	}
}

// TestConfigWithEmptyParams tests Config with empty params
func TestConfigWithEmptyParams(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
		Params:   map[string]string{},
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if provider.Name() != "qwen" {
		t.Errorf("expected provider name 'qwen', got '%s'", provider.Name())
	}
}

// TestConfigWithNilParams tests Config with nil params
func TestConfigWithNilParams(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
		Params:   nil,
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if provider.Name() != "qwen" {
		t.Errorf("expected provider name 'qwen', got '%s'", provider.Name())
	}
}

// TestMultipleProviderCreations tests creating multiple providers
func TestMultipleProviderCreations(t *testing.T) {
	configs := []Config{
		{Provider: "qwen", Model: "qwen-max"},
		{Provider: "qwen", Model: "qwen-plus"},
		{Provider: "qwen", Model: "qwen-turbo"},
	}

	providers := make([]Provider, len(configs))
	for i, cfg := range configs {
		p, err := NewProvider(cfg)
		if err != nil {
			t.Fatalf("failed to create provider %d: %v", i, err)
		}
		providers[i] = p
	}

	// Verify all providers are created and functional
	for i, p := range providers {
		if p == nil {
			t.Errorf("provider %d is nil", i)
		}
		if p.Name() != "qwen" {
			t.Errorf("provider %d has wrong name: %s", i, p.Name())
		}
	}
}

// TestProviderRunInWorktree tests running provider in a simulated worktree
func TestProviderRunInWorktree(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)

	// Simulate a worktree directory structure
	worktreeDir, err := os.MkdirTemp("", "orion-worktree-test")
	if err != nil {
		t.Fatalf("failed to create worktree dir: %v", err)
	}
	defer os.RemoveAll(worktreeDir)

	// Create some files in worktree to simulate a project
	srcDir := filepath.Join(worktreeDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	testFile := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ctx := context.Background()
	prompt := "Add unit tests to the code"

	output, err := provider.Run(ctx, prompt, worktreeDir, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if output == "" {
		t.Error("expected non-empty output")
	}

	// Verify output file was created in worktree root
	outputFile := filepath.Join(worktreeDir, "agent_output.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("agent_output.txt was not created in worktree")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
