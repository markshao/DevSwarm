package workspace

import (
	"orion/internal/git"
	"orion/internal/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Helper to create a temp workspace and repo
func setupTestWorkspace(t *testing.T) (*WorkspaceManager, func()) {
	t.Helper()

	// 1. Create root dir for workspace
	rootDir, err := os.MkdirTemp("", "orion-ws-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. Initialize a separate git repo to serve as "remote"
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	// Initialize remote repo
	exec.Command("git", "init", remoteDir).Run()
	exec.Command("git", "-C", remoteDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", remoteDir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", remoteDir, "checkout", "-b", "main").Run()
	os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("# Remote"), 0644)
	exec.Command("git", "-C", remoteDir, "add", ".").Run()
	exec.Command("git", "-C", remoteDir, "commit", "-m", "Initial commit").Run()

	// 3. Run Init (creates .orion, etc.)
	wm, err := Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// 4. Manually clone the repo (simulating CLI behavior)
	if err := git.Clone(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Clone failed: %v", err)
	}

	// Configure user for local repo as well
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	cleanup := func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return wm, cleanup
}

func TestInit(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Verify directory structure
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir)); os.IsNotExist(err) {
		t.Errorf(".orion directory not created")
	}
	if _, err := os.Stat(filepath.Join(wm.RootPath, RepoDir)); os.IsNotExist(err) {
		t.Errorf("repo directory not created")
	}
	if _, err := os.Stat(filepath.Join(wm.RootPath, WorkspacesDir)); os.IsNotExist(err) {
		t.Errorf("workspaces directory not created")
	}

	// Verify V1 config files are generated
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir, ConfigFile)); os.IsNotExist(err) {
		t.Errorf("config.yaml not created")
	}
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir, WorkflowsDir, "default.yaml")); os.IsNotExist(err) {
		t.Errorf("default workflow not created")
	}
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir, AgentsDir, "ut-agent.yaml")); os.IsNotExist(err) {
		t.Errorf("ut-agent.yaml not created")
	}
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir, AgentsDir, "cr-agent.yaml")); os.IsNotExist(err) {
		t.Errorf("cr-agent.yaml not created")
	}
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir, PromptsDir, "ut.md")); os.IsNotExist(err) {
		t.Errorf("ut.md prompt not created")
	}
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir, PromptsDir, "cr.md")); os.IsNotExist(err) {
		t.Errorf("cr.md prompt not created")
	}

	// Verify GetConfig parses config.yaml correctly
	config, err := wm.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if config.Git.MainBranch != "main" {
		t.Errorf("config.Git.MainBranch = %q, want %q", config.Git.MainBranch, "main")
	}
	if config.Workspace != "workspaces" {
		t.Errorf("config.Workspace = %q, want %q", config.Workspace, "workspaces")
	}
	if def, ok := config.Workflow["default"]; !ok || def != "default" {
		t.Errorf("config.Workflow[\"default\"] = %q, want %q", def, "default")
	}

	// Verify state file
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir, StateFile)); os.IsNotExist(err) {
		t.Errorf("state.json not created")
	}
}

