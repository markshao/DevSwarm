package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestNewQwenProvider tests the QwenProvider constructor
func TestNewQwenProvider(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	p := NewQwenProvider(cfg)
	if p == nil {
		t.Fatal("NewQwenProvider() returned nil")
	}

	if p.Name() != "qwen" {
		t.Errorf("Name() = %q, want %q", p.Name(), "qwen")
	}
}

// TestQwenProviderRun tests the QwenProvider.Run method
func TestQwenProviderRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	p := NewQwenProvider(cfg)
	ctx := context.Background()
	prompt := "Test prompt content"

	output, err := p.Run(ctx, prompt, tmpDir, []string{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if output == "" {
		t.Errorf("Run() returned empty output")
	}

	// Verify the agent_output.txt file was created
	outputFile := filepath.Join(tmpDir, "agent_output.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("agent_output.txt was not created")
	} else {
		content, _ := os.ReadFile(outputFile)
		if len(content) == 0 {
			t.Errorf("agent_output.txt is empty")
		}
	}
}

// TestNewProvider tests the provider factory function
func TestNewProvider(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		wantErr   bool
		errContains string
	}{
		{
			name:     "qwen provider",
			provider: "qwen",
			wantErr:  false,
		},
		{
			name:      "trae provider not implemented",
			provider:  "trae",
			wantErr:   true,
			errContains: "not yet implemented",
		},
		{
			name:      "unknown provider",
			provider:  "unknown",
			wantErr:   true,
			errContains: "unknown provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Provider: tt.provider,
				Model:    "test-model",
			}

			p, err := NewProvider(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errContains != "" && err != nil {
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("NewProvider() error = %q, want to contain %q", err.Error(), tt.errContains)
					}
				}
			} else if p == nil {
				t.Errorf("NewProvider() returned nil provider")
			}
		})
	}
}

// TestRenderPrompt tests the RenderPrompt function
func TestRenderPrompt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-prompt-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create base template directory
	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	tests := []struct {
		name             string
		baseTemplate     string
		agentPrompt      string
		ctx              PromptContext
		wantContains     string
		wantErr          bool
		customBaseContent string
	}{
		{
			name:         "simple prompt with context",
			baseTemplate: "default",
			agentPrompt:  "Write unit tests for the code",
			ctx: PromptContext{
				Env:        []string{"TEST=true"},
				BaseBranch: "main",
			},
			wantContains: "Write unit tests for the code",
			wantErr:      false,
		},
		{
			name:         "missing base template uses fallback",
			baseTemplate: "nonexistent",
			agentPrompt:  "Simple task",
			ctx:          PromptContext{},
			wantContains: "Simple task",
			wantErr:      false,
		},
		{
			name:         "template with variables",
			baseTemplate: "default",
			agentPrompt:  "Review changes from {{.BaseBranch}}",
			ctx: PromptContext{
				BaseBranch: "develop",
			},
			wantContains: "Review changes from develop",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write base template if custom content provided
			if tt.customBaseContent != "" {
				basePath := filepath.Join(promptsDir, tt.baseTemplate+".tmpl")
				if err := os.WriteFile(basePath, []byte(tt.customBaseContent), 0644); err != nil {
					t.Fatalf("failed to write base template: %v", err)
				}
			}

			got, err := RenderPrompt(tmpDir, tt.baseTemplate, tt.agentPrompt, tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.wantContains != "" && !contains(got, tt.wantContains) {
					t.Errorf("RenderPrompt() = %q, want to contain %q", got, tt.wantContains)
				}
			}
		})
	}
}

// TestRenderPromptWithInvalidTemplate tests error handling for invalid templates
func TestRenderPromptWithInvalidTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-prompt-invalid-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create base template directory
	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	// Write invalid template syntax
	basePath := filepath.Join(promptsDir, "invalid.tmpl")
	if err := os.WriteFile(basePath, []byte("{{.Invalid"), 0644); err != nil {
		t.Fatalf("failed to write invalid template: %v", err)
	}

	_, err = RenderPrompt(tmpDir, "invalid", "test prompt", PromptContext{})
	if err == nil {
		t.Errorf("RenderPrompt() with invalid template should return error")
	}
}

// Helper function to check if a string contains a substring
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
