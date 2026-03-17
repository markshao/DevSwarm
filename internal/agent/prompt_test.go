package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderPrompt 测试基本的 prompt 渲染功能
func TestRenderPrompt(t *testing.T) {
	// 创建临时目录模拟 .orion 结构
	tmpDir, err := os.MkdirTemp("", "orion-agent-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 prompts 目录
	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	// 创建基础模板文件
	baseTemplate := `Task: {{.Task}}
Branch: {{.BaseBranch}}
Files: {{.ChangedFiles}}`
	baseTemplatePath := filepath.Join(promptsDir, "default.tmpl")
	if err := os.WriteFile(baseTemplatePath, []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("failed to write base template: %v", err)
	}

	// 测试渲染
	ctx := PromptContext{
		Task:         "Run unit tests",
		BaseBranch:   "main",
		ChangedFiles: []string{"main.go", "main_test.go"},
	}

	result, err := RenderPrompt(tmpDir, "default", "Test the code", ctx)
	if err != nil {
		t.Fatalf("RenderPrompt failed: %v", err)
	}

	// 验证结果
	if !strings.Contains(result, "Task: Test the code") {
		t.Errorf("result should contain 'Task: Test the code', got: %s", result)
	}
	if !strings.Contains(result, "Branch: main") {
		t.Errorf("result should contain 'Branch: main', got: %s", result)
	}
}

// TestRenderPromptWithTemplateVars 测试 agent prompt 本身包含模板变量
func TestRenderPromptWithTemplateVars(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-tmpl-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	// 基础模板
	baseTemplate := `{{.Task}}`
	baseTemplatePath := filepath.Join(promptsDir, "default.tmpl")
	if err := os.WriteFile(baseTemplatePath, []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("failed to write base template: %v", err)
	}

	// Agent prompt 包含模板变量
	agentPrompt := `Please review changes in branch {{.BaseBranch}}`

	ctx := PromptContext{
		BaseBranch: "feature/test",
	}

	result, err := RenderPrompt(tmpDir, "default", agentPrompt, ctx)
	if err != nil {
		t.Fatalf("RenderPrompt failed: %v", err)
	}

	if !strings.Contains(result, "Please review changes in branch feature/test") {
		t.Errorf("result should contain rendered agent prompt, got: %s", result)
	}
}

// TestRenderPromptWithMissingTemplate 测试基础模板不存在时的回退行为
func TestRenderPromptWithMissingTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-missing-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 不创建 prompts 目录，测试回退行为
	ctx := PromptContext{
		Task: "Simple task",
	}

	result, err := RenderPrompt(tmpDir, "nonexistent", "Do something", ctx)
	if err != nil {
		t.Fatalf("RenderPrompt should not fail with missing template: %v", err)
	}

	// 应该回退到默认模板，只返回 task
	// 默认模板是 {{.Task}}，所以应该返回 "Do something"
	if result != "Do something" {
		t.Errorf("result should be 'Do something', got: %s", result)
	}
}

// TestRenderPromptWithEmptyEnv 测试空环境变量的情况
func TestRenderPromptWithEmptyEnv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-empty-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	baseTemplate := `Task: {{.Task}}`
	baseTemplatePath := filepath.Join(promptsDir, "default.tmpl")
	if err := os.WriteFile(baseTemplatePath, []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("failed to write base template: %v", err)
	}

	ctx := PromptContext{
		Task:         "Test task",
		Env:          []string{},
		ChangedFiles: []string{},
		BaseBranch:   "",
	}

	result, err := RenderPrompt(tmpDir, "default", "Do it", ctx)
	if err != nil {
		t.Fatalf("RenderPrompt failed: %v", err)
	}

	// 模板渲染后应该是 "Task: Do it"（agent prompt 被用作 Task）
	if result != "Task: Do it" {
		t.Errorf("result should be 'Task: Do it', got: %s", result)
	}
}

// TestRenderPromptWithMultipleFiles 测试多个变更文件
func TestRenderPromptWithMultipleFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-multi-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	baseTemplate := `Files: {{.ChangedFiles}}`
	baseTemplatePath := filepath.Join(promptsDir, "default.tmpl")
	if err := os.WriteFile(baseTemplatePath, []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("failed to write base template: %v", err)
	}

	ctx := PromptContext{
		ChangedFiles: []string{"file1.go", "file2.go", "file3_test.go"},
	}

	result, err := RenderPrompt(tmpDir, "default", "Test", ctx)
	if err != nil {
		t.Fatalf("RenderPrompt failed: %v", err)
	}

	// Go 的默认格式化会输出 [file1.go file2.go file3_test.go]
	if !strings.Contains(result, "[file1.go file2.go file3_test.go]") {
		t.Errorf("result should contain all files, got: %s", result)
	}
}

