package workflow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orion/internal/types"
	"orion/internal/workspace"

	"gopkg.in/yaml.v3"
)

// setupTestWorkspace creates a temporary workspace for testing
func setupTestWorkspace(t *testing.T) (*workspace.WorkspaceManager, string) {
	t.Helper()
	rootPath := t.TempDir()

	// Create necessary directories
	dirs := []string{
		workspace.RepoDir,
		workspace.WorkspacesDir,
		workspace.MetaDir,
		filepath.Join(workspace.MetaDir, workspace.WorkflowsDir),
		filepath.Join(workspace.MetaDir, workspace.RunsDir),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(rootPath, d), 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", d, err)
		}
	}

	repoPath := filepath.Join(rootPath, workspace.RepoDir)

	// Initialize git repo
	cmd := exec.Command("git", "init", repoPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Config user
	_ = exec.Command("git", "-C", repoPath, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", repoPath, "config", "user.name", "Test User").Run()

	// Initial commit
	readme := filepath.Join(repoPath, "README.md")
	_ = os.WriteFile(readme, []byte("# Test Repo"), 0644)
	_ = exec.Command("git", "-C", repoPath, "add", ".").Run()
	_ = exec.Command("git", "-C", repoPath, "commit", "-m", "Initial commit").Run()

	// Create state.json
	state := types.State{
		RepoPath: repoPath,
	}
	stateData, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(rootPath, workspace.MetaDir, workspace.StateFile), stateData, 0644); err != nil {
		t.Fatalf("Failed to write state.json: %v", err)
	}

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("Failed to load workspace manager: %v", err)
	}
	return wm, rootPath
}

func TestStartRun(t *testing.T) {
	wm, rootPath := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Create a dummy workflow
	wf := types.Workflow{
		Name: "test-workflow",
		Pipeline: []types.PipelineStep{
			{ID: "step1", Agent: "test-agent"},
		},
	}
	wfData, _ := yaml.Marshal(wf)
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "test-workflow.yaml")
	if err := os.WriteFile(wfPath, wfData, 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// We expect StartRun to fail at execution step because "test-agent" config doesn't exist
	// and we don't want to actually run agent.
	// But it should create the Run object and persist it before failing.

	// Create dummy agent config to pass config loading
	agentPath := filepath.Join(rootPath, workspace.MetaDir, workspace.AgentsDir)
	os.MkdirAll(agentPath, 0755)
	agentConfig := types.Agent{
		Name:   "test-agent",
		Prompt: "test.md",
		Runtime: types.AgentRuntime{
			Provider: "test-provider",
			Model:    "test-model",
		},
	}
	agentData, _ := yaml.Marshal(agentConfig)
	os.WriteFile(filepath.Join(agentPath, "test-agent.yaml"), agentData, 0644)

	// Create dummy prompt
	promptPath := filepath.Join(rootPath, workspace.MetaDir, workspace.PromptsDir)
	os.MkdirAll(promptPath, 0755)
	os.WriteFile(filepath.Join(promptPath, "test.md"), []byte("hello"), 0644)

	// Start Run
	run, err := engine.StartRun("test-workflow", "manual", "master", "test-node")

	// Even if it fails during execution (e.g. git worktree add might fail if we are not careful with branches),
	// it should return a run object.
	// Actually StartRun executes synchronously. If execution fails, it returns the run object but with error?
	// No, StartRun returns (*Run, error). If setup fails, it returns error.
	// If pipeline execution fails, it returns run (with status Failed) and nil error.

	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	if run.ID == "" {
		t.Error("Run ID is empty")
	}
	if run.Status != StatusSuccess && run.Status != StatusFailed {
		t.Errorf("Unexpected run status: %s", run.Status)
	}
	if run.TriggeredByNode != "test-node" {
		t.Errorf("Expected TriggeredByNode 'test-node', got '%s'", run.TriggeredByNode)
	}

	// Verify persistence
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("Expected 1 run, got %d", len(runs))
	}
	if runs[0].ID != run.ID {
		t.Errorf("Run ID mismatch in ListRuns")
	}
}

// TestListRunsEmpty tests ListRuns when no runs exist
func TestListRunsEmpty(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}
}

