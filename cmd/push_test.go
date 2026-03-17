package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orion/internal/git"
	"orion/internal/types"
	"orion/internal/workspace"
)

// setupTestWorkspaceForPush creates a temporary workspace for push command testing
func setupTestWorkspaceForPush(t *testing.T) (rootPath, repoPath, remotePath string, cleanup func()) {
	t.Helper()

	// 1. Create root dir for workspace
	rootDir, err := os.MkdirTemp("", "orion-push-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. Create remote repo
	remoteDir, err := os.MkdirTemp("", "orion-remote-push-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	// Initialize remote repo as bare repo
	exec.Command("git", "init", "--bare", remoteDir).Run()

	// 3. Initialize workspace
	wm, err := workspace.Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// 4. Clone repo
	if err := git.Clone(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Clone failed: %v", err)
	}

	// Configure local repo
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	// Create initial commit and push to remote
	readmePath := filepath.Join(wm.State.RepoPath, "README.md")
	os.WriteFile(readmePath, []byte("# Test Repo"), 0644)
	exec.Command("git", "-C", wm.State.RepoPath, "add", ".").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "commit", "-m", "Initial commit").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "push", "-u", "origin", "main").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm.State.RepoPath, remoteDir, cleanup
}

// TestPushBranchIntegration tests the push branch functionality at the git level
func TestPushBranchIntegration(t *testing.T) {
	rootPath, repoPath, remotePath, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create a node
	nodeName := "test-push-branch"
	if err := wm.SpawnNode(nodeName, "feature/test-push-branch", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Make changes in the node's worktree
	testFile := filepath.Join(node.WorktreePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Commit changes in the node
	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Test commit").Run()

	// Push the branch using git.PushBranch
	if err := git.PushBranch(repoPath, node.ShadowBranch); err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// Verify branch exists in remote
	output, err := exec.Command("git", "ls-remote", remotePath, node.ShadowBranch).CombinedOutput()
	if err != nil {
		t.Errorf("branch not found in remote: %s", string(output))
	}
	if !strings.Contains(string(output), node.ShadowBranch) {
		t.Errorf("expected branch reference in output: %s", string(output))
	}
}

// TestNodeStatusValidationForPush tests the node status validation logic
func TestNodeStatusValidationForPush(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Test different status scenarios
	tests := []struct {
		name        string
		status      types.NodeStatus
		force       bool
		shouldAllow bool
		reason      string
	}{
		{"ReadyToPush without force", types.StatusReadyToPush, false, true, "should allow push"},
		{"ReadyToPush with force", types.StatusReadyToPush, true, true, "should allow push"},
		{"Working without force", types.StatusWorking, false, false, "should block push"},
		{"Working with force", types.StatusWorking, true, true, "force should allow push"},
		{"Fail without force", types.StatusFail, false, false, "should block push"},
		{"Fail with force", types.StatusFail, true, true, "force should allow push"},
		{"Pushed without force", types.StatusPushed, false, false, "should block push (already pushed)"},
		{"Pushed with force", types.StatusPushed, true, true, "force should allow push"},
		{"Empty status (legacy)", "", false, false, "should block push (treated as WORKING)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test node
			nodeName := "test-" + strings.ToLower(strings.ReplaceAll(tt.name, " ", "-"))
			if err := wm.SpawnNode(nodeName, "feature/"+nodeName, "main", "test", true); err != nil {
				t.Fatalf("SpawnNode failed: %v", err)
			}

			// Set the status
			if tt.status != types.StatusWorking {
				if err := wm.UpdateNodeStatus(nodeName, tt.status); err != nil {
					t.Fatalf("UpdateNodeStatus failed: %v", err)
				}
			}

			// Check status validation logic
			node := wm.State.Nodes[nodeName]
			isAllowed := tt.force || node.Status == types.StatusReadyToPush

			if isAllowed != tt.shouldAllow {
				t.Errorf("status validation mismatch: %s - expected allow=%v, got allow=%v", tt.reason, tt.shouldAllow, isAllowed)
			}
		})
	}
}

// TestUpdateNodeStatusAfterPush tests that node status is correctly updated after push
func TestUpdateNodeStatusAfterPush(t *testing.T) {
	rootPath, repoPath, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-status-after-push"
	if err := wm.SpawnNode(nodeName, "feature/test-status-push", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Make changes and commit
	testFile := filepath.Join(node.WorktreePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Test commit").Run()

	// Set status to READY_TO_PUSH
	if err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Push the branch
	if err := git.PushBranch(repoPath, node.ShadowBranch); err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// Update status to PUSHED (simulating what push command does)
	if err := wm.UpdateNodeStatus(nodeName, types.StatusPushed); err != nil {
		t.Fatalf("UpdateNodeStatus after push failed: %v", err)
	}

	// Verify status
	wm2, _ := workspace.NewManager(rootPath)
	updatedNode := wm2.State.Nodes[nodeName]
	if updatedNode.Status != types.StatusPushed {
		t.Errorf("expected status to be PUSHED, got %s", updatedNode.Status)
	}
}

// TestFindNodeByPathForPush tests finding node by path for push auto-detection
func TestFindNodeByPathForPush(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-find-path-push"
	if err := wm.SpawnNode(nodeName, "feature/test-find-push", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Test auto-detection from worktree path
	detectedName, detectedNode, err := wm.FindNodeByPath(node.WorktreePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}
	if detectedName != nodeName {
		t.Errorf("expected to detect node %s, got %s", nodeName, detectedName)
	}
	if detectedNode.Status != types.StatusWorking {
		t.Errorf("expected status WORKING, got %s", detectedNode.Status)
	}

	// Test auto-detection from subdirectory
	subDir := filepath.Join(node.WorktreePath, "src", "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	detectedName2, detectedNode2, err := wm.FindNodeByPath(subDir)
	if err != nil {
		t.Fatalf("FindNodeByPath failed for subdirectory: %v", err)
	}
	if detectedName2 != nodeName {
		t.Errorf("expected to detect node %s from subdirectory, got %s", nodeName, detectedName2)
	}
	if detectedNode2.Status != types.StatusWorking {
		t.Errorf("expected status WORKING, got %s", detectedNode2.Status)
	}
}

// TestPushCommandFlagParsing tests the push command flag parsing
func TestPushCommandFlagParsing(t *testing.T) {
	// Test that --force flag is properly defined
	forceFlag := pushCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Error("push command should have --force flag")
	}
	if forceFlag.Shorthand != "f" {
		t.Errorf("expected --force shorthand to be 'f', got %s", forceFlag.Shorthand)
	}
	if forceFlag.DefValue != "false" {
		t.Errorf("expected --force default value to be 'false', got %s", forceFlag.DefValue)
	}
}

// TestPushCommandUsage tests the push command usage and description
func TestPushCommandUsage(t *testing.T) {
	if pushCmd.Use == "" {
		t.Error("push command should have Use defined")
	}
	if pushCmd.Short == "" {
		t.Error("push command should have Short description")
	}
	if pushCmd.Long == "" {
		t.Error("push command should have Long description")
	}

	// Check that usage mentions node status
	if !strings.Contains(pushCmd.Long, "READY_TO_PUSH") {
		t.Error("push command Long description should mention READY_TO_PUSH status")
	}
}
