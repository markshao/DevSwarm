package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"orion/internal/agent"
	"orion/internal/git"
	"orion/internal/tmux"
	"orion/internal/types"
	"orion/internal/workspace"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Engine struct {
	wm *workspace.WorkspaceManager
}

func NewEngine(wm *workspace.WorkspaceManager) *Engine {
	return &Engine{wm: wm}
}

// StartRun initializes a new run and starts executing it.
// Currently synchronous.
func (e *Engine) StartRun(workflowName, trigger, baseBranch, triggeredByNode string) (*Run, error) {
	// 1. Load workflow definition
	wfPath := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.WorkflowsDir, workflowName+".yaml")
	wfData, err := os.ReadFile(wfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow %s: %w", workflowName, err)
	}

	var wf types.Workflow
	if err := yaml.Unmarshal(wfData, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow %s: %w", workflowName, err)
	}

	// 2. Use provided baseBranch or default to current branch of main repo if empty
	if baseBranch == "" {
		var err error
		baseBranch, err = git.GetCurrentBranch(e.wm.State.RepoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to determine base branch: %w", err)
		}
	}

	// Capture trigger data (e.g. commit hash) if triggered by commit
	triggerData := ""
	if trigger == "commit" {
		// Get latest commit hash from main repo
		hash, err := git.GetLatestCommitHash(e.wm.State.RepoPath)
		if err == nil {
			triggerData = hash[:7] // Short hash
		}
	}

	// 3. Create Run structure
	runID := fmt.Sprintf("run-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
	runDir := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create run directory: %w", err)
	}

	run := &Run{
		ID:              runID,
		Workflow:        workflowName,
		Trigger:         trigger,
		TriggerData:     triggerData,
		BaseBranch:      baseBranch,
		TriggeredByNode: triggeredByNode,
		Status:          StatusRunning, // Mark as running immediately
		StartTime:       time.Now(),
		Steps:           make([]StepStatus, len(wf.Pipeline)),
	}

	for i, step := range wf.Pipeline {
		run.Steps[i] = StepStatus{
			ID:     step.ID,
			Agent:  step.Agent,
			Status: StatusPending,
		}
	}

	// 4. Persist initial status
	if err := e.saveRunStatus(run); err != nil {
		return nil, err
	}

	// 5. Execute pipeline (Synchronous for now to ensure completion in CLI)
	// In a real system, this might be handed off to a worker pool or daemon.
	e.executePipeline(run, &wf)

	return run, nil
}

func (e *Engine) executePipeline(run *Run, wf *types.Workflow) {
	// Simple sequential execution
	for i, stepDef := range wf.Pipeline {
		step := &run.Steps[i]
		step.StartTime = time.Now()
		step.Status = StatusRunning
		step.NodeName = fmt.Sprintf("%s-%s-%s", run.ID, step.ID, stepDef.Suffix)
		_ = e.saveRunStatus(run)

		// Create Node and Execute Agent
		err := e.executeStep(run, step, &stepDef)

		step.EndTime = time.Now()
		if err != nil {
			step.Status = StatusFailed
			step.Error = err.Error()
			run.Status = StatusFailed
			run.EndTime = time.Now()
			_ = e.saveRunStatus(run)
			return // Stop execution on failure
		}
		step.Status = StatusSuccess
		_ = e.saveRunStatus(run)
	}

	run.Status = StatusSuccess
	run.EndTime = time.Now()
	_ = e.saveRunStatus(run)
}

