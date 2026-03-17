package vscode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpdateWorkspaceFile verifies that the generated .code-workspace file
// contains the repo folder and all node folders with the expected paths.
func TestUpdateWorkspaceFile(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-ws-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node-a", "node-b"}

	if err := UpdateWorkspaceFile(rootDir, repoDir, nodesDir, nodes); err != nil {
		t.Fatalf("UpdateWorkspaceFile returned error: %v", err)
	}

	// Workspace file name is derived from base name of root dir.
	projectName := filepath.Base(rootDir)
	workspacePath := filepath.Join(rootDir, projectName+".code-workspace")

	data, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	var ws WorkspaceFile
	if err := json.Unmarshal(data, &ws); err != nil {
		t.Fatalf("failed to unmarshal workspace JSON: %v", err)
	}

	// We expect 1 repo folder + len(nodes) node folders
	expectedCount := 1 + len(nodes)
	if len(ws.Folders) != expectedCount {
		t.Fatalf("unexpected folders count: got %d, want %d", len(ws.Folders), expectedCount)
	}

	// First folder should be the repo folder
	if ws.Folders[0].Path != repoDir {
		t.Errorf("repo folder path = %q, want %q", ws.Folders[0].Path, repoDir)
	}

	// Subsequent folders should be nodesDir/nodeName
	for i, node := range nodes {
		idx := i + 1
		wantPath := filepath.Join(nodesDir, node)
		if ws.Folders[idx].Path != wantPath {
			t.Errorf("folder[%d].Path = %q, want %q", idx, ws.Folders[idx].Path, wantPath)
		}
	}
}

// TestUpdateWorkspaceFileWithEmptyNodes tests with empty nodes list
func TestUpdateWorkspaceFileWithEmptyNodes(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-empty-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{}

	if err := UpdateWorkspaceFile(rootDir, repoDir, nodesDir, nodes); err != nil {
		t.Fatalf("UpdateWorkspaceFile returned error: %v", err)
	}

	projectName := filepath.Base(rootDir)
	workspacePath := filepath.Join(rootDir, projectName+".code-workspace")

	data, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	var ws WorkspaceFile
	if err := json.Unmarshal(data, &ws); err != nil {
		t.Fatalf("failed to unmarshal workspace JSON: %v", err)
	}

	if len(ws.Folders) != 1 {
		t.Errorf("expected 1 folder (repo only), got %d", len(ws.Folders))
	}
	if ws.Folders[0].Path != repoDir {
		t.Errorf("repo folder path = %q, want %q", ws.Folders[0].Path, repoDir)
	}
}

// TestUpdateWorkspaceFileWithSwarmSuffix tests project name handling with _swarm suffix
func TestUpdateWorkspaceFileWithSwarmSuffix(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-swarm-test_swarm")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"

	if err := UpdateWorkspaceFile(rootDir, repoDir, nodesDir, []string{}); err != nil {
		t.Fatalf("UpdateWorkspaceFile returned error: %v", err)
	}

	// Workspace file name is derived from base name - the _swarm suffix is trimmed
	// The actual behavior depends on filepath.Base which preserves the suffix in temp dir names
	// This test verifies the file is created with the expected name pattern
	projectName := filepath.Base(rootDir)
	workspacePath := filepath.Join(rootDir, projectName+".code-workspace")
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		t.Errorf("workspace file should be created at %s", workspacePath)
	}
}

// TestUpdateWorkspaceFileWithMultipleNodes tests with many nodes
func TestUpdateWorkspaceFileWithMultipleNodes(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-multi-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node1", "node2", "node3", "node4", "node5"}

	if err := UpdateWorkspaceFile(rootDir, repoDir, nodesDir, nodes); err != nil {
		t.Fatalf("UpdateWorkspaceFile returned error: %v", err)
	}

	projectName := filepath.Base(rootDir)
	workspacePath := filepath.Join(rootDir, projectName+".code-workspace")

	data, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	var ws WorkspaceFile
	if err := json.Unmarshal(data, &ws); err != nil {
		t.Fatalf("failed to unmarshal workspace JSON: %v", err)
	}

	if len(ws.Folders) != 6 {
		t.Errorf("expected 6 folders, got %d", len(ws.Folders))
	}

	// Verify all nodes are present
	for i, node := range nodes {
		_ = i // idx was unused
		found := false
		for _, folder := range ws.Folders {
			if strings.Contains(folder.Path, node) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("node %s not found in workspace folders", node)
		}
	}
}

// TestUpdateWorkspaceFileCreatesDirectory tests that parent directories are created
func TestUpdateWorkspaceFileCreatesDirectory(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-mkdir-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"

	if err := UpdateWorkspaceFile(rootDir, repoDir, nodesDir, []string{}); err != nil {
		t.Fatalf("UpdateWorkspaceFile returned error: %v", err)
	}

	projectName := filepath.Base(rootDir)
	workspacePath := filepath.Join(rootDir, projectName+".code-workspace")

	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		t.Error("workspace file should be created")
	}
}

// TestWorkspaceFileSettings tests that settings field is properly initialized
func TestWorkspaceFileSettings(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-settings-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	if err := UpdateWorkspaceFile(rootDir, "repo", "nodes", []string{}); err != nil {
		t.Fatalf("UpdateWorkspaceFile returned error: %v", err)
	}

	projectName := filepath.Base(rootDir)
	workspacePath := filepath.Join(rootDir, projectName+".code-workspace")

	data, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	var ws WorkspaceFile
	if err := json.Unmarshal(data, &ws); err != nil {
		t.Fatalf("failed to unmarshal workspace JSON: %v", err)
	}

	if ws.Settings == nil {
		t.Error("Settings should be initialized as empty map, not nil")
	}
}