// TestRenderPromptWithSpecialChars 测试特殊字符
func TestRenderPromptWithSpecialChars(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-special-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	baseTemplate := `Task: {{.Task}}`
	baseTemplatePath := filepath.Join(promptsDir, "default.tmpl")
	if err := os.WriteFile(baseTemplatePath, []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("failed to write base template: %v", err)
	}

	// 测试包含特殊字符的 task
	ctx := PromptContext{
		Task: "Test with <special> & 'chars'",
	}

	result, err := RenderPrompt(tmpDir, "default", ctx.Task, ctx)
	if err != nil {
		t.Fatalf("RenderPrompt failed: %v", err)
	}

	if !strings.Contains(result, "Test with <special> & 'chars'") {
		t.Errorf("result should preserve special characters, got: %s", result)
	}
}

// TestRenderPromptWithComplexTemplate 测试复杂模板
func TestRenderPromptWithComplexTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-complex-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	// 复杂模板包含条件逻辑
	baseTemplate := `# Code Review Task
{{.Task}}

## Context
- Base Branch: {{.BaseBranch}}
- Changed Files:
{{- range .ChangedFiles}}
  - {{.}}
{{- end}}
`
	baseTemplatePath := filepath.Join(promptsDir, "complex.tmpl")
	if err := os.WriteFile(baseTemplatePath, []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("failed to write base template: %v", err)
	}

	ctx := PromptContext{
		Task:         "Initial task (will be overridden)",
		BaseBranch:   "feature/auth",
		ChangedFiles: []string{"auth.go", "auth_test.go"},
	}

	// agentPrompt 会被渲染后赋值给 ctx.Task
	result, err := RenderPrompt(tmpDir, "complex", "Review the code carefully", ctx)
	if err != nil {
		t.Fatalf("RenderPrompt failed: %v", err)
	}

	// 验证模板渲染
	if !strings.Contains(result, "# Code Review Task") {
		t.Errorf("result should contain header")
	}
	// agentPrompt "Review the code carefully" 被赋值给 ctx.Task
	if !strings.Contains(result, "Review the code carefully") {
		t.Errorf("result should contain agentPrompt as task, got: %s", result)
	}
	if !strings.Contains(result, "Base Branch: feature/auth") {
		t.Errorf("result should contain base branch")
	}
	if !strings.Contains(result, "- auth.go") {
		t.Errorf("result should contain file list")
	}
}

// TestRenderPromptWithInvalidTemplate 测试无效模板语法
func TestRenderPromptWithInvalidTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-invalid-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	// 无效的模板语法（缺少闭合标签）
	invalidTemplate := `Task: {{.Task`
	baseTemplatePath := filepath.Join(promptsDir, "invalid.tmpl")
	if err := os.WriteFile(baseTemplatePath, []byte(invalidTemplate), 0644); err != nil {
		t.Fatalf("failed to write base template: %v", err)
	}

	ctx := PromptContext{
		Task: "Test",
	}

	_, err = RenderPrompt(tmpDir, "invalid", "Do it", ctx)
	if err == nil {
		t.Errorf("RenderPrompt should fail with invalid template syntax")
	}
}

// TestRenderPromptWithInvalidAgentPrompt 测试无效的 agent prompt 模板
func TestRenderPromptWithInvalidAgentPrompt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-invalid-prompt-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	baseTemplate := `{{.Task}}`
	baseTemplatePath := filepath.Join(promptsDir, "default.tmpl")
	if err := os.WriteFile(baseTemplatePath, []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("failed to write base template: %v", err)
	}

	// 无效的 agent prompt 模板
	invalidAgentPrompt := `{{.NonExistentField}}`

	ctx := PromptContext{
		Task: "Test",
	}

	_, err = RenderPrompt(tmpDir, "default", invalidAgentPrompt, ctx)
	if err == nil {
		t.Errorf("RenderPrompt should fail with invalid agent prompt template")
	}
}

// TestRenderPromptPreservesNewlines 测试保留换行符
func TestRenderPromptPreservesNewlines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-agent-newline-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	promptsDir := filepath.Join(tmpDir, ".orion", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	baseTemplate := `Line 1
Line 2
{{.Task}}
Line 4`
	baseTemplatePath := filepath.Join(promptsDir, "default.tmpl")
	if err := os.WriteFile(baseTemplatePath, []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("failed to write base template: %v", err)
	}

	ctx := PromptContext{
		Task: "Middle line",
	}

	result, err := RenderPrompt(tmpDir, "default", "Middle line", ctx)
	if err != nil {
		t.Fatalf("RenderPrompt failed: %v", err)
	}

	// 验证换行符被保留
	lines := strings.Split(result, "\n")
	if len(lines) < 4 {
		t.Errorf("result should have at least 4 lines, got %d", len(lines))
	}
}
