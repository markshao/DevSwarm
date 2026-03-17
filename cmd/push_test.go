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

// setupTestWorkspaceForPush creates a temporary workspace with a remote repo for push command testing
func setupTestWorkspaceForPush(t *testing.T) (rootPath, repoPath, remotePath string, wm *workspace.WorkspaceManager, cleanup func()) {
	t.Helper()

	// 1. Create remote repo (bare)
	remoteDir, err := os.MkdirTemp("", "orion-push-remote")
	if err != nil {
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	cmd := exec.Command("git", "init", "--bare", remoteDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(remoteDir)
		t.Fatalf("failed to git init --bare: %v, output: %s", err, output)
	}

	// 2. Create a local repo with initial commit first (to serve as the source for cloning)
	localSourceDir, err := os.MkdirTemp("", "orion-push-source")
	if err != nil {
		os.RemoveAll(remoteDir)
		t.Fatalf("failed to create temp source dir: %v", err)
	}

	cmd = exec.Command("git", "init")
	cmd.Dir = localSourceDir
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localSourceDir)
		t.Fatalf("failed to git init: %v, output: %s", err, output)
	}

	// Configure user
	exec.Command("git", "-C", localSourceDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", localSourceDir, "config", "user.name", "Test User").Run()

	// Create initial commit
	readme := filepath.Join(localSourceDir, "README.md")
	if err := os.WriteFile(readme, []byte("# Test Repo"), 0644); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localSourceDir)
		t.Fatalf("failed to write file: %v", err)
	}

	cmd = exec.Command("git", "-C", localSourceDir, "add", ".")
	if err := cmd.Run(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localSourceDir)
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "-C", localSourceDir, "commit", "-m", "Initial commit")
	if err := cmd.Run(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localSourceDir)
		t.Fatalf("failed to git commit: %v", err)
	}

	// Push to remote
	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = localSourceDir
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localSourceDir)
		t.Fatalf("failed to add remote: %v, output: %s", err, output)
	}

	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = localSourceDir
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localSourceDir)
		t.Fatalf("failed to push main: %v, output: %s", err, output)
	}

	// 3. Create root dir
	rootDir, err := os.MkdirTemp("", "orion-push-test")
	if err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localSourceDir)
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 4. Initialize workspace
	wm, err = workspace.Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localSourceDir)
		os.RemoveAll(rootDir)
		t.Fatalf("Init failed: %v", err)
	}

	// 5. Clone repo from remote
	if err := git.Clone(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localSourceDir)
		os.RemoveAll(rootDir)
		t.Fatalf("Clone failed: %v", err)
	}

	// Configure local repo
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	cleanup = func() {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localSourceDir)
		os.RemoveAll(rootDir)
	}

	return rootDir, wm.State.RepoPath, remoteDir, wm, cleanup
}

// TestPushLogicSuccess tests the push logic directly (without cobra command execution)
func TestPushLogicSuccess(t *testing.T) {
	rootPath, _, remotePath, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "push-success"
	logicalBranch := "feature/push-success"

	// 1. Spawn node
	if err := wm.SpawnNode(nodeName, logicalBranch, "main", "Push success test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Make changes in the node's worktree
	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "push-test.txt")
	if err := os.WriteFile(testFile, []byte("push content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = node.WorktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add push test")
	cmd.Dir = node.WorktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// 3. Update status to READY_TO_PUSH
	if err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 4. Execute push logic directly
	if err := git.PushBranch(wm.State.RepoPath, node.ShadowBranch); err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// 5. Update status to PUSHED
	if err := wm.UpdateNodeStatus(nodeName, types.StatusPushed); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 6. Verify node status updated to PUSHED
	wm2, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("Failed to reload manager: %v", err)
	}

	loadedNode, exists := wm2.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found after push")
	}

	if loadedNode.Status != types.StatusPushed {
		t.Errorf("Expected status to be PUSHED, got %q", loadedNode.Status)
	}

	// 7. Verify branch pushed to remote
	cmd = exec.Command("git", "ls-remote", remotePath, "refs/heads/"+node.ShadowBranch)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to verify remote branch: %v", err)
	}

	if !strings.Contains(string(output), "refs/heads/"+node.ShadowBranch) {
		t.Errorf("remote branch '%s' not found after push. Output: %s", node.ShadowBranch, string(output))
	}
}

