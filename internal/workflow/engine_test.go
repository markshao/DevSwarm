package workflow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

// TestStartRunWithMissingWorkflow tests error handling when workflow file is missing
func TestStartRunWithMissingWorkflow(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	_, err := engine.StartRun("non-existent-workflow", "manual", "main", "test-node")
	if err == nil {
		t.Errorf("StartRun should fail with missing workflow file")
	}
}

// TestStartRunWithInvalidWorkflow tests error handling when workflow file is invalid YAML
func TestStartRunWithInvalidWorkflow(t *testing.T) {
	wm, rootPath := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Create invalid YAML
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "invalid.yaml")
	if err := os.WriteFile(wfPath, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write invalid workflow file: %v", err)
	}

	_, err := engine.StartRun("invalid", "manual", "main", "test-node")
	if err == nil {
		t.Errorf("StartRun should fail with invalid YAML")
	}
}

// TestStartRunWithEmptyBaseBranch tests that empty base branch falls back to current branch
func TestStartRunWithEmptyBaseBranch(t *testing.T) {
	wm, rootPath := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Create a valid workflow
	wf := types.Workflow{
		Name: "test-workflow",
		Pipeline: []types.PipelineStep{
			{ID: "step1", Agent: "test-agent", Suffix: "ut"},
		},
	}
	wfData, _ := yaml.Marshal(wf)
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "test-workflow.yaml")
	if err := os.WriteFile(wfPath, wfData, 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Create agent config
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

	// Create prompt
	promptPath := filepath.Join(rootPath, workspace.MetaDir, workspace.PromptsDir)
	os.MkdirAll(promptPath, 0755)
	os.WriteFile(filepath.Join(promptPath, "test.md"), []byte("hello"), 0644)

	// Start run with empty base branch
	run, err := engine.StartRun("test-workflow", "manual", "", "test-node")
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	// Should default to "main" (the current branch in test repo)
	if run.BaseBranch != "main" {
		t.Errorf("Expected BaseBranch to default to 'main', got '%s'", run.BaseBranch)
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
		t.Errorf("Expected 0 runs, got %d", len(runs))
	}
}

// TestListRunsMultiple tests ListRuns with multiple runs
func TestListRunsMultiple(t *testing.T) {
	wm, rootPath := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Create multiple run status files manually
	runIDs := []string{"run-20240101-aaa", "run-20240102-bbb", "run-20240103-ccc"}
	for i, runID := range runIDs {
		runDir := filepath.Join(rootPath, workspace.MetaDir, workspace.RunsDir, runID)
		if err := os.MkdirAll(runDir, 0755); err != nil {
			t.Fatalf("Failed to create run dir: %v", err)
		}

		run := Run{
			ID:         runID,
			Workflow:   "test",
			Trigger:    "manual",
			BaseBranch: "main",
			Status:     StatusSuccess,
			StartTime:  time.Now().Add(time.Duration(i) * time.Hour),
		}
		runData, _ := json.Marshal(run)
		statusPath := filepath.Join(runDir, "status.json")
		if err := os.WriteFile(statusPath, runData, 0644); err != nil {
			t.Fatalf("Failed to write status.json: %v", err)
		}
	}

	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	// Should return 3 runs sorted by start time descending
	if len(runs) != 3 {
		t.Errorf("Expected 3 runs, got %d", len(runs))
	}

	// Verify sorting (descending by start time)
	for i := 0; i < len(runs)-1; i++ {
		if runs[i].StartTime.Before(runs[i+1].StartTime) {
			t.Errorf("Runs should be sorted by StartTime descending")
		}
	}
}

// TestGetRun tests GetRun method
func TestGetRun(t *testing.T) {
	wm, rootPath := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Create a run manually
	runID := "run-20240101-test"
	runDir := filepath.Join(rootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("Failed to create run dir: %v", err)
	}

	expectedRun := Run{
		ID:         runID,
		Workflow:   "test-workflow",
		Trigger:    "commit",
		BaseBranch: "feature/test",
		Status:     StatusRunning,
		StartTime:  time.Now(),
	}
	runData, _ := json.Marshal(expectedRun)
	statusPath := filepath.Join(runDir, "status.json")
	if err := os.WriteFile(statusPath, runData, 0644); err != nil {
		t.Fatalf("Failed to write status.json: %v", err)
	}

	// Get the run
	run, err := engine.GetRun(runID)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}

	if run.ID != expectedRun.ID {
		t.Errorf("Run ID mismatch: got %s, want %s", run.ID, expectedRun.ID)
	}
	if run.Workflow != expectedRun.Workflow {
		t.Errorf("Workflow mismatch: got %s, want %s", run.Workflow, expectedRun.Workflow)
	}
	if run.Status != expectedRun.Status {
		t.Errorf("Status mismatch: got %s, want %s", run.Status, expectedRun.Status)
	}
}