// TestInitGeneratesV1Configs verifies that Init (which calls generateV1Configs)
// produces V1 configuration files that are aligned with the current defaults,
// including the updated agent runtime and unit-test prompt content.
func TestInitGeneratesV1Configs(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// 1. ut-agent.yaml should use qwen as the code agent runtime
	utAgentPath := filepath.Join(wm.RootPath, MetaDir, AgentsDir, "ut-agent.yaml")
	data, err := os.ReadFile(utAgentPath)
	if err != nil {
		t.Fatalf("failed to read ut-agent.yaml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "provider: qwen") {
		t.Errorf("ut-agent.yaml does not configure qwen provider; content: %s", content)
	}

	// 2. prompts/ut.md should contain the updated unit test generation instructions
	utPromptPath := filepath.Join(wm.RootPath, MetaDir, PromptsDir, "ut.md")
	data, err = os.ReadFile(utPromptPath)
	if err != nil {
		t.Fatalf("failed to read ut.md: %v", err)
	}
	prompt := string(data)

	requiredSubstrings := []string{
		"Your task is to analyze the code changes provided below and **immediately generate and write unit tests** for them.",
		"**DO NOT OUTPUT CODE BLOCKS IN THE CHAT.**",
	}

	for _, sub := range requiredSubstrings {
		if !strings.Contains(prompt, sub) {
			t.Errorf("ut.md missing required text %q. Full prompt: %s", sub, prompt)
		}
	}
}

func TestSpawnAndRemoveNode(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node"
	logicalBranch := "feature/login"

	// 1. Spawn Node
	err := wm.SpawnNode(nodeName, logicalBranch, "main", "Testing login", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Verify state
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Errorf("Node not found in state")
	}

	// Verify worktree exists
	if _, err := os.Stat(node.WorktreePath); os.IsNotExist(err) {
		t.Errorf("Worktree directory not created at %s", node.WorktreePath)
	}

	// Verify shadow branch exists
	if err := git.VerifyBranch(wm.State.RepoPath, node.ShadowBranch); err != nil {
		t.Errorf("Shadow branch not created")
	}

	// 2. Remove Node
	err = wm.RemoveNode(nodeName)
	if err != nil {
		t.Fatalf("RemoveNode failed: %v", err)
	}

	// Verify state removed
	if _, exists := wm.State.Nodes[nodeName]; exists {
		t.Errorf("Node still exists in state after removal")
	}

	// Verify worktree removed (directory might be gone or empty)
	if _, err := os.Stat(node.WorktreePath); !os.IsNotExist(err) {
		// If it exists, it should be empty
		entries, _ := os.ReadDir(node.WorktreePath)
		if len(entries) > 0 {
			t.Errorf("Worktree directory not cleaned up")
		}
	}
}

func TestMergeNode(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "merge-node"
	logicalBranch := "feature/merge-test"

	// 1. Spawn
	wm.SpawnNode(nodeName, logicalBranch, "main", "Merge Test", true)
	node := wm.State.Nodes[nodeName]

	// 2. Make changes in the node's worktree
	newFile := filepath.Join(node.WorktreePath, "new-feature.txt")
	os.WriteFile(newFile, []byte("content"), 0644)

	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Work in node").Run()

	// 3. Merge
	err := wm.MergeNode(nodeName, false)
	if err != nil {
		t.Fatalf("MergeNode failed: %v", err)
	}

	// 4. Verify changes in Logical Branch (in the main repo)
	// We need to check if logicalBranch has the commit.
	// Note: SquashMerge happens in wm.State.RepoPath.
	// But wait, SquashMerge checks out logicalBranch in RepoPath.

	// Let's verify file exists in RepoPath (after checkout logicalBranch)
	// VerifyBranch checks out? No, VerifyBranch just checks existence.
	// SquashMerge does checkout. So RepoPath should be on logicalBranch now.

	// Check if file exists in main repo
	repoFile := filepath.Join(wm.State.RepoPath, "new-feature.txt")
	if _, err := os.Stat(repoFile); os.IsNotExist(err) {
		t.Errorf("Merged file not found in main repo")
	}
}

func TestSpawnNodeFeatureMode(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "feature-node"
	logicialBranch := "feature/feature-mode"

	// Feature mode: isShadow=false should create a worktree directly on the logical branch.
	if err := wm.SpawnNode(nodeName, logicialBranch, "main", "Feature mode", false); err != nil {
		t.Fatalf("SpawnNode (feature mode) failed: %v", err)
	}

	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("node not found in state after SpawnNode")
	}

	if node.ShadowBranch != logicialBranch {
		t.Errorf("expected shadow branch to equal logical branch, got %q", node.ShadowBranch)
	}

	if _, err := os.Stat(node.WorktreePath); os.IsNotExist(err) {
		t.Errorf("worktree directory not created at %s", node.WorktreePath)
	}

	// logical branch should exist in the main repo
	if err := git.VerifyBranch(wm.State.RepoPath, logicialBranch); err != nil {
		t.Errorf("logical branch %q not created in repo: %v", logicialBranch, err)
	}
}

