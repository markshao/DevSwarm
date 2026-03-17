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
		t.Errorf("Expected provider name 'qwen', got '%s'", provider.Name())
	}
}

// TestQwenProviderName tests the Name method
func TestQwenProviderName(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-turbo",
	}

	provider := NewQwenProvider(cfg)
	name := provider.Name()

	if name != "qwen" {
		t.Errorf("Expected name 'qwen', got '%s'", name)
	}
}

// TestQwenProviderRun tests the Run method
func TestQwenProviderRun(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "orion-agent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)
	ctx := context.Background()
	prompt := "Test prompt for agent"

	output, err := provider.Run(ctx, prompt, tmpDir, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if output == "" {
		t.Error("Expected non-empty output from Run")
	}

	// Verify agent_output.txt was created
	outputFile := filepath.Join(tmpDir, "agent_output.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("agent_output.txt was not created")
	} else {
		// Verify content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		} else {
			expectedContent := "Agent executed with prompt:\n" + prompt
			if string(content) != expectedContent {
				t.Errorf("Output content mismatch.\nExpected: %s\nGot: %s", expectedContent, string(content))
			}
		}
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

	// Use non-existent directory
	invalidDir := "/non/existent/path/that/does/not/exist"
	_, err := provider.Run(ctx, "test prompt", invalidDir, nil)

	if err == nil {
		t.Error("Run should fail with invalid directory")
	}
}

// TestQwenProviderRunWithEnv tests Run with environment variables
func TestQwenProviderRunWithEnv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-env-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)
	ctx := context.Background()

	// Run with custom env
	env := []string{"TEST_VAR=test_value"}
	_, err = provider.Run(ctx, "test prompt", tmpDir, env)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
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
				Model:    "model",
			},
			wantErr:     true,
			errContains: "unknown provider",
		},
		{
			name: "empty provider",
			config: Config{
				Provider: "",
				Model:    "model",
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
					t.Error("Expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errContains, err.Error())
				}
				if provider != nil {
					t.Error("Expected nil provider on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("Expected non-nil provider")
				}
			}
		})
	}
}

// TestQwenProviderRunContextCancellation tests context cancellation
func TestQwenProviderRunContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-cancel-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Run should still work as it doesn't actually use context for cancellation in mock
	_, err = provider.Run(ctx, "test prompt", tmpDir, nil)
	if err != nil {
		t.Errorf("Run failed with cancelled context: %v", err)
	}
}

// TestQwenProviderRunWithSpecialChars tests Run with special characters in prompt
func TestQwenProviderRunWithSpecialChars(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-special-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider := NewQwenProvider(cfg)
	ctx := context.Background()

	// Prompt with special characters
	specialPrompt := "Test prompt with special chars: \n\t\"'&<>"
	_, err = provider.Run(ctx, specialPrompt, tmpDir, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify content
	outputFile := filepath.Join(tmpDir, "agent_output.txt")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Errorf("Failed to read output file: %v", err)
	} else {
		expectedContent := "Agent executed with prompt:\n" + specialPrompt
		if string(content) != expectedContent {
			t.Errorf("Output content mismatch for special chars prompt")
		}
	}
}

// TestConfigStruct tests the Config struct
func TestConfigStruct(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
		APIKey:   "test-api-key",
		Endpoint: "https://api.example.com",
		Params: map[string]string{
			"temperature": "0.7",
			"max_tokens":  "1000",
		},
	}

	if cfg.Provider != "qwen" {
		t.Errorf("Provider mismatch: %s", cfg.Provider)
	}
	if cfg.Model != "qwen-max" {
		t.Errorf("Model mismatch: %s", cfg.Model)
	}
	if cfg.APIKey != "test-api-key" {
		t.Errorf("APIKey mismatch: %s", cfg.APIKey)
	}
	if cfg.Endpoint != "https://api.example.com" {
		t.Errorf("Endpoint mismatch: %s", cfg.Endpoint)
	}
	if len(cfg.Params) != 2 {
		t.Errorf("Params length mismatch: %d", len(cfg.Params))
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