// TestListRunsMultiple tests ListRuns with multiple runs
func TestListRunsMultiple(t *testing.T) {
	wm, rootPath := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Create workflow file
	wf := types.Workflow{
		Name: "test-wf",
		Pipeline: []types.PipelineStep{
			{ID: "step1", Agent: "test-agent"},
		},
	}
	wfData, _ := yaml.Marshal(wf)
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "test-wf.yaml")
	os.WriteFile(wfPath, wfData, 0644)

	// Create agent config and prompt
	agentPath := filepath.Join(rootPath, workspace.MetaDir, workspace.AgentsDir)
	os.MkdirAll(agentPath, 0755)
	agentConfig := types.Agent{
		Name:   "test-agent",
		Prompt: "test.md",
		Runtime: types.AgentRuntime{
			Provider: "test-provider",
			Model:    "test-model",
		},
	}
	agentData, _ := yaml.Marshal(agentConfig)
	os.WriteFile(filepath.Join(agentPath, "test-agent.yaml"), agentData, 0644)

	promptPath := filepath.Join(rootPath, workspace.MetaDir, workspace.PromptsDir)
	os.MkdirAll(promptPath, 0755)
	os.WriteFile(filepath.Join(promptPath, "test.md"), []byte("hello"), 0644)

	// Start multiple runs
	var runIDs []string
	for i := 0; i < 3; i++ {
		run, err := engine.StartRun("test-wf", "manual", "master", "")
		if err != nil {
			t.Fatalf("StartRun %d failed: %v", i, err)
		}
		runIDs = append(runIDs, run.ID)
		time.Sleep(10 * time.Millisecond) // Ensure different start times
	}

	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	if len(runs) != 3 {
		t.Errorf("expected 3 runs, got %d", len(runs))
	}

	// Verify runs are sorted by start time descending
	for i := 1; i < len(runs); i++ {
		if runs[i-1].StartTime.Before(runs[i].StartTime) {
			t.Error("runs should be sorted by start time descending")
			break
		}
	}
}

// TestGetRun tests getting a specific run
func TestGetRun(t *testing.T) {
	wm, rootPath := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Create workflow
	wf := types.Workflow{
		Name: "test-wf",
		Pipeline: []types.PipelineStep{
			{ID: "step1", Agent: "test-agent"},
		},
	}
	wfData, _ := yaml.Marshal(wf)
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "test-wf.yaml")
	os.WriteFile(wfPath, wfData, 0644)

	// Create agent config and prompt
	agentPath := filepath.Join(rootPath, workspace.MetaDir, workspace.AgentsDir)
	os.MkdirAll(agentPath, 0755)
	agentConfig := types.Agent{
		Name:   "test-agent",
		Prompt: "test.md",
		Runtime: types.AgentRuntime{
			Provider: "test-provider",
			Model:    "test-model",
		},
	}
	agentData, _ := yaml.Marshal(agentConfig)
	os.WriteFile(filepath.Join(agentPath, "test-agent.yaml"), agentData, 0644)

	promptPath := filepath.Join(rootPath, workspace.MetaDir, workspace.PromptsDir)
	os.MkdirAll(promptPath, 0755)
	os.WriteFile(filepath.Join(promptPath, "test.md"), []byte("hello"), 0644)

	run, err := engine.StartRun("test-wf", "manual", "master", "test-node")
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	// Get the run
	retrieved, err := engine.GetRun(run.ID)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}

	if retrieved.ID != run.ID {
		t.Errorf("run ID mismatch: got %s, want %s", retrieved.ID, run.ID)
	}
	if retrieved.Workflow != "test-wf" {
		t.Errorf("workflow mismatch: got %s, want test-wf", retrieved.Workflow)
	}
	if retrieved.TriggeredByNode != "test-node" {
		t.Errorf("triggered by node mismatch: got %s, want %s", retrieved.TriggeredByNode, "test-node")
	}
}

// TestGetRunNotFound tests getting a non-existent run
func TestGetRunNotFound(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	_, err := engine.GetRun("non-existent-run-id")
	if err == nil {
		t.Error("expected error for non-existent run")
	}
}

// TestStartRunWithNonExistentWorkflow tests StartRun with a workflow that doesn't exist
func TestStartRunWithNonExistentWorkflow(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	_, err := engine.StartRun("non-existent-workflow", "manual", "master", "")
	if err == nil {
		t.Error("expected error for non-existent workflow")
	}
	if !strings.Contains(err.Error(), "failed to read workflow") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestStartRunWithInvalidWorkflowYAML tests StartRun with invalid YAML
func TestStartRunWithInvalidWorkflowYAML(t *testing.T) {
	wm, rootPath := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Create invalid YAML
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "invalid.yaml")
	os.WriteFile(wfPath, []byte("invalid: yaml: content: ["), 0644)

	_, err := engine.StartRun("invalid", "manual", "master", "")
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "failed to parse workflow") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestRenderPrompt tests the renderPrompt function
func TestRenderPrompt(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	tmplContent := `Hello {{.Name}},
Your branch is {{.Branch}}.
Commit: {{.CommitID}}`

	data := map[string]string{
		"Name":     "User",
		"Branch":   "feature/test",
		"CommitID": "abc123",
	}

	result, err := engine.renderPrompt(tmplContent, data)
	if err != nil {
		t.Fatalf("renderPrompt failed: %v", err)
	}

	expected := "Hello User,\nYour branch is feature/test.\nCommit: abc123"
	if result != expected {
		t.Errorf("rendered prompt mismatch:\ngot: %q\nwant: %q", result, expected)
	}
}