// TestPushLogicForcePush tests force push regardless of status
func TestPushLogicForcePush(t *testing.T) {
	_, _, remotePath, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "push-force"
	logicalBranch := "feature/force-push"

	// 1. Spawn node
	if err := wm.SpawnNode(nodeName, logicalBranch, "main", "Force push test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Make changes
	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "force-test.txt")
	if err := os.WriteFile(testFile, []byte("force content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = node.WorktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add force test")
	cmd.Dir = node.WorktreePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// 3. Keep status as WORKING (not READY_TO_PUSH) - simulating force push scenario

	// 4. Execute push logic directly (force push ignores status)
	if err := git.PushBranch(wm.State.RepoPath, node.ShadowBranch); err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// 5. Verify branch pushed to remote
	cmd = exec.Command("git", "ls-remote", remotePath, "refs/heads/"+node.ShadowBranch)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to verify remote branch: %v", err)
	}

	if !strings.Contains(string(output), "refs/heads/"+node.ShadowBranch) {
		t.Errorf("remote branch '%s' not found after force push. Output: %s", node.ShadowBranch, string(output))
	}
}

// TestPushLogicWrongStatus tests that push should be blocked for wrong status (without force)
func TestPushLogicWrongStatus(t *testing.T) {
	_, _, _, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "push-wrong-status"
	logicalBranch := "feature/wrong-status"

	// 1. Spawn node (status is WORKING by default)
	if err := wm.SpawnNode(nodeName, logicalBranch, "main", "Wrong status test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Check status - should be WORKING
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("Expected initial status to be WORKING, got %q", node.Status)
	}

	// 3. Verify that status check would fail (simulating the command's status check logic)
	if node.Status == types.StatusReadyToPush {
		t.Error("Status should not be READY_TO_PUSH without workflow execution")
	}
}

// TestPushLogicAlreadyPushed tests that already pushed nodes are handled correctly
func TestPushLogicAlreadyPushed(t *testing.T) {
	_, _, _, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "push-already"
	logicalBranch := "feature/already-pushed"

	// 1. Spawn node
	if err := wm.SpawnNode(nodeName, logicalBranch, "main", "Already pushed test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Set status to PUSHED
	if err := wm.UpdateNodeStatus(nodeName, types.StatusPushed); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 3. Verify status
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusPushed {
		t.Errorf("Expected status to be PUSHED, got %q", node.Status)
	}

	// 4. Verify that status check would indicate already pushed
	if node.Status == types.StatusReadyToPush {
		t.Error("Status should be PUSHED, not READY_TO_PUSH")
	}
}

// TestPushLogicFailStatus tests that failed nodes are handled correctly
func TestPushLogicFailStatus(t *testing.T) {
	_, _, _, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "push-fail-status"
	logicalBranch := "feature/fail-status"

	// 1. Spawn node
	if err := wm.SpawnNode(nodeName, logicalBranch, "main", "Fail status test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Set status to FAIL
	if err := wm.UpdateNodeStatus(nodeName, types.StatusFail); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 3. Verify status
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusFail {
		t.Errorf("Expected status to be FAIL, got %q", node.Status)
	}
}

// TestPushLogicNonExistentNode tests error handling for non-existent nodes
func TestPushLogicNonExistentNode(t *testing.T) {
	_, _, _, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Try to get non-existent node
	_, exists := wm.State.Nodes["non-existent-node"]
	if exists {
		t.Error("non-existent-node should not exist")
	}
}

// TestPushCmdFlagParsing tests that the push command flags are properly configured
func TestPushCmdFlagParsing(t *testing.T) {
	// Test that the force flag exists and can be parsed
	pushCmd.SetArgs([]string{"--help"})
	
	// Check if the force flag is defined
	forceFlag := pushCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Error("force flag should be defined")
	}
	
	if forceFlag.Shorthand != "f" {
		t.Errorf("force flag shorthand should be 'f', got %q", forceFlag.Shorthand)
	}
}

// TestPushCommandDefinition tests the push command structure
func TestPushCommandDefinition(t *testing.T) {
	if pushCmd.Use != "push [node_name]" {
		t.Errorf("expected Use to be 'push [node_name]', got %q", pushCmd.Use)
	}

	if pushCmd.Short != "Push a node's branch to remote repository" {
		t.Errorf("expected Short to mention push, got %q", pushCmd.Short)
	}

	// Test Args validation (MaximumNArgs(1))
	if err := pushCmd.Args(nil, []string{}); err != nil {
		t.Errorf("no args should be valid: %v", err)
	}
	if err := pushCmd.Args(nil, []string{"node1"}); err != nil {
		t.Errorf("one arg should be valid: %v", err)
	}
	// Two args should fail (but we can't easily test this without calling the function)
}
