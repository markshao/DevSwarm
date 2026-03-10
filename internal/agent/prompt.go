package agent

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"
	"os"
)

// PromptContext holds data available to the prompt template.
type PromptContext struct {
	Task        string
	Env         []string
	ChangedFiles []string
	BaseBranch  string
}

// RenderPrompt loads the base template and the specific agent prompt,
// then renders the final prompt string.
// 
// rootDir: project root (where .orion/ is)
// baseTemplateName: e.g. "default" (maps to .orion/prompts/default.tmpl)
// agentPrompt: the specific instruction from the agent yaml
func RenderPrompt(rootDir, baseTemplateName, agentPrompt string, ctx PromptContext) (string, error) {
	// 1. Read Base Template
	basePath := filepath.Join(rootDir, ".orion", "prompts", baseTemplateName+".tmpl")
	baseContent, err := os.ReadFile(basePath)
	if err != nil {
		// Fallback if not found? Or strict error?
		// Let's use a built-in default if missing for bootstrapping
		baseContent = []byte(`{{.Task}}`) 
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to read base template %s: %w", basePath, err)
		}
	}

	// 2. Parse Template
	// We define a sub-template "task" which the base template can include if it wants,
	// or we simply inject the rendered agentPrompt into the base template.
	
	// Better approach:
	// The base template expects {{.Task}} to be the user's specific prompt.
	// We might also want to render the agentPrompt itself if it has variables.
	
	tmpl, err := template.New("base").Parse(string(baseContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse base template: %w", err)
	}

	// 3. Render
	// We treat agentPrompt as the "Task" field in the context.
	// If agentPrompt itself is a template, we should render it first.
	taskTmpl, err := template.New("task").Parse(agentPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to parse agent prompt: %w", err)
	}
	
	var taskBuf bytes.Buffer
	if err := taskTmpl.Execute(&taskBuf, ctx); err != nil {
		return "", fmt.Errorf("failed to execute agent prompt template: %w", err)
	}
	
	ctx.Task = taskBuf.String()
	
	var finalBuf bytes.Buffer
	if err := tmpl.Execute(&finalBuf, ctx); err != nil {
		return "", fmt.Errorf("failed to execute base template: %w", err)
	}

	return finalBuf.String(), nil
}
