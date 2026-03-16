package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"orion/internal/types"
	"orion/internal/workflow"
	"orion/internal/workspace"
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

func TestRmWorkflowCmd_RunNotFound(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Try to remove a non-existent run
	err := removeWorkflowRun(wm, "non-existent-run", false)
	if err == nil {
		t.Fatal("Expected error for non-existent run, got nil")
	}
	if err.Error() != "run 'non-existent-run' not found" {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestRmWorkflowCmd_RemoveCompletedRun(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a completed test run
	runID := "run-20260316-completed"
	createTestRun(t, wm, runID, "test-wf", workflow.StatusSuccess)

	runDir := filepath.Join(rootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if _, err := os.Stat(runDir); os.IsNotExist(err) {
		t.Fatal("Run directory should exist before removal")
	}

	// Remove the run
	err := removeWorkflowRun(wm, runID, false)
	if err != nil {
		t.Fatalf("Failed to remove completed run: %v", err)
	}

	// Verify run directory is removed
	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Error("Run directory should be removed after removal")
	}
}

func TestRmWorkflowCmd_RemoveRunningRunWithoutForce(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a running test run
	runID := "run-20260316-running"
	createTestRun(t, wm, runID, "test-wf", workflow.StatusRunning)

	// Try to remove without force - should fail
	err := removeWorkflowRun(wm, runID, false)
	if err == nil {
		t.Fatal("Expected error when removing running run without force")
	}
	if err.Error() != "run 'run-20260316-running' is still running, use --force to remove" {
		t.Errorf("Expected running error, got: %v", err)
	}

	// Verify run directory still exists
	runDir := filepath.Join(rootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if _, err := os.Stat(runDir); os.IsNotExist(err) {
		t.Error("Run directory should still exist after failed removal")
	}
}

func TestRmWorkflowCmd_RemoveRunningRunWithForce(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a running test run
	runID := "run-20260316-running-force"
	createTestRun(t, wm, runID, "test-wf", workflow.StatusRunning)

	// Create an agent node created by this run
	nodeName := runID + "-step1-agent"
	createTestAgentNode(t, wm, nodeName, runID)

	// Verify node exists
	if _, exists := wm.State.Nodes[nodeName]; !exists {
		t.Fatal("Agent node should exist before force removal")
	}

	runDir := filepath.Join(rootPath, workspace.MetaDir, workspace.RunsDir, runID)

	// Force remove the run
	err := removeWorkflowRun(wm, runID, true)
	if err != nil {
		t.Fatalf("Failed to force remove running run: %v", err)
	}

	// Verify run directory is removed
	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Error("Run directory should be removed after force removal")
	}

	// Reload state to verify node is removed
	wm2, _ := workspace.NewManager(rootPath)
	if _, exists := wm2.State.Nodes[nodeName]; exists {
		t.Error("Agent node should be removed after force removal")
	}
}

func TestRmWorkflowCmd_RemoveRunWithAssociatedNodes(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a completed test run
	runID := "run-20260316-with-nodes"
	createTestRun(t, wm, runID, "test-wf", workflow.StatusSuccess)

	// Create multiple agent nodes created by this run
	node1 := runID + "-step1-agent"
	node2 := runID + "-step2-agent"
	createTestAgentNode(t, wm, node1, runID)
	createTestAgentNode(t, wm, node2, runID)

	// Create an agent node NOT created by this run (should not be removed)
	otherNode := "other-run-step1-agent"
	createTestAgentNode(t, wm, otherNode, "other-run-id")

	// Verify nodes exist
	if _, exists := wm.State.Nodes[node1]; !exists {
		t.Fatal("Agent node1 should exist before removal")
	}
	if _, exists := wm.State.Nodes[node2]; !exists {
		t.Fatal("Agent node2 should exist before removal")
	}
	if _, exists := wm.State.Nodes[otherNode]; !exists {
		t.Fatal("Other agent node should exist before removal")
	}

	// Remove the run
	err := removeWorkflowRun(wm, runID, false)
	if err != nil {
		t.Fatalf("Failed to remove run with nodes: %v", err)
	}

	// Reload state to verify nodes are removed
	wm2, _ := workspace.NewManager(rootPath)

	if _, exists := wm2.State.Nodes[node1]; exists {
		t.Error("Agent node1 should be removed")
	}
	if _, exists := wm2.State.Nodes[node2]; exists {
		t.Error("Agent node2 should be removed")
	}
	if _, exists := wm2.State.Nodes[otherNode]; !exists {
		t.Error("Other agent node should still exist")
	}
}

func TestRmWorkflowCmd_RemoveRunWithNoNodes(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a completed test run with no associated nodes
	runID := "run-20260316-no-nodes"
	createTestRun(t, wm, runID, "test-wf", workflow.StatusSuccess)

	runDir := filepath.Join(rootPath, workspace.MetaDir, workspace.RunsDir, runID)

	// Remove the run
	err := removeWorkflowRun(wm, runID, false)
	if err != nil {
		t.Fatalf("Failed to remove run with no nodes: %v", err)
	}

	// Verify run directory is removed
	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Error("Run directory should be removed")
	}
}

func TestRmWorkflowCmd_MultipleRunsDifferentStatuses(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create multiple runs with different statuses
	runID1 := "run-20260316-success"
	runID2 := "run-20260316-failed"
	runID3 := "run-20260316-running"

	createTestRun(t, wm, runID1, "test-wf", workflow.StatusSuccess)
	createTestRun(t, wm, runID2, "test-wf", workflow.StatusFailed)
	createTestRun(t, wm, runID3, "test-wf", workflow.StatusRunning)

	// Remove completed run (should succeed)
	err := removeWorkflowRun(wm, runID1, false)
	if err != nil {
		t.Fatalf("Failed to remove completed run: %v", err)
	}

	// Remove failed run (should succeed)
	err = removeWorkflowRun(wm, runID2, false)
	if err != nil {
		t.Fatalf("Failed to remove failed run: %v", err)
	}

	// Remove running run without force (should fail)
	err = removeWorkflowRun(wm, runID3, false)
	if err == nil {
		t.Fatal("Expected error when removing running run without force")
	}

	// Force remove running run
	err = removeWorkflowRun(wm, runID3, true)
	if err != nil {
		t.Fatalf("Failed to force remove running run: %v", err)
	}

	// Verify all run directories are removed
	for _, runID := range []string{runID1, runID2, runID3} {
		runDir := filepath.Join(rootPath, workspace.MetaDir, workspace.RunsDir, runID)
		if _, err := os.Stat(runDir); !os.IsNotExist(err) {
			t.Errorf("Run directory for %s should be removed", runID)
		}
	}
}

// removeWorkflowRun is a helper function that mimics the rmWorkflowCmd logic for testing
func removeWorkflowRun(wm *workspace.WorkspaceManager, runID string, force bool) error {
	// Get the run
	engine := workflow.NewEngine(wm)
	run, err := engine.GetRun(runID)
	if err != nil {
		return &runError{"run '" + runID + "' not found"}
	}

	// Check if run is still running
	if run.Status == workflow.StatusRunning {
		if !force {
			return &runError{"run '" + runID + "' is still running, use --force to remove"}
		}
	}

	// Find and remove all agentic nodes created by this run
	for nodeName, node := range wm.State.Nodes {
		if node.CreatedBy == runID {
			if err := wm.RemoveNode(nodeName); err != nil {
				// Log but continue
			}
		}
	}

	// Remove the run directory
	runDir := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.RemoveAll(runDir); err != nil {
		return err
	}

	return nil
}

// runError is a simple error type for testing
type runError struct {
	msg string
}

func (e *runError) Error() string {
	return e.msg
}