// TestGetRunMissing tests GetRun with non-existent run
func TestGetRunMissing(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	_, err := engine.GetRun("non-existent-run")
	if err == nil {
		t.Errorf("GetRun should fail with non-existent run")
	}
}

// TestResolveBaseBranchNoDeps tests resolveBaseBranch with no dependencies
func TestResolveBaseBranchNoDeps(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	run := &Run{
		ID:         "test-run",
		BaseBranch: "main",
	}
	stepDef := &types.PipelineStep{
		ID:        "step1",
		DependsOn: []string{},
	}

	baseBranch, err := engine.resolveBaseBranch(run, stepDef)
	if err != nil {
		t.Fatalf("resolveBaseBranch failed: %v", err)
	}
	if baseBranch != "main" {
		t.Errorf("Expected base branch 'main', got '%s'", baseBranch)
	}
}

// TestResolveBaseBranchWithDeps tests resolveBaseBranch with dependencies
func TestResolveBaseBranchWithDeps(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	run := &Run{
		ID:         "test-run",
		BaseBranch: "main",
		Steps: []StepStatus{
			{ID: "step1", ShadowBranch: "orion/test-run/step1"},
			{ID: "step2", ShadowBranch: "orion/test-run/step2"},
		},
	}
	stepDef := &types.PipelineStep{
		ID:        "step3",
		DependsOn: []string{"step2"},
	}

	baseBranch, err := engine.resolveBaseBranch(run, stepDef)
	if err != nil {
		t.Fatalf("resolveBaseBranch failed: %v", err)
	}
	if baseBranch != "orion/test-run/step2" {
		t.Errorf("Expected base branch 'orion/test-run/step2', got '%s'", baseBranch)
	}
}

// TestResolveBaseBranchMissingDep tests resolveBaseBranch with missing dependency
func TestResolveBaseBranchMissingDep(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	run := &Run{
		ID:         "test-run",
		BaseBranch: "main",
		Steps: []StepStatus{
			{ID: "step1", ShadowBranch: "orion/test-run/step1"},
		},
	}
	stepDef := &types.PipelineStep{
		ID:        "step2",
		DependsOn: []string{"non-existent-step"},
	}

	_, err := engine.resolveBaseBranch(run, stepDef)
	if err == nil {
		t.Errorf("resolveBaseBranch should fail with missing dependency")
	}
}

// TestRenderPrompt tests the renderPrompt helper
func TestRenderPromptHelper(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	tmplContent := `Hello {{.Name}}, you are {{.Age}} years old.`
	data := map[string]string{
		"Name": "Alice",
		"Age":  "30",
	}

	result, err := engine.renderPrompt(tmplContent, data)
	if err != nil {
		t.Fatalf("renderPrompt failed: %v", err)
	}

	expected := "Hello Alice, you are 30 years old."
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestRenderPromptWithInvalidTemplate tests renderPrompt with invalid template
func TestRenderPromptWithInvalidTemplate(t *testing.T) {
	wm, _ := setupTestWorkspace(t)
	engine := NewEngine(wm)

	tmplContent := `Invalid {{.Missing`
	data := map[string]string{}

	_, err := engine.renderPrompt(tmplContent, data)
	if err == nil {
		t.Errorf("renderPrompt should fail with invalid template")
	}
}

// TestRunStatusSerialization tests Run status JSON serialization
func TestRunStatusSerialization(t *testing.T) {
	run := Run{
		ID:              "test-run",
		Workflow:        "test",
		Trigger:         "manual",
		BaseBranch:      "main",
		TriggeredByNode: "node1",
		Status:          StatusSuccess,
		StartTime:       time.Now(),
		EndTime:         time.Now(),
		Steps: []StepStatus{
			{
				ID:           "step1",
				Agent:        "test-agent",
				Status:       StatusSuccess,
				StartTime:    time.Now(),
				EndTime:      time.Now(),
				NodeName:     "node1",
				ShadowBranch: "orion/step1",
			},
		},
	}

	// Serialize
	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Deserialize
	var decoded Run
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.ID != run.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, run.ID)
	}
	if decoded.Status != run.Status {
		t.Errorf("Status mismatch: got %s, want %s", decoded.Status, run.Status)
	}
	if len(decoded.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(decoded.Steps))
	}
}

// TestRunStatusConstants tests RunStatus constants
func TestRunStatusConstants(t *testing.T) {
	tests := []struct {
		status   RunStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusSuccess, "success"},
		{StatusFailed, "failed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Status %s has unexpected string value", tt.status)
		}
	}
}