func (e *Engine) executeStep(run *Run, step *StepStatus, stepDef *types.PipelineStep) error {
	// 1. Determine Base Branch (Dependency Chaining)
	baseBranch, err := e.resolveBaseBranch(run, stepDef)
	if err != nil {
		return fmt.Errorf("failed to resolve base branch: %w", err)
	}

	// 2. Define Shadow Branch
	shadowBranch := fmt.Sprintf("orion/%s/%s", run.ID, step.ID)
	step.ShadowBranch = shadowBranch

	// 3. Spawn Node
	node, err := e.spawnAgentNode(step.NodeName, shadowBranch, baseBranch, run.ID)
	if err != nil {
		return fmt.Errorf("failed to spawn node: %w", err)
	}

	// 4. Load Config
	config, err := e.wm.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 5. Load Agent Configuration
	agentPath := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.AgentsDir, stepDef.Agent+".yaml")
	agentData, err := os.ReadFile(agentPath)
	if err != nil {
		return fmt.Errorf("failed to load agent config %s: %w", stepDef.Agent, err)
	}
	var agentDef types.Agent
	if err := yaml.Unmarshal(agentData, &agentDef); err != nil {
		return fmt.Errorf("failed to parse agent config: %w", err)
	}

	// 6. Determine Provider
	providerName := agentDef.Runtime.Provider
	if providerName == "" {
		providerName = config.Agents.DefaultProvider
	}
	if providerName == "" {
		providerName = "qwen" // Fallback
	}

	providerSettings := config.Agents.Providers[providerName]

	// Create Provider
	prov, err := agent.NewProvider(agent.Config{
		Provider: providerName,
		Model:    agentDef.Runtime.Model, // Agent specific model overrides provider default?
		// Actually agentDef.Runtime.Model should take precedence if set,
		// or use providerSettings.Model
		APIKey:   os.Getenv(providerSettings.APIKeyEnv),
		Endpoint: providerSettings.Endpoint,
		Params:   providerSettings.Params,
	})
	if err != nil {
		return fmt.Errorf("failed to create agent provider: %w", err)
	}

	// Use model from agent definition if present, else from provider settings
	if agentDef.Runtime.Model == "" {
		// This logic belongs inside the provider or we handle it here by passing the right config
		// But agent.NewProvider takes agent.Config which has Model field.
		// Let's ensure we pass the right model.
		if providerSettings.Model != "" {
			// We need to re-create or update provider config?
			// Simpler: Just pass the right model to NewProvider
			// If agentDef has model, use it. Else use providerSettings.Model.
		}
	}
	// Re-do provider creation with correct model logic
	model := agentDef.Runtime.Model
	if model == "" {
		model = providerSettings.Model
	}

	prov, err = agent.NewProvider(agent.Config{
		Provider: providerName,
		Model:    model,
		APIKey:   os.Getenv(providerSettings.APIKeyEnv),
		Endpoint: providerSettings.Endpoint,
		Params:   providerSettings.Params,
	})
	if err != nil {
		return fmt.Errorf("failed to create agent provider: %w", err)
	}

	// 7. Prepare Prompt Context
	// Get changed files
	changedFiles, err := git.GetChangedFiles(node.WorktreePath, "HEAD~1", "HEAD")
	if err != nil {
		changedFiles = []string{}
	}

	promptCtx := agent.PromptContext{
		Task: agentDef.Prompt, // The specific task from agent yaml
		Env:  agentDef.Env,    // We should resolve env vars? Or just pass names?
		// The template expects strings.
		ChangedFiles: changedFiles,
		BaseBranch:   baseBranch,
	}

	// Resolve Env vars if needed, or just pass the list as context
	// The default template iterates over .Env, so let's pass the resolved values
	var resolvedEnv []string
	for _, envVar := range agentDef.Env {
		val := os.Getenv(envVar)
		if val != "" {
			resolvedEnv = append(resolvedEnv, fmt.Sprintf("%s=%s", envVar, val))
		} else {
			resolvedEnv = append(resolvedEnv, envVar)
		}
	}
	promptCtx.Env = resolvedEnv

	// 8. Render Prompt
	baseTemplate := agentDef.BaseTemplate
	if baseTemplate == "" {
		baseTemplate = "default"
	}

	finalPrompt, err := agent.RenderPrompt(e.wm.RootPath, baseTemplate, agentDef.Prompt, promptCtx)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	// 9. Execute Agent
	// We run this inside the node's worktree
	// Note: Our QwenProvider implementation currently just writes a file.
	// In a real scenario, it might call an API.

	output, err := prov.Run(context.Background(), finalPrompt, node.WorktreePath, resolvedEnv)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Log output (simulated logging)
	// In production, we might write this to a log file or stream it.
	_ = output

	// 10. Commit Changes
	// The agent should have modified files. We commit them.
	// We need to check if there are changes first.
	if hasChanges, _ := git.HasChanges(node.WorktreePath); hasChanges {
		if err := e.commitChanges(node.WorktreePath, fmt.Sprintf("Agent %s Result", step.ID)); err != nil {
			return fmt.Errorf("failed to commit agent changes: %w", err)
		}
	} else {
		// If agent didn't commit (e.g. QwenProvider just wrote a file but didn't git add/commit),
		// we might need to do it here if the instruction said "YOU MUST COMMIT".
		// But if the agent follows instructions, it should have committed.
		// For our mock QwenProvider, it creates a file but doesn't commit.
		// So let's force commit here for safety in this prototype.
		_ = e.commitChanges(node.WorktreePath, fmt.Sprintf("Agent %s Result (Auto-commit)", step.ID))
	}

	return nil
}

func (e *Engine) resolveBaseBranch(run *Run, stepDef *types.PipelineStep) (string, error) {
	if len(stepDef.DependsOn) == 0 {
		return run.BaseBranch, nil
	}

	// Find the shadow branch of the dependency
	// Assuming single dependency for now
	depID := stepDef.DependsOn[0]
	for _, s := range run.Steps {
		if s.ID == depID {
			if s.ShadowBranch == "" {
				return "", fmt.Errorf("dependency %s has no shadow branch", depID)
			}
			return s.ShadowBranch, nil
		}
	}
	return "", fmt.Errorf("dependency %s not found", depID)
}

