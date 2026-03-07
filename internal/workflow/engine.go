package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"devswarm/internal/git"
	"devswarm/internal/tmux"
	"devswarm/internal/types"
	"devswarm/internal/workspace"

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
func (e *Engine) StartRun(workflowName, trigger, baseBranch string) (*Run, error) {
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

	// 3. Create Run structure
	runID := fmt.Sprintf("run-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
	runDir := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create run directory: %w", err)
	}

	run := &Run{
		ID:         runID,
		Workflow:   workflowName,
		Trigger:    trigger,
		BaseBranch: baseBranch,
		Status:     StatusRunning, // Mark as running immediately
		StartTime:  time.Now(),
		Steps:      make([]StepStatus, len(wf.Pipeline)),
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
	// Naming: devswarm/<run-id>/<step-id>
	shadowBranch := fmt.Sprintf("devswarm/%s/%s", run.ID, step.ID)
	step.ShadowBranch = shadowBranch

	// 3. Spawn Node (Worktree + Shadow Branch + Tmux)
	node, err := e.spawnAgentNode(step.NodeName, shadowBranch, baseBranch)
	if err != nil {
		return fmt.Errorf("failed to spawn node: %w", err)
	}

	// 4. Load Agent Configuration
	agentPath := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.AgentsDir, stepDef.Agent+".yaml")
	agentData, err := os.ReadFile(agentPath)
	if err != nil {
		return fmt.Errorf("failed to load agent config %s: %w", stepDef.Agent, err)
	}
	var agent types.Agent
	if err := yaml.Unmarshal(agentData, &agent); err != nil {
		return fmt.Errorf("failed to parse agent config: %w", err)
	}

	// 5. Load and Render Prompt
	promptPath := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.PromptsDir, agent.Prompt)
	promptContent, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("failed to load prompt %s: %w", agent.Prompt, err)
	}

	// Get Diff Context (from base branch to HEAD of shadow branch - which is same as base initially)
	// Actually, we want the diff of the *previous step* or the *human commit*.
	// git diff baseBranch~1 baseBranch
	diffContext, err := e.getDiffContext(node.WorktreePath, "HEAD~1", "HEAD")
	if err != nil {
		// If HEAD~1 fails (e.g. initial commit), try empty
		diffContext = ""
	}

	renderedPrompt, err := e.renderPrompt(string(promptContent), map[string]string{
		"Branch": shadowBranch,
		"Diff":   diffContext,
	})
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	// Write prompt to file in worktree for agent to consume
	promptFile := filepath.Join(node.WorktreePath, "agent_prompt.md")
	if err := os.WriteFile(promptFile, []byte(renderedPrompt), 0644); err != nil {
		return fmt.Errorf("failed to write prompt file: %w", err)
	}

	// 6. Execute Agent Command
	// Construct command: qwen -p <prompt_file> -y
	// We run this using os/exec in the worktree directory.
	// If agent runtime is tmux, we could wrap it, but for simplicity/reliability in v1,
	// we run it as a subprocess attached to the created tmux session or just directly.
	// Since user wants "executor: tmux", we should try to use tmux if possible.
	// But getting exit code from detached tmux is hard.
	// Strategy: Run command via `tmux send-keys` and wait for a marker file.
	
	// Command to run inside tmux:
	// "qwen -p agent_prompt.md -y; echo $? > .agent_exit_code"
	agentCmd := fmt.Sprintf("%s -p agent_prompt.md -y", agent.Runtime.CodeAgent)
	fullCmd := fmt.Sprintf("%s; echo $? > .agent_exit_code", agentCmd)

	// We use the existing session created by spawnAgentNode
	sessionName := fmt.Sprintf("devswarm-%s", step.NodeName)
	if err := tmux.SendKeys(sessionName, fullCmd); err != nil {
		return fmt.Errorf("failed to send command to tmux: %w", err)
	}

	// Wait for completion (poll for .agent_exit_code)
	exitCode, err := e.waitForAgent(node.WorktreePath)
	if err != nil {
		return fmt.Errorf("agent execution error: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("agent failed with exit code %d", exitCode)
	}

	// 7. Commit Changes
	// The agent modified files in the worktree. We commit them to the shadow branch.
	if err := e.commitChanges(node.WorktreePath, fmt.Sprintf("Agent %s Result", step.ID)); err != nil {
		return fmt.Errorf("failed to commit agent changes: %w", err)
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

func (e *Engine) spawnAgentNode(nodeName, shadowBranch, baseBranch string) (*types.Node, error) {
	worktreePath := filepath.Join(e.wm.RootPath, workspace.WorkspacesDir, nodeName)

	// 1. Create Shadow Branch & Worktree
	// We use git.AddWorktree directly.
	// If shadowBranch == baseBranch, we are just checking it out (not typical for agent, but possible).
	// Typically shadowBranch is new, baseBranch is existing.
	if err := git.AddWorktree(e.wm.State.RepoPath, worktreePath, shadowBranch, baseBranch); err != nil {
		return nil, err
	}

	// 2. Create Tmux Session
	sessionName := fmt.Sprintf("devswarm-%s", nodeName)
	if err := tmux.NewSession(sessionName, worktreePath); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// 3. Update State
	node := types.Node{
		Name:          nodeName,
		LogicalBranch: baseBranch, // Logically related to base
		ShadowBranch:  shadowBranch,
		WorktreePath:  worktreePath,
		Purpose:       "agent",
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

	return runs, nil
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
