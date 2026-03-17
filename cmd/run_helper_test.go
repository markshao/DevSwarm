package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orion/internal/git"
	"orion/internal/workspace"
)

// setupTestWorkspaceForWorkflow creates a temporary workspace for workflow command testing
func setupTestWorkspaceForWorkflow(t *testing.T) (rootPath string, wm *workspace.WorkspaceManager, cleanup func()) {
	t.Helper()

	rootDir, err := os.MkdirTemp("", "orion-workflow-cmd-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

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

	// Initialize workspace
	wm, err = workspace.Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// Clone repo
	if err := git.Clone(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Clone failed: %v", err)
	}

	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm, cleanup
}

// TestGetRunWorktreePathMainRepo tests GetRunWorktreePath for main repo
func TestGetRunWorktreePathMainRepo(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	path, err := GetRunWorktreePath(rootPath, "")
	if err != nil {
		t.Fatalf("GetRunWorktreePath failed: %v", err)
	}

	expectedPath := filepath.Join(rootPath, workspace.RepoDir)
	if path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, path)
	}
}

// TestGetRunWorktreePathWithNode tests GetRunWorktreePath for a node
func TestGetRunWorktreePathWithNode(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	wm, _ := workspace.NewManager(rootPath)
	nodeName := "test-node"
	if err := wm.SpawnNode(nodeName, "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	path, err := GetRunWorktreePath(rootPath, nodeName)
	if err != nil {
		t.Fatalf("GetRunWorktreePath failed: %v", err)
	}

	expectedPath := wm.State.Nodes[nodeName].WorktreePath
	if path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, path)
	}
}

// TestExecuteInWorktreeNoCommand tests ExecuteInWorktree with no command
func TestExecuteInWorktreeNoCommand(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	_, exitCode, err := ExecuteInWorktree(rootPath, "", []string{})
	if err == nil {
		t.Error("expected error for empty command")
	}
	if exitCode != -1 {
		t.Errorf("expected exit code -1, got %d", exitCode)
	}
}

// TestExecuteInWorktreeWithInvalidCommand tests ExecuteInWorktree with invalid command
func TestExecuteInWorktreeWithInvalidCommand(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	_, _, err := ExecuteInWorktree(rootPath, "", []string{"non_existent_command_xyz123"})
	if err == nil {
		t.Error("expected error for non-existent command")
	}
}

// TestDetermineExecDirInsideMainRepo tests determineExecDir when inside main repo
func TestDetermineExecDirInsideMainRepo(t *testing.T) {
	rootPath, repoPath, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	wm, _ := workspace.NewManager(rootPath)

	execDir, worktree, err := determineExecDir(wm, repoPath, "")
	if err != nil {
		t.Fatalf("determineExecDir failed: %v", err)
	}

	if execDir != repoPath {
		t.Errorf("expected execDir %s, got %s", repoPath, execDir)
	}
	if worktree != "" {
		t.Errorf("expected empty worktree, got %s", worktree)
	}
}

// TestDetermineExecDirWithTargetWorktree tests determineExecDir with target worktree
func TestDetermineExecDirWithTargetWorktree(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	wm, _ := workspace.NewManager(rootPath)
	nodeName := "target-node"
	if err := wm.SpawnNode(nodeName, "feature/target", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	nodePath := wm.State.Nodes[nodeName].WorktreePath

	execDir, worktree, err := determineExecDir(wm, rootPath, nodeName)
	if err != nil {
		t.Fatalf("determineExecDir failed: %v", err)
	}

	// Compare with resolved paths
	execDirResolved, _ := filepath.EvalSymlinks(execDir)
	nodePathResolved, _ := filepath.EvalSymlinks(nodePath)
	if execDirResolved != nodePathResolved {
		t.Errorf("expected execDir %s, got %s", nodePath, execDir)
	}
	if worktree != nodeName {
		t.Errorf("expected worktree %s, got %s", nodeName, worktree)
	}
}

// TestDetermineExecDirWithInvalidWorktree tests determineExecDir with invalid worktree
func TestDetermineExecDirWithInvalidWorktree(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	wm, _ := workspace.NewManager(rootPath)

	_, _, err := determineExecDir(wm, rootPath, "non-existent-node")
	if err == nil {
		t.Error("expected error for non-existent worktree")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestDetermineExecDirStaysInNodeSubdir tests determineExecDir stays in node subdirectory
func TestDetermineExecDirStaysInNodeSubdir(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	wm, _ := workspace.NewManager(rootPath)
	nodeName := "subdir-node"
	if err := wm.SpawnNode(nodeName, "feature/subdir", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]
	subDir := filepath.Join(node.WorktreePath, "src", "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	execDir, worktree, err := determineExecDir(wm, subDir, nodeName)
	if err != nil {
		t.Fatalf("determineExecDir failed: %v", err)
	}

	// Should stay in the subdirectory
	execDirResolved, _ := filepath.EvalSymlinks(execDir)
	subDirResolved, _ := filepath.EvalSymlinks(subDir)
	if execDirResolved != subDirResolved {
		t.Errorf("expected to stay in subdir %s, got %s", subDir, execDir)
	}
	if worktree != nodeName {
		t.Errorf("expected worktree %s, got %s", nodeName, worktree)
	}
}

// TestIsSubDir tests the isSubDir helper function
func TestIsSubDir(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{
			name:   "same directory",
			parent: "/tmp/test",
			child:  "/tmp/test",
			want:   true,
		},
		{
			name:   "direct subdirectory",
			parent: "/tmp/test",
			child:  "/tmp/test/sub",
			want:   true,
		},
		{
			name:   "nested subdirectory",
			parent: "/tmp/test",
			child:  "/tmp/test/a/b/c",
			want:   true,
		},
		{
			name:   "outside parent",
			parent: "/tmp/test",
			child:  "/tmp/other",
			want:   false,
		},
		{
			name:   "sibling directory",
			parent: "/tmp/test",
			child:  "/tmp/test-sibling",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubDir(tt.parent, tt.child)
			if got != tt.want {
				t.Errorf("isSubDir(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
			}
		})
	}
}
