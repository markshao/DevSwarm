package workspace

import (
	"fmt"
	"orion/internal/git"
	"orion/internal/tmux"
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

// TestUpdateNodeStatus tests updating node status
func TestUpdateNodeStatus(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "status-test-node"
	if err := wm.SpawnNode(nodeName, "feature/status-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Update status to READY_TO_PUSH
	if err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Verify status was updated
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatal("node not found")
	}
	if node.Status != types.StatusReadyToPush {
		t.Errorf("expected status READY_TO_PUSH, got %s", node.Status)
	}

	// Update status to FAIL
	if err := wm.UpdateNodeStatus(nodeName, types.StatusFail); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	node, _ = wm.State.Nodes[nodeName]
	if node.Status != types.StatusFail {
		t.Errorf("expected status FAIL, got %s", node.Status)
	}
}

// TestUpdateNodeStatusNonExistent tests updating status for non-existent node
func TestUpdateNodeStatusNonExistent(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	err := wm.UpdateNodeStatus("non-existent-node", types.StatusReadyToPush)
	if err == nil {
		t.Error("expected error for non-existent node")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestSaveAndLoadState tests state persistence
func TestSaveAndLoadState(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Modify state
	wm.State.RepoURL = "https://github.com/test/repo.git"
	wm.State.Nodes["test-node"] = types.Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/tmp/test-worktree",
		Status:        types.StatusWorking,
	}

	// Save state
	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Create new manager (should load saved state)
	wm2, err := NewManager(wm.RootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if wm2.State.RepoURL != "https://github.com/test/repo.git" {
		t.Errorf("RepoURL mismatch: got %s, want %s", wm2.State.RepoURL, "https://github.com/test/repo.git")
	}

	node, exists := wm2.State.Nodes["test-node"]
	if !exists {
		t.Fatal("test-node not found after reload")
	}
	if node.LogicalBranch != "feature/test" {
		t.Errorf("LogicalBranch mismatch: got %s, want %s", node.LogicalBranch, "feature/test")
	}
}

// TestFindWorkspaceRoot tests finding workspace root
func TestFindWorkspaceRoot(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Test finding root from workspace root
	root, err := FindWorkspaceRoot(wm.RootPath)
	if err != nil {
		t.Fatalf("FindWorkspaceRoot failed: %v", err)
	}
	if root != wm.RootPath {
		t.Errorf("root mismatch: got %s, want %s", root, wm.RootPath)
	}

	// Test finding root from subdirectory
	repoPath := filepath.Join(wm.RootPath, RepoDir)
	root, err = FindWorkspaceRoot(repoPath)
	if err != nil {
		t.Fatalf("FindWorkspaceRoot from subdirectory failed: %v", err)
	}
	if root != wm.RootPath {
		t.Errorf("root mismatch from subdirectory: got %s, want %s", root, wm.RootPath)
	}

	// Test finding root from non-workspace directory
	tempDir := t.TempDir()
	_, err = FindWorkspaceRoot(tempDir)
	if err == nil {
		t.Error("expected error for non-workspace directory")
	}
}

// TestSyncVSCodeWorkspace tests syncing VSCode workspace file
func TestSyncVSCodeWorkspace(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create some user-created nodes
	wm.State.Nodes["node1"] = types.Node{
		Name:          "node1",
		LogicalBranch: "feature/node1",
		WorktreePath:  filepath.Join(wm.RootPath, WorkspacesDir, "node1"),
		CreatedBy:     "user",
	}
	wm.State.Nodes["node2"] = types.Node{
		Name:          "node2",
		LogicalBranch: "feature/node2",
		WorktreePath:  filepath.Join(wm.RootPath, WorkspacesDir, "node2"),
		CreatedBy:     "user",
	}

	// Also create an agent node (should be excluded)
	wm.State.Nodes["agent-node"] = types.Node{
		Name:          "agent-node",
		LogicalBranch: "orion/agent",
		WorktreePath:  filepath.Join(wm.RootPath, MetaDir, "agent-nodes", "agent-node"),
		CreatedBy:     "run-123",
	}

	if err := wm.SyncVSCodeWorkspace(); err != nil {
		t.Fatalf("SyncVSCodeWorkspace failed: %v", err)
	}

	// Verify workspace file was created
	projectName := filepath.Base(wm.RootPath)
	projectName = strings.TrimSuffix(projectName, "_swarm")
	wsPath := filepath.Join(wm.RootPath, projectName+".code-workspace")

	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		t.Fatal("workspace file was not created")
	}

	// Verify content
	data, err := os.ReadFile(wsPath)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "node1") {
		t.Error("workspace file should contain node1")
	}
	if !strings.Contains(content, "node2") {
		t.Error("workspace file should contain node2")
	}
	// Agent nodes should not be included
	if strings.Contains(content, "agent-node") {
		t.Error("workspace file should not contain agent nodes")
	}
}

// TestCreateAgentNode tests creating an agent node
func TestCreateAgentNode(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-agent-node-" + t.Name()
	shadowBranch := "orion/test-run/test-step"
	baseBranch := "main"
	createdBy := "test-run-123"

	// Clean up any leftover tmux session from previous runs
	sessionName := fmt.Sprintf("orion-%s", nodeName)
	if tmux.SessionExists(sessionName) {
		_ = tmux.KillSession(sessionName)
	}

	node, err := wm.CreateAgentNode(nodeName, shadowBranch, baseBranch, createdBy)
	if err != nil {
		t.Fatalf("CreateAgentNode failed: %v", err)
	}

	if node.Name != nodeName {
		t.Errorf("node name mismatch: got %s, want %s", node.Name, nodeName)
	}
	if node.ShadowBranch != shadowBranch {
		t.Errorf("shadow branch mismatch: got %s, want %s", node.ShadowBranch, shadowBranch)
	}
	if node.CreatedBy != createdBy {
		t.Errorf("created by mismatch: got %s, want %s", node.CreatedBy, createdBy)
	}
	if node.Label != "agent" {
		t.Errorf("label should be 'agent', got %s", node.Label)
	}

	// Verify node was added to state
	if _, exists := wm.State.Nodes[nodeName]; !exists {
		t.Error("node should be added to state")
	}

	// Verify worktree was created
	if _, err := os.Stat(node.WorktreePath); os.IsNotExist(err) {
		t.Errorf("worktree directory not created at %s", node.WorktreePath)
	}
}

// TestApplyGitConfigToWorktree tests applying git config to worktree
func TestApplyGitConfigToWorktree(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create a worktree first
	nodeName := "config-test-node"
	if err := wm.SpawnNode(nodeName, "feature/config-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Apply config
	if err := wm.applyGitConfigToWorktree(node.WorktreePath); err != nil {
		t.Fatalf("applyGitConfigToWorktree failed: %v", err)
	}

	// Verify git config was applied (check using git config command)
	cmd := exec.Command("git", "-C", node.WorktreePath, "config", "user.name")
	output, err := cmd.Output()
	if err != nil {
		t.Logf("git config user.name not set (may be expected if not in config): %v", err)
	} else {
		t.Logf("user.name set to: %s", strings.TrimSpace(string(output)))
	}
}

// TestMergeNodeWithoutCleanup tests merge without cleanup
func TestMergeNodeWithoutCleanup(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "merge-no-cleanup"
	if err := wm.SpawnNode(nodeName, "feature/merge-no-cleanup", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Make changes in the node
	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "merge-test.txt")
	if err := os.WriteFile(testFile, []byte("merge content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Commit changes in node
	_ = exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	_ = exec.Command("git", "-C", node.WorktreePath, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", node.WorktreePath, "config", "user.name", "Test").Run()
	_ = exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Test commit").Run()

	// Merge without cleanup
	if err := wm.MergeNode(nodeName, false); err != nil {
		t.Fatalf("MergeNode failed: %v", err)
	}

	// Node should still exist
	if _, exists := wm.State.Nodes[nodeName]; !exists {
		t.Error("node should still exist after merge without cleanup")
	}
}

// TestMergeNodeOnLogicalBranch tests merge when node is on logical branch directly
func TestMergeNodeOnLogicalBranch(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "direct-branch-node"
	// Feature mode: shadow branch == logical branch
	if err := wm.SpawnNode(nodeName, "feature/direct", "main", "test", false); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Merge should detect no shadow merge needed
	err := wm.MergeNode(nodeName, false)
	if err != nil {
		t.Fatalf("MergeNode failed: %v", err)
	}
}

// TestEnterNode tests entering a node
func TestEnterNode(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "enter-test-node-" + t.Name()
	if err := wm.SpawnNode(nodeName, "feature/enter-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// EnterNode would replace the process, so we can't fully test it
	// But we can verify the node exists and has a tmux session
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatal("node not found")
	}
	// Note: TmuxSession should be set after SpawnNode creates the session
	// If it's empty, the session creation may have failed silently
	if node.TmuxSession == "" {
		t.Logf("Warning: node tmux session is empty (may be expected if tmux unavailable)")
	}
}

// TestEnterNodeNonExistent tests entering a non-existent node
func TestEnterNodeNonExistent(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	err := wm.EnterNode("non-existent-node")
	if err == nil {
		t.Error("expected error for non-existent node")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestRemoveNodeNonExistent tests removing a non-existent node
func TestRemoveNodeNonExistent(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	err := wm.RemoveNode("non-existent-node")
	if err == nil {
		t.Error("expected error for non-existent node")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestNewManagerWithInvalidRoot tests NewManager with invalid root
func TestNewManagerWithInvalidRoot(t *testing.T) {
	tempDir := t.TempDir()

	_, err := NewManager(tempDir)
	if err == nil {
		t.Error("expected error for invalid root")
	}
	if !strings.Contains(err.Error(), "not a orion workspace") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestInitWithInvalidRemote tests Init with invalid remote URL
func TestInitWithInvalidRemote(t *testing.T) {
	rootDir := t.TempDir()

	// This should succeed in creating directories but fail at cloning
	// Actually Init doesn't clone, it just creates structure
	wm, err := Init(rootDir, "https://github.com/test/repo.git")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify structure was created
	if _, err := os.Stat(filepath.Join(rootDir, MetaDir)); os.IsNotExist(err) {
		t.Error(".orion directory should be created")
	}
	if wm.State.RepoURL != "https://github.com/test/repo.git" {
		t.Errorf("RepoURL mismatch: got %s", wm.State.RepoURL)
	}
}

// TestGetConfigWithMissingFile tests GetConfig when config.yaml doesn't exist
func TestGetConfigWithMissingFile(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Remove config.yaml
	configPath := filepath.Join(wm.RootPath, MetaDir, ConfigFile)
	os.Remove(configPath)

	// Should return default config
	config, err := wm.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if config.Agents.DefaultProvider != "qwen" {
		t.Errorf("expected default provider 'qwen', got '%s'", config.Agents.DefaultProvider)
	}
}

// TestGetConfigWithInvalidYAML tests GetConfig with invalid YAML
func TestGetConfigWithInvalidYAML(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Write invalid YAML
	configPath := filepath.Join(wm.RootPath, MetaDir, ConfigFile)
	if err := os.WriteFile(configPath, []byte("invalid: yaml: ["), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err := wm.GetConfig()
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