// TestRenderPromptWithInvalidTemplate tests renderPrompt with invalid template
func TestRenderPromptWithInvalidTemplate(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Invalid template syntax
	tmplContent := `Hello {{.Name`

	_, err := engine.renderPrompt(tmplContent, map[string]string{"Name": "User"})
	if err == nil {
		t.Error("expected error for invalid template")
	}
}

// TestRenderPromptWithMissingData tests renderPrompt with missing template data
func TestRenderPromptWithMissingData(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	tmplContent := `Hello {{.Name}}, value: {{.Missing}}`

	data := map[string]string{
		"Name": "User",
		// Missing is intentionally not provided
	}

	result, err := engine.renderPrompt(tmplContent, data)
	if err != nil {
		t.Fatalf("renderPrompt failed: %v", err)
	}

	// Template should render with empty value for missing key
	if !strings.Contains(result, "Hello User, value:") {
		t.Errorf("unexpected result: %s", result)
	}
}

// TestSaveRunStatus tests saving run status to file
func TestSaveRunStatus(t *testing.T) {
	wm, rootPath := setupTestWorkspace(t)
	engine := NewEngine(wm)

	run := &Run{
		ID:         "test-run-123",
		Workflow:   "test-workflow",
		Trigger:    "manual",
		BaseBranch: "main",
		Status:     StatusRunning,
		StartTime:  time.Now(),
		Steps: []StepStatus{
			{ID: "step1", Agent: "agent1", Status: StatusPending},
		},
	}

	// Create run directory
	runDir := filepath.Join(rootPath, workspace.MetaDir, workspace.RunsDir, run.ID)
	os.MkdirAll(runDir, 0755)

	err := engine.saveRunStatus(run)
	if err != nil {
		t.Fatalf("saveRunStatus failed: %v", err)
	}

	// Verify file was created
	statusPath := filepath.Join(runDir, "status.json")
	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		t.Fatal("status.json was not created")
	}

	// Verify content
	data, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatalf("failed to read status.json: %v", err)
	}

	var loaded Run
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal status.json: %v", err)
	}

	if loaded.ID != run.ID {
		t.Errorf("run ID mismatch: got %s, want %s", loaded.ID, run.ID)
	}
	if loaded.Status != StatusRunning {
		t.Errorf("status mismatch: got %s, want %s", loaded.Status, StatusRunning)
	}
}

// TestWaitForAgentTimeout tests waitForAgent timeout behavior
func TestWaitForAgentTimeout(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Create temp directory
	worktreePath := t.TempDir()

	// Don't create the marker file, so it should timeout
	sessionName := "orion-test-timeout"

	// This will timeout quickly because marker file doesn't exist
	// We use a modified version that times out faster for testing
	// For now, just verify the function handles missing files
	_, err := engine.waitForAgent(sessionName, worktreePath)
	if err == nil {
		t.Error("expected error when waiting for non-existent agent")
	}
}

// TestWaitForAgentWithExitCode tests waitForAgent with exit code file
func TestWaitForAgentWithExitCode(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	worktreePath := t.TempDir()

	// Create marker file with exit code
	markerFile := filepath.Join(worktreePath, ".agent_exit_code")
	if err := os.WriteFile(markerFile, []byte("0"), 0644); err != nil {
		t.Fatalf("failed to create marker file: %v", err)
	}

	exitCode, err := engine.waitForAgent("orion-test", worktreePath)
	if err != nil {
		t.Fatalf("waitForAgent failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

// TestWaitForAgentWithNonZeroExitCode tests waitForAgent with non-zero exit code
func TestWaitForAgentWithNonZeroExitCode(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	worktreePath := t.TempDir()

	// Create marker file with non-zero exit code
	markerFile := filepath.Join(worktreePath, ".agent_exit_code")
	if err := os.WriteFile(markerFile, []byte("42"), 0644); err != nil {
		t.Fatalf("failed to create marker file: %v", err)
	}

	exitCode, err := engine.waitForAgent("orion-test", worktreePath)
	if err != nil {
		t.Fatalf("waitForAgent failed: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}
}

// TestRunStatusTransitions tests run status transitions
func TestRunStatusTransitions(t *testing.T) {
	// Verify status constants
	if StatusRunning == "" {
		t.Error("StatusRunning should not be empty")
	}
	if StatusSuccess == "" {
		t.Error("StatusSuccess should not be empty")
	}
	if StatusFailed == "" {
		t.Error("StatusFailed should not be empty")
	}
	if StatusPending == "" {
		t.Error("StatusPending should not be empty")
	}
}

// TestStepStatus tests StepStatus structure
func TestStepStatus(t *testing.T) {
	step := StepStatus{
		ID:     "step1",
		Agent:  "test-agent",
		Status: StatusPending,
	}

	if step.ID != "step1" {
		t.Errorf("step ID mismatch")
	}
	if step.Agent != "test-agent" {
		t.Errorf("step agent mismatch")
	}
	if step.Status != StatusPending {
		t.Errorf("step status mismatch")
	}
}
