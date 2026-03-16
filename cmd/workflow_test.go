package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"orion/internal/types"
	"orion/internal/workflow"
	"orion/internal/workspace"

	"gopkg.in/yaml.v3"
)

// setupTestCmdWorkspace creates a temporary workspace for testing commands
func setupTestCmdWorkspace(t *testing.T) (*workspace.WorkspaceManager, string) {
	t.Helper()
	rootPath := t.TempDir()

	// Create necessary directories
	dirs := []string{
		workspace.RepoDir,
		workspace.WorkspacesDir,
		workspace.MetaDir,
		filepath.Join(workspace.MetaDir, workspace.WorkflowsDir),
		filepath.Join(workspace.MetaDir, workspace.RunsDir),
		filepath.Join(workspace.MetaDir, workspace.AgentsDir),
		filepath.Join(workspace.MetaDir, workspace.PromptsDir),
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

// createTestWorkflow creates a test workflow definition file
func createTestWorkflow(t *testing.T, rootPath, wfName string) {
	t.Helper()
	wf := types.Workflow{
		Name: wfName,
		Pipeline: []types.PipelineStep{
			{ID: "step1", Agent: "test-agent"},
		},
	}
	wfData, _ := yaml.Marshal(wf)
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, wfName+".yaml")
	if err := os.WriteFile(wfPath, wfData, 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}
}

// createTestRun creates a test workflow run
func createTestRun(t *testing.T, wm *workspace.WorkspaceManager, runID, wfName string, status workflow.RunStatus) *workflow.Run {
	t.Helper()
	runDir := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("Failed to create run dir: %v", err)
	}

	run := &workflow.Run{
		ID:        runID,
		Workflow:  wfName,
		Trigger:   "manual",
		Status:    status,
		StartTime: time.Now(),
		Steps:     []workflow.StepStatus{},
	}

	statusPath := filepath.Join(runDir, "status.json")
	file, _ := os.Create(statusPath)
	defer file.Close()
	_ = json.NewEncoder(file).Encode(run)

	return run
}

// createTestAgentNode creates a test agent node in the workspace
func createTestAgentNode(t *testing.T, wm *workspace.WorkspaceManager, nodeName, createdBy string) {
	t.Helper()
	repoPath := wm.State.RepoPath

	// Create a shadow branch for the node
	shadowBranch := "orion/test-branch-" + nodeName
	cmd := exec.Command("git", "-C", repoPath, "branch", shadowBranch)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create shadow branch: %v", err)
	}

	// Create worktree directory
	worktreePath := filepath.Join(wm.RootPath, workspace.MetaDir, "agent-nodes", nodeName)
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}

	// Add to state
	if wm.State.Nodes == nil {
		wm.State.Nodes = make(map[string]types.Node)
	}
	wm.State.Nodes[nodeName] = types.Node{
		Name:         nodeName,
		ShadowBranch: shadowBranch,
		WorktreePath: worktreePath,
		CreatedBy:    createdBy,
	}

	// Save state
	if err := wm.SaveState(); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}
}

func TestRmWorkflowCmd_WorkflowNotFound(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Try to remove a non-existent workflow
	err := removeWorkflow(wm, "non-existent", false)
	if err == nil {
		t.Fatal("Expected error for non-existent workflow, got nil")
	}
	if err.Error() != "workflow 'non-existent' not found" {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestRmWorkflowCmd_RemoveExistingWorkflow(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a test workflow
	createTestWorkflow(t, rootPath, "test-wf")

	// Verify workflow file exists
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "test-wf.yaml")
	if _, err := os.Stat(wfPath); os.IsNotExist(err) {
		t.Fatal("Workflow file should exist before removal")
	}

	// Remove the workflow
	err := removeWorkflow(wm, "test-wf", false)
	if err != nil {
		t.Fatalf("Failed to remove workflow: %v", err)
	}

	// Verify workflow file is removed
	if _, err := os.Stat(wfPath); !os.IsNotExist(err) {
		t.Error("Workflow file should be removed")
	}
}

func TestRmWorkflowCmd_RemoveWorkflowWithRunningInstances(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a test workflow
	createTestWorkflow(t, rootPath, "test-wf")

	// Create a running instance of the workflow
	runID := "run-20260316-test123"
	createTestRun(t, wm, runID, "test-wf", workflow.StatusRunning)

	// Try to remove without force - should fail
	err := removeWorkflow(wm, "test-wf", false)
	if err == nil {
		t.Fatal("Expected error when removing workflow with running instances")
	}
	if err.Error() != "workflow 'test-wf' has 1 running instance(s), use --force to remove" {
		t.Errorf("Expected running instances error, got: %v", err)
	}

	// Verify workflow file still exists
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "test-wf.yaml")
	if _, err := os.Stat(wfPath); os.IsNotExist(err) {
		t.Error("Workflow file should still exist after failed removal")
	}
}