func (e *Engine) spawnAgentNode(nodeName, shadowBranch, baseBranch, createdBy string) (*types.Node, error) {
	// Agent nodes are stored in .orion/agent-nodes/ to keep them hidden from the main workspace list
	agentNodesDir := filepath.Join(e.wm.RootPath, workspace.MetaDir, "agent-nodes")
	if err := os.MkdirAll(agentNodesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create agent nodes directory: %w", err)
	}
	worktreePath := filepath.Join(agentNodesDir, nodeName)

	// 1. Create Shadow Branch & Worktree
	// We use git.AddWorktree directly.
	// If shadowBranch == baseBranch, we are just checking it out (not typical for agent, but possible).
	// Typically shadowBranch is new, baseBranch is existing.
	if err := git.AddWorktree(e.wm.State.RepoPath, worktreePath, shadowBranch, baseBranch); err != nil {
		return nil, err
	}

	// 2. Create Tmux Session
	sessionName := fmt.Sprintf("orion-%s", nodeName)
	if err := tmux.NewSession(sessionName, worktreePath); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// 3. Update State
	node := types.Node{
		Name:          nodeName,
		LogicalBranch: baseBranch, // Logically related to base
		ShadowBranch:  shadowBranch,
		WorktreePath:  worktreePath,
		Label:         "agent",
		CreatedBy:     createdBy,
		TmuxSession:   sessionName,
		CreatedAt:     time.Now(),
	}

	if e.wm.State.Nodes == nil {
		e.wm.State.Nodes = make(map[string]types.Node)
	}
	e.wm.State.Nodes[nodeName] = node
	if err := e.wm.SaveState(); err != nil {
		return nil, err
	}
	e.wm.SyncVSCodeWorkspace()

	return &node, nil
}

func (e *Engine) getDiffContext(path, from, to string) (string, error) {
	cmd := exec.Command("git", "diff", from, to)
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (e *Engine) renderPrompt(tmplContent string, data interface{}) (string, error) {
	tmpl, err := template.New("prompt").Parse(tmplContent)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (e *Engine) waitForAgent(worktreePath string) (int, error) {
	markerFile := filepath.Join(worktreePath, ".agent_exit_code")
	timeout := time.After(5 * time.Minute) // 5 minute timeout for agent
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return -1, fmt.Errorf("timeout waiting for agent")
		case <-ticker.C:
			data, err := os.ReadFile(markerFile)
			if err == nil {
				// File exists, read exit code
				codeStr := strings.TrimSpace(string(data))
				var code int
				fmt.Sscanf(codeStr, "%d", &code)
				return code, nil
			}
		}
	}
}

func (e *Engine) commitChanges(worktreePath, msg string) error {
	// Clean up transient files before committing
	_ = os.Remove(filepath.Join(worktreePath, ".agent_exit_code"))
	_ = os.Remove(filepath.Join(worktreePath, "agent_prompt.md"))

	// git add .
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = worktreePath
	if err := addCmd.Run(); err != nil {
		return err
	}

	// git commit -m msg
	// Check if there are changes first?
	// git diff --cached --quiet returns 0 if no changes
	checkCmd := exec.Command("git", "diff", "--cached", "--quiet")
	checkCmd.Dir = worktreePath
	if err := checkCmd.Run(); err == nil {
		// No changes to commit
		return nil
	}

	commitCmd := exec.Command("git", "commit", "-m", msg)
	commitCmd.Dir = worktreePath
	return commitCmd.Run()
}

func (e *Engine) saveRunStatus(run *Run) error {
	path := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir, run.ID, "status.json")
	// Ensure parent directory exists to avoid failures when called from tests or
	// auxiliary tooling that may not have created the run directory yet.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(run)
}

func (e *Engine) ListRuns() ([]Run, error) {
	runsDir := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir)
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Run{}, nil
		}
		return nil, err
	}

	var runs []Run
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		statusPath := filepath.Join(runsDir, entry.Name(), "status.json")
		data, err := os.ReadFile(statusPath)
		if err != nil {
			continue // Skip corrupted/incomplete runs
		}

		var run Run
		if err := json.Unmarshal(data, &run); err == nil {
			runs = append(runs, run)
		}
	}

	// Sort by start time descending
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartTime.After(runs[j].StartTime)
	})

	// Deduplicate runs by ID (in case of stale data or file system glitches)
	seen := make(map[string]bool)
	uniqueRuns := []Run{}
	for _, run := range runs {
		if !seen[run.ID] {
			seen[run.ID] = true
			uniqueRuns = append(uniqueRuns, run)
		}
	}

	return uniqueRuns, nil
}

func (e *Engine) GetRun(runID string) (*Run, error) {
	path := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID, "status.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var run Run
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, err
	}
	return &run, nil
}