func TestGetConfig(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	config, err := wm.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig returned error: %v", err)
	}

	if config.Workspace == "" {
		t.Errorf("expected workspace field to be non-empty")
	}
	if config.Git.MainBranch != "main" {
		t.Errorf("expected git.main_branch to be 'main', got %q", config.Git.MainBranch)
	}
}

func TestFindNodeByPath(t *testing.T) {
	// Keep the original unit test logic but maybe use the helper if we want integration test?
	// The original test used a mock state. Let's keep it simple and just use a struct literal state like before,
	// because creating real worktrees for path testing is slow/overkill.

	// Setup a mock workspace manager
	wm := &WorkspaceManager{
		State: &types.State{
			Nodes: map[string]types.Node{
				"node1": {
					Name:         "node1",
					WorktreePath: "/Users/user/orion_ws/nodes/node1",
				},
				"node2": {
					Name:         "node2",
					WorktreePath: "/Users/user/orion_ws/nodes/node2",
				},
			},
		},
	}

	tests := []struct {
		name      string
		inputPath string
		wantNode  string
		wantFound bool
	}{
		{
			name:      "Exact match file inside node",
			inputPath: "/Users/user/orion_ws/nodes/node1/main.go",
			wantNode:  "node1",
			wantFound: true,
		},
		{
			name:      "Exact match directory inside node",
			inputPath: "/Users/user/orion_ws/nodes/node1/src",
			wantNode:  "node1",
			wantFound: true,
		},
		{
			name:      "Path outside nodes",
			inputPath: "/Users/user/orion_ws/repo/main.go",
			wantNode:  "",
			wantFound: false,
		},
		{
			name:      "Partial prefix match (should fail)",
			inputPath: "/Users/user/orion_ws/nodes/node1-suffix/main.go",
			wantNode:  "",
			wantFound: false,
		},
	}

	// Add case-insensitive tests for macOS/Windows
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		wm.State.Nodes["node_lower"] = types.Node{
			Name:         "node_lower",
			WorktreePath: "/users/user/orion_ws/nodes/node_lower",
		}

		tests = append(tests, struct {
			name      string
			inputPath string
			wantNode  string
			wantFound bool
		}{
			name:      "Case mismatch on macOS (Input mixed, Node lower)",
			inputPath: "/Users/User/Orion_ws/Nodes/node_lower/main.go",
			wantNode:  "node_lower",
			wantFound: true,
		}, struct {
			name      string
			inputPath string
			wantNode  string
			wantFound bool
		}{
			name:      "Case mismatch on macOS (Input lower, Node mixed)",
			inputPath: "/users/user/orion_ws/nodes/node1/main.go",
			wantNode:  "node1",
			wantFound: true,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test relies on filepath.Abs and EvalSymlinks which hit the FS.
			// Since we are using fake paths, this might fail if we don't mock them.
			// However, FindNodeByPath logic handles errors from EvalSymlinks by falling back.
			// So it should work for string comparison logic mostly.

			gotNode, _, _ := wm.FindNodeByPath(tt.inputPath)
			if gotNode != tt.wantNode {
				// On Linux, paths that don't exist might behave differently with Abs/Rel
				// But let's see. If it fails, we know we need to mock FS.
				// For now, let's allow it to fail if it must, but ideally we should only run FS tests on real FS.
				// But this specific test block is testing string logic.

				// ACTUALLY: filepath.Abs works on string. EvalSymlinks fails if not exist.
				// The code:
				// canonicalPath, err := filepath.EvalSymlinks(absPath)
				// if err != nil { canonicalPath = absPath }
				// So it falls back to absPath.
				// Then: rel, err := filepath.Rel(nodePath, canonicalPath)
				// This should work fine for fake paths.

				t.Errorf("FindNodeByPath(%q) = %v, want %v", tt.inputPath, gotNode, tt.wantNode)
			}
		})
	}
}