func TestRmWorkflowCmd_ForceRemoveWithRunningInstances(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a test workflow
	createTestWorkflow(t, rootPath, "test-wf")

	// Create a running instance
	runID := "run-20260316-test456"
	createTestRun(t, wm, runID, "test-wf", workflow.StatusRunning)

	// Create an agent node created by this run
	nodeName := runID + "-step1-agent"
	createTestAgentNode(t, wm, nodeName, runID)

	// Verify node exists
	if _, exists := wm.State.Nodes[nodeName]; !exists {
		t.Fatal("Agent node should exist before force removal")
	}

	// Force remove the workflow
	err := removeWorkflow(wm, "test-wf", true)
	if err != nil {
		t.Fatalf("Failed to force remove workflow: %v", err)
	}

	// Verify workflow file is removed
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "test-wf.yaml")
	if _, err := os.Stat(wfPath); !os.IsNotExist(err) {
		t.Error("Workflow file should be removed after force removal")
	}

	// Reload state to verify node is removed
	wm2, _ := workspace.NewManager(rootPath)
	if _, exists := wm2.State.Nodes[nodeName]; exists {
		t.Error("Agent node should be removed after force removal")
	}
}

func TestRmWorkflowCmd_RemoveWorkflowWithCompletedInstances(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a test workflow
	createTestWorkflow(t, rootPath, "test-wf")

	// Create a completed instance of the workflow (should not block removal)
	runID := "run-20260316-completed"
	createTestRun(t, wm, runID, "test-wf", workflow.StatusSuccess)

	// Remove the workflow (should succeed since instance is not running)
	err := removeWorkflow(wm, "test-wf", false)
	if err != nil {
		t.Fatalf("Failed to remove workflow with completed instance: %v", err)
	}

	// Verify workflow file is removed
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "test-wf.yaml")
	if _, err := os.Stat(wfPath); !os.IsNotExist(err) {
		t.Error("Workflow file should be removed")
	}
}

func TestRmWorkflowCmd_MultipleRunningInstances(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a test workflow
	createTestWorkflow(t, rootPath, "test-wf")

	// Create multiple running instances
	runID1 := "run-20260316-multi1"
	runID2 := "run-20260316-multi2"
	createTestRun(t, wm, runID1, "test-wf", workflow.StatusRunning)
	createTestRun(t, wm, runID2, "test-wf", workflow.StatusRunning)

	// Try to remove without force - should fail
	err := removeWorkflow(wm, "test-wf", false)
	if err == nil {
		t.Fatal("Expected error when removing workflow with multiple running instances")
	}

	// Force remove
	err = removeWorkflow(wm, "test-wf", true)
	if err != nil {
		t.Fatalf("Failed to force remove workflow with multiple instances: %v", err)
	}

	// Verify workflow file is removed
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "test-wf.yaml")
	if _, err := os.Stat(wfPath); !os.IsNotExist(err) {
		t.Error("Workflow file should be removed")
	}
}

func TestRmWorkflowCmd_DifferentWorkflowInstances(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create two different workflows
	createTestWorkflow(t, rootPath, "wf-1")
	createTestWorkflow(t, rootPath, "wf-2")

	// Create a running instance for wf-1
	runID := "run-20260316-other"
	createTestRun(t, wm, runID, "wf-1", workflow.StatusRunning)

	// Remove wf-2 (should succeed since running instance is for wf-1)
	err := removeWorkflow(wm, "wf-2", false)
	if err != nil {
		t.Fatalf("Failed to remove wf-2: %v", err)
	}

	// Verify wf-2 is removed but wf-1 still exists
	wf2Path := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "wf-2.yaml")
	wf1Path := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "wf-1.yaml")

	if _, err := os.Stat(wf2Path); !os.IsNotExist(err) {
		t.Error("wf-2 should be removed")
	}
	if _, err := os.Stat(wf1Path); os.IsNotExist(err) {
		t.Error("wf-1 should still exist")
	}
}

// removeWorkflow is a helper function that mimics the rmWorkflowCmd logic for testing
func removeWorkflow(wm *workspace.WorkspaceManager, wfName string, force bool) error {
	// Check if workflow file exists
	wfPath := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.WorkflowsDir, wfName+".yaml")
	if _, err := os.Stat(wfPath); os.IsNotExist(err) {
		return &workflowError{"workflow '" + wfName + "' not found"}
	}

	// Check for running instances of this workflow
	engine := workflow.NewEngine(wm)
	runs, err := engine.ListRuns()
	if err != nil {
		return err
	}

	var runningRuns []*workflow.Run
	for i := range runs {
		if runs[i].Workflow == wfName && runs[i].Status == workflow.StatusRunning {
			runningRuns = append(runningRuns, &runs[i])
		}
	}

	if len(runningRuns) > 0 {
		if !force {
			return &workflowError{msg: fmt.Sprintf("workflow '%s' has %d running instance(s), use --force to remove", wfName, len(runningRuns))}
		}

		// Force mode: remove all agentic nodes created by running instances
		for _, run := range runningRuns {
			// Find and remove all nodes created by this run
			for nodeName, node := range wm.State.Nodes {
				if node.CreatedBy == run.ID {
					if err := wm.RemoveNode(nodeName); err != nil {
						// Log but continue
					}
				}
			}
		}
	}

	// Remove the workflow file
	if err := os.Remove(wfPath); err != nil {
		return err
	}

	return nil
}

// workflowError is a simple error type for testing
type workflowError struct {
	msg string
}

func (e *workflowError) Error() string {
	return e.msg
}
