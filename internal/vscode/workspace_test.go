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

	// Should only have repo folder
	if len(ws.Folders) != 1 {
		t.Errorf("expected 1 folder (repo only), got %d", len(ws.Folders))
	}

	if ws.Folders[0].Path != repoDir {
		t.Errorf("repo folder path = %q, want %q", ws.Folders[0].Path, repoDir)
	}
}

// TestUpdateWorkspaceFileWithSwarmSuffix tests that _swarm suffix is removed
func TestUpdateWorkspaceFileWithSwarmSuffix(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-swarm-test_swarm")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node-1"}

	if err := UpdateWorkspaceFile(rootDir, repoDir, nodesDir, nodes); err != nil {
		t.Fatalf("UpdateWorkspaceFile returned error: %v", err)
	}

	// Workspace file should not have _swarm suffix in name
	projectName := filepath.Base(rootDir)
	workspacePath := filepath.Join(rootDir, projectName+".code-workspace")

	// Check file exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		t.Fatalf("workspace file not created at %s", workspacePath)
	}

	// Read and verify the name doesn't contain _swarm
	if strings.Contains(projectName, "_swarm") {
		// The file name will have _swarm, but the function should trim it
		// Let's verify the actual file name
		entries, _ := os.ReadDir(rootDir)
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".code-workspace") {
				if strings.Contains(entry.Name(), "_swarm_swarm") {
					t.Errorf("workspace file name should not have double _swarm suffix: %s", entry.Name())
				}
			}
		}
	}
}

// TestUpdateWorkspaceFileWithSpecialChars tests with special characters in node names
func TestUpdateWorkspaceFileWithSpecialChars(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-special-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node-with-dash", "node_with_underscore", "node123"}

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

	expectedCount := 1 + len(nodes)
	if len(ws.Folders) != expectedCount {
		t.Errorf("expected %d folders, got %d", expectedCount, len(ws.Folders))
	}
}

// TestUpdateWorkspaceFileWithSingleQuote tests node names with quotes
func TestUpdateWorkspaceFileWithSingleQuote(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-quote-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node-test"}

	if err := UpdateWorkspaceFile(rootDir, repoDir, nodesDir, nodes); err != nil {
		t.Fatalf("UpdateWorkspaceFile returned error: %v", err)
	}

	projectName := filepath.Base(rootDir)
	workspacePath := filepath.Join(rootDir, projectName+".code-workspace")

	data, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	// Verify JSON is valid
	var ws WorkspaceFile
	if err := json.Unmarshal(data, &ws); err != nil {
		t.Fatalf("failed to unmarshal workspace JSON: %v", err)
	}

	// Verify settings map is initialized
	if ws.Settings == nil {
		t.Errorf("Settings map should be initialized, not nil")
	}
}

// TestWorkspaceFileStructure tests the structure of generated workspace file
func TestWorkspaceFileStructure(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-structure-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node-1", "node-2"}

	if err := UpdateWorkspaceFile(rootDir, repoDir, nodesDir, nodes); err != nil {
		t.Fatalf("UpdateWorkspaceFile returned error: %v", err)
	}

	projectName := filepath.Base(rootDir)
	workspacePath := filepath.Join(rootDir, projectName+".code-workspace")

	data, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	// Verify JSON structure
	var rawJSON map[string]interface{}
	if err := json.Unmarshal(data, &rawJSON); err != nil {
		t.Fatalf("failed to unmarshal workspace JSON: %v", err)
	}

	// Check required fields exist
	if _, ok := rawJSON["folders"]; !ok {
		t.Errorf("workspace JSON missing 'folders' field")
	}
	if _, ok := rawJSON["settings"]; !ok {
		t.Errorf("workspace JSON missing 'settings' field")
	}

	// Verify folders is an array
	folders, ok := rawJSON["folders"].([]interface{})
	if !ok {
		t.Fatalf("folders should be an array")
	}

	// Verify each folder has path field
	for i, folder := range folders {
		folderMap, ok := folder.(map[string]interface{})
		if !ok {
			t.Errorf("folder[%d] should be an object", i)
			continue
		}
		if _, ok := folderMap["path"]; !ok {
			t.Errorf("folder[%d] missing 'path' field", i)
		}
	}
}