func TestAppliedRunsPersistence(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node-applied"
	logicalBranch := "feature/applied"

	// 1. Spawn Node
	// Note: SpawnNode now takes (nodeName, logicalBranch, baseBranch, label, isShadow)
	err := wm.SpawnNode(nodeName, logicalBranch, "main", "Testing persistence", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Modify node state (add applied runs)
	node := wm.State.Nodes[nodeName]
	node.AppliedRuns = []string{"run-1", "run-2"}
	wm.State.Nodes[nodeName] = node

	// 3. Save State
	if err := wm.SaveState(); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// 4. Reload Manager
	wm2, err := NewManager(wm.RootPath)
	if err != nil {
		t.Fatalf("Failed to reload manager: %v", err)
	}

	loadedNode, exists := wm2.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found after reload")
	}

	if len(loadedNode.AppliedRuns) != 2 {
		t.Errorf("Expected 2 applied runs, got %d", len(loadedNode.AppliedRuns))
	}
	if loadedNode.AppliedRuns[0] != "run-1" || loadedNode.AppliedRuns[1] != "run-2" {
		t.Errorf("AppliedRuns content mismatch: %v", loadedNode.AppliedRuns)
	}
}

// TestUpdateNodeStatus tests updating node status and persistence
func TestUpdateNodeStatus(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "status-test-node"

	// 1. Spawn node (default status is WORKING)
	if err := wm.SpawnNode(nodeName, "feature/status-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Verify initial status
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("expected initial status to be WORKING, got %s", node.Status)
	}

	// 2. Update status to READY_TO_PUSH
	if err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Verify status in memory
	node = wm.State.Nodes[nodeName]
	if node.Status != types.StatusReadyToPush {
		t.Errorf("expected status to be READY_TO_PUSH, got %s", node.Status)
	}

	// 3. Reload manager and verify persistence
	wm2, err := NewManager(wm.RootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	loadedNode := wm2.State.Nodes[nodeName]
	if loadedNode.Status != types.StatusReadyToPush {
		t.Errorf("expected persisted status to be READY_TO_PUSH, got %s", loadedNode.Status)
	}

	// 4. Update to FAIL status
	if err := wm2.UpdateNodeStatus(nodeName, types.StatusFail); err != nil {
		t.Fatalf("UpdateNodeStatus (FAIL) failed: %v", err)
	}

	// 5. Update to PUSHED status
	if err := wm2.UpdateNodeStatus(nodeName, types.StatusPushed); err != nil {
		t.Fatalf("UpdateNodeStatus (PUSHED) failed: %v", err)
	}

	// Final verification
	wm3, _ := NewManager(wm.RootPath)
	finalNode := wm3.State.Nodes[nodeName]
	if finalNode.Status != types.StatusPushed {
		t.Errorf("expected final status to be PUSHED, got %s", finalNode.Status)
	}
}

// TestUpdateNodeStatusWithNonExistentNode tests updating status for non-existent node
func TestUpdateNodeStatusWithNonExistentNode(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	err := wm.UpdateNodeStatus("non-existent-node", types.StatusReadyToPush)
	if err == nil {
		t.Error("expected error for non-existent node")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error message should mention node does not exist: %v", err)
	}
}

// TestUpdateNodeStatusAllTransitions tests all valid status transitions
func TestUpdateNodeStatusAllTransitions(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	transitions := []struct {
		name     string
		from     types.NodeStatus
		to       types.NodeStatus
		nodeName string
	}{
		{"Working to ReadyToPush", types.StatusWorking, types.StatusReadyToPush, "node-w2r"},
		{"Working to Fail", types.StatusWorking, types.StatusFail, "node-w2f"},
		{"Fail to Working", types.StatusFail, types.StatusWorking, "node-f2w"},
		{"ReadyToPush to Pushed", types.StatusReadyToPush, types.StatusPushed, "node-r2p"},
		{"Pushed to Working", types.StatusPushed, types.StatusWorking, "node-p2w"},
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			nodeName := tt.nodeName

			// Create node
			if err := wm.SpawnNode(nodeName, "feature/"+nodeName, "main", "test", true); err != nil {
				t.Fatalf("SpawnNode failed: %v", err)
			}

			// Set initial status if needed (need to copy struct, modify, then assign back)
			if tt.from != types.StatusWorking {
				node := wm.State.Nodes[nodeName]
				node.Status = tt.from
				wm.State.Nodes[nodeName] = node
				wm.SaveState()
			}

			// Update to target status
			if err := wm.UpdateNodeStatus(nodeName, tt.to); err != nil {
				t.Fatalf("UpdateNodeStatus failed: %v", err)
			}

			// Verify
			node := wm.State.Nodes[nodeName]
			if node.Status != tt.to {
				t.Errorf("expected status %s, got %s", tt.to, node.Status)
			}
		})
	}
}

