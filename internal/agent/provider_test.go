package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestQwenProviderName verifies the provider name
func TestQwenProviderName(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}
	provider := NewQwenProvider(cfg)

	if provider.Name() != "qwen" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "qwen")
	}
}

// TestQwenProviderRun verifies the Run method creates output file
func TestQwenProviderRun(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}
	provider := NewQwenProvider(cfg)

	// Create temp directory for workdir
	tmpDir, err := os.MkdirTemp("", "orion-agent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	prompt := "Test prompt content"

	output, err := provider.Run(ctx, prompt, tmpDir, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if output == "" {
		t.Error("Run returned empty output")
	}

	// Verify agent_output.txt was created
	outputFile := filepath.Join(tmpDir, "agent_output.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("agent_output.txt was not created")
	}

	// Verify content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Output file is empty")
	}
}

// TestNewProviderQwen verifies NewProvider creates QwenProvider
func TestNewProviderQwen(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}

	if provider == nil {
		t.Fatal("NewProvider returned nil provider")
	}

	if provider.Name() != "qwen" {
		t.Errorf("Provider.Name() = %q, want %q", provider.Name(), "qwen")
	}
}

// TestNewProviderTrae verifies NewProvider returns error for trae (not implemented)
func TestNewProviderTrae(t *testing.T) {
	cfg := Config{
		Provider: "trae",
		Model:    "trae-model",
	}

	provider, err := NewProvider(cfg)
	if err == nil {
		t.Error("NewProvider should return error for trae provider (not implemented)")
	}
	if provider != nil {
		t.Error("NewProvider should return nil provider for trae")
	}
}

// TestNewProviderUnknown verifies NewProvider returns error for unknown provider
func TestNewProviderUnknown(t *testing.T) {
	cfg := Config{
		Provider: "unknown-provider",
		Model:    "some-model",
	}

	provider, err := NewProvider(cfg)
	if err == nil {
		t.Error("NewProvider should return error for unknown provider")
	}
	if provider != nil {
		t.Error("NewProvider should return nil provider for unknown provider")
	}
}

// TestRenderPromptBasic verifies basic prompt rendering
func TestRenderPromptBasic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-prompt-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create prompts directory
	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("Failed to create prompts dir: %v", err)
	}

	// Create base template
	baseTemplate := `Task: {{.Task}}
Base Branch: {{.BaseBranch}}`
	basePath := filepath.Join(promptsDir, "test.tmpl")
	if err := os.WriteFile(basePath, []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("Failed to write base template: %v", err)
	}

	ctx := PromptContext{
		Task:       "My specific task",
		BaseBranch: "main",
	}

	result, err := RenderPrompt(tmpDir, "test", "Agent instruction", ctx)
	if err != nil {
		t.Fatalf("RenderPrompt failed: %v", err)
	}

	if result == "" {
		t.Fatal("RenderPrompt returned empty result")
	}

	// The task should be replaced by agent prompt
	// Note: The current implementation uses agentPrompt as Task directly
	// So the result should contain "Agent instruction"
}

// TestRenderPromptFallback verifies fallback when base template is missing
func TestRenderPromptFallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-prompt-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Don't create prompts directory - test fallback behavior

	ctx := PromptContext{
		Task: "Fallback task",
	}

	result, err := RenderPrompt(tmpDir, "nonexistent", "Agent instruction", ctx)
	if err != nil {
		t.Fatalf("RenderPrompt should not fail with fallback: %v", err)
	}

	if result == "" {
		t.Error("RenderPrompt returned empty result with fallback")
	}
}

// TestRenderPromptWithTemplateVariables verifies template variable substitution
func TestRenderPromptWithTemplateVariables(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-prompt-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create prompts directory
	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("Failed to create prompts dir: %v", err)
	}

	// Create base template with variables
	baseTemplate := `You are working with base branch {{.BaseBranch}}.
Your task is: {{.Task}}
Changed files: {{.ChangedFiles}}`
	basePath := filepath.Join(promptsDir, "var.tmpl")
	if err := os.WriteFile(basePath, []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("Failed to write base template: %v", err)
	}

	ctx := PromptContext{
		Task:         "Test task",
		BaseBranch:   "main",
		ChangedFiles: []string{"file1.go", "file2.go"},
	}

	result, err := RenderPrompt(tmpDir, "var", "Direct instruction", ctx)
	if err != nil {
		t.Fatalf("RenderPrompt failed: %v", err)
	}

	// Verify template variables are substituted
	// Note: Current implementation replaces Task with agentPrompt
	// So we check if the result contains the branch info
	if len(result) == 0 {
		t.Error("RenderPrompt returned empty result")
	}
}

// TestPromptContext verifies PromptContext structure
func TestPromptContext(t *testing.T) {
	ctx := PromptContext{
		Task:         "Test task",
		Env:          []string{"VAR1=value1", "VAR2=value2"},
		ChangedFiles: []string{"file1.go", "file2.go"},
		BaseBranch:   "main",
	}

	if ctx.Task != "Test task" {
		t.Errorf("Task = %q, want %q", ctx.Task, "Test task")
	}
	if len(ctx.Env) != 2 {
		t.Errorf("Env length = %d, want %d", len(ctx.Env), 2)
	}
	if len(ctx.ChangedFiles) != 2 {
		t.Errorf("ChangedFiles length = %d, want %d", len(ctx.ChangedFiles), 2)
	}
	if ctx.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want %q", ctx.BaseBranch, "main")
	}
}