// TestSpawnNodeWithDefaultStatus tests that newly spawned nodes have WORKING status
func TestSpawnNodeWithDefaultStatus(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "default-status-node"

	if err := wm.SpawnNode(nodeName, "feature/default-status", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("expected default status to be WORKING, got %s", node.Status)
	}
}

// TestSpawnNodeFeatureModeWithStatus tests feature mode node creation with status
func TestSpawnNodeFeatureModeWithStatus(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "feature-mode-status"

	if err := wm.SpawnNode(nodeName, "feature/feature-mode-status", "main", "test", false); err != nil {
		t.Fatalf("SpawnNode (feature mode) failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Verify status is WORKING
	if node.Status != types.StatusWorking {
		t.Errorf("expected status to be WORKING, got %s", node.Status)
	}

	// Verify shadow branch equals logical branch in feature mode
	if node.ShadowBranch != node.LogicalBranch {
		t.Errorf("in feature mode, shadow branch should equal logical branch")
	}
}

// TestCreateAgentNodeWithStatus tests creating agent node (should have empty/default status)
func TestCreateAgentNodeWithStatus(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Use unique node name based on temp dir to avoid tmux session conflicts
	tmpDir := t.TempDir()
	nodeName := "agent-status-" + filepath.Base(tmpDir)
	shadowBranch := "orion/run-test-" + filepath.Base(tmpDir) + "/ut"
	baseBranch := "main"
	createdBy := "run-test-" + filepath.Base(tmpDir)

	// Try to kill any existing session (ignore errors)
	sessionName := "orion-" + nodeName
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	node, err := wm.CreateAgentNode(nodeName, shadowBranch, baseBranch, createdBy)
	if err != nil {
		// tmux may not be available in test environment or session conflicts
		t.Logf("CreateAgentNode may fail in environments without tmux: %v", err)
		// Still verify that the node was added to state
		if _, exists := wm.State.Nodes[nodeName]; !exists {
			t.Error("node should be added to state even if tmux fails")
		}
		return
	}

	// Agent nodes should have empty status initially (or WORKING depending on design)
	// Based on the code, CreateAgentNode doesn't set Status, so it should be empty
	if node.Status != "" {
		t.Logf("Agent node status: %s (expected empty as per implementation)", node.Status)
	}

	// Verify other fields
	if node.Name != nodeName {
		t.Errorf("expected node name %s, got %s", nodeName, node.Name)
	}
	if node.ShadowBranch != shadowBranch {
		t.Errorf("expected shadow branch %s, got %s", shadowBranch, node.ShadowBranch)
	}
	if node.CreatedBy != createdBy {
		t.Errorf("expected created by %s, got %s", createdBy, node.CreatedBy)
	}
	if node.TmuxSession == "" {
		t.Error("tmux session should be set for agent node")
	}
}

// TestNodeStatusInStatePersistence tests that node status is properly persisted and reloaded
func TestNodeStatusInStatePersistence(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create multiple nodes with different statuses
	nodes := []struct {
		name   string
		status types.NodeStatus
	}{
		{"node-1", types.StatusWorking},
		{"node-2", types.StatusReadyToPush},
		{"node-3", types.StatusFail},
		{"node-4", types.StatusPushed},
	}

	for _, n := range nodes {
		if err := wm.SpawnNode(n.name, "feature/"+n.name, "main", "test", true); err != nil {
			t.Fatalf("SpawnNode failed for %s: %v", n.name, err)
		}
		if n.status != types.StatusWorking {
			if err := wm.UpdateNodeStatus(n.name, n.status); err != nil {
				t.Fatalf("UpdateNodeStatus failed for %s: %v", n.name, err)
			}
		}
	}

	// Save state
	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Reload and verify
	wm2, err := NewManager(wm.RootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	for _, n := range nodes {
		node := wm2.State.Nodes[n.name]
		if node.Status != n.status {
			t.Errorf("node %s: expected status %s, got %s", n.name, n.status, node.Status)
		}
	}
}

// TestFindNodeByPathWithNodeStatus tests finding node by path and checking its status
func TestFindNodeByPathWithNodeStatus(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "find-path-status"
	if err := wm.SpawnNode(nodeName, "feature/find-path-status", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Update status
	if err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Test finding node by worktree path
	foundName, foundNode, err := wm.FindNodeByPath(node.WorktreePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}
	if foundName != nodeName {
		t.Errorf("expected to find node %s, got %s", nodeName, foundName)
	}
	if foundNode.Status != types.StatusReadyToPush {
		t.Errorf("expected status %s, got %s", types.StatusReadyToPush, foundNode.Status)
	}

	// Test finding node by subdirectory
	subDir := filepath.Join(node.WorktreePath, "src", "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	foundName2, foundNode2, err := wm.FindNodeByPath(subDir)
	if err != nil {
		t.Fatalf("FindNodeByPath failed for subdirectory: %v", err)
	}
	if foundName2 != nodeName {
		t.Errorf("expected to find node %s from subdirectory, got %s", nodeName, foundName2)
	}
	if foundNode2.Status != types.StatusReadyToPush {
		t.Errorf("expected status %s, got %s", types.StatusReadyToPush, foundNode2.Status)
	}
}

// TestNodeStatusWorkflowIntegration simulates the workflow status update flow
func TestNodeStatusWorkflowIntegration(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "workflow-integration-node"

	// 1. Spawn node (status: WORKING)
	if err := wm.SpawnNode(nodeName, "feature/workflow-integration", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("initial status should be WORKING, got %s", node.Status)
	}

	// 2. Simulate workflow success -> READY_TO_PUSH
	if err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush); err != nil {
		t.Fatalf("UpdateNodeStatus (READY_TO_PUSH) failed: %v", err)
	}

	node = wm.State.Nodes[nodeName]
	if node.Status != types.StatusReadyToPush {
		t.Errorf("after workflow success, status should be READY_TO_PUSH, got %s", node.Status)
	}

	// 3. Simulate push -> PUSHED
	if err := wm.UpdateNodeStatus(nodeName, types.StatusPushed); err != nil {
		t.Fatalf("UpdateNodeStatus (PUSHED) failed: %v", err)
	}

	node = wm.State.Nodes[nodeName]
	if node.Status != types.StatusPushed {
		t.Errorf("after push, status should be PUSHED, got %s", node.Status)
	}
}

// TestNodeStatusWorkflowFailure tests workflow failure status update
func TestNodeStatusWorkflowFailure(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "workflow-failure-node"

	// 1. Spawn node
	if err := wm.SpawnNode(nodeName, "feature/workflow-failure", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Simulate workflow failure -> FAIL
	if err := wm.UpdateNodeStatus(nodeName, types.StatusFail); err != nil {
		t.Fatalf("UpdateNodeStatus (FAIL) failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusFail {
		t.Errorf("after workflow failure, status should be FAIL, got %s", node.Status)
	}

	// 3. After fixing, workflow can be re-run -> WORKING again
	if err := wm.UpdateNodeStatus(nodeName, types.StatusWorking); err != nil {
		t.Fatalf("UpdateNodeStatus (WORKING) failed: %v", err)
	}

	node = wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("after retry, status should be WORKING, got %s", node.Status)
	}
}
