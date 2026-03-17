package vscode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpdateWorkspaceFile tests creating a new workspace file
func TestUpdateWorkspaceFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "orion-vscode-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node1", "node2"}

	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Get expected file name based on how UpdateWorkspaceFile generates it
	projectName := strings.TrimSuffix(filepath.Base(tmpDir), "_swarm")
	expectedFile := filepath.Join(tmpDir, projectName+".code-workspace")

	// Verify file was created
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Workspace file not created at %s", expectedFile)
	}

	// Verify content
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read workspace file: %v", err)
	}

	var workspace WorkspaceFile
	if err := json.Unmarshal(content, &workspace); err != nil {
		t.Fatalf("Failed to parse workspace file: %v", err)
	}

	// Verify folders
	expectedFolderCount := 1 + len(nodes) // repo + nodes
	if len(workspace.Folders) != expectedFolderCount {
		t.Errorf("Expected %d folders, got %d", expectedFolderCount, len(workspace.Folders))
	}

	// Verify repo folder
	if len(workspace.Folders) > 0 {
		if workspace.Folders[0].Path != repoDir {
			t.Errorf("First folder should be repo, got %s", workspace.Folders[0].Path)
		}
	}

	// Verify node folders
	for i, node := range nodes {
		expectedPath := filepath.Join(nodesDir, node)
		if workspace.Folders[i+1].Path != expectedPath {
			t.Errorf("Node %d folder path mismatch. Expected %s, got %s", i, expectedPath, workspace.Folders[i+1].Path)
		}
	}
}

// TestUpdateWorkspaceFileWithEmptyNodes tests with empty nodes list
func TestUpdateWorkspaceFileWithEmptyNodes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-vscode-empty-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{}

	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Get expected file name
	projectName := strings.TrimSuffix(filepath.Base(tmpDir), "_swarm")
	expectedFile := filepath.Join(tmpDir, projectName+".code-workspace")

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read workspace file: %v", err)
	}

	var workspace WorkspaceFile
	if err := json.Unmarshal(content, &workspace); err != nil {
		t.Fatalf("Failed to parse workspace file: %v", err)
	}

	// Should only have repo folder
	if len(workspace.Folders) != 1 {
		t.Errorf("Expected 1 folder (repo only), got %d", len(workspace.Folders))
	}
}

// TestUpdateWorkspaceFileWithSwarmSuffix tests removing _swarm suffix from project name
func TestUpdateWorkspaceFileWithSwarmSuffix(t *testing.T) {
	// Create temp directory with _swarm suffix
	tmpDir, err := os.MkdirTemp("", "orion-vscode-swarm-test_swarm")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node1"}

	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Verify file name has _swarm removed
	projectName := strings.TrimSuffix(filepath.Base(tmpDir), "_swarm")
	expectedFile := filepath.Join(tmpDir, projectName+".code-workspace")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Workspace file should have _swarm suffix removed, expected %s", expectedFile)
	}
}

// TestUpdateWorkspaceFileWithSingleQuote tests with special characters in node names
func TestUpdateWorkspaceFileWithSpecialChars(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-vscode-special-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node-with-dash", "node_with_underscore"}

	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Get expected file name
	projectName := strings.TrimSuffix(filepath.Base(tmpDir), "_swarm")
	expectedFile := filepath.Join(tmpDir, projectName+".code-workspace")

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read workspace file: %v", err)
	}

	var workspace WorkspaceFile
	if err := json.Unmarshal(content, &workspace); err != nil {
		t.Fatalf("Failed to parse workspace file: %v", err)
	}

	// Verify folders count
	if len(workspace.Folders) != 3 {
		t.Errorf("Expected 3 folders, got %d", len(workspace.Folders))
	}
}

// TestWorkspaceFileJSONFormat tests the JSON format of workspace file
func TestWorkspaceFileJSONFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-vscode-json-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node1"}

	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Get expected file name
	projectName := strings.TrimSuffix(filepath.Base(tmpDir), "_swarm")
	expectedFile := filepath.Join(tmpDir, projectName+".code-workspace")

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read workspace file: %v", err)
	}

	// Verify it's valid JSON with indentation
	var rawJSON interface{}
	if err := json.Unmarshal(content, &rawJSON); err != nil {
		t.Fatalf("Content is not valid JSON: %v", err)
	}

	// Verify indentation (should have 2 spaces based on encoder.SetIndent("", "  "))
	contentStr := string(content)
	if !contains(contentStr, "  \"folders\"") {
		t.Error("Workspace file should be indented with 2 spaces")
	}
}

// TestUpdateWorkspaceFileOverwrite tests overwriting existing workspace file
func TestUpdateWorkspaceFileOverwrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-vscode-overwrite-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"

	// Create initial workspace
	nodes1 := []string{"node1"}
	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes1)
	if err != nil {
		t.Fatalf("First UpdateWorkspaceFile failed: %v", err)
	}

	// Get expected file name
	projectName := strings.TrimSuffix(filepath.Base(tmpDir), "_swarm")
	expectedFile := filepath.Join(tmpDir, projectName+".code-workspace")

	// Read initial content
	content1, _ := os.ReadFile(expectedFile)

	// Update with different nodes
	nodes2 := []string{"node1", "node2", "node3"}
	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes2)
	if err != nil {
		t.Fatalf("Second UpdateWorkspaceFile failed: %v", err)
	}

	// Read updated content
	content2, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read updated workspace file: %v", err)
	}

	// Content should be different
	if string(content1) == string(content2) {
		t.Error("Workspace file content should be different after update")
	}

	// Verify new content has 4 folders (repo + 3 nodes)
	var workspace WorkspaceFile
	if err := json.Unmarshal(content2, &workspace); err != nil {
		t.Fatalf("Failed to parse updated workspace file: %v", err)
	}

	if len(workspace.Folders) != 4 {
		t.Errorf("Expected 4 folders after update, got %d", len(workspace.Folders))
	}
}

// TestFolderStruct tests the Folder struct
func TestFolderStruct(t *testing.T) {
	folder := Folder{
		Path: "test/path",
	}

	if folder.Path != "test/path" {
		t.Errorf("Folder path mismatch: %s", folder.Path)
	}
}

// TestWorkspaceFileStruct tests the WorkspaceFile struct
func TestWorkspaceFileStruct(t *testing.T) {
	workspace := WorkspaceFile{
		Folders: []Folder{
			{Path: "folder1"},
			{Path: "folder2"},
		},
		Settings: map[string]interface{}{
			"editor.tabSize": 4,
		},
	}

	if len(workspace.Folders) != 2 {
		t.Errorf("Expected 2 folders, got %d", len(workspace.Folders))
	}

	if workspace.Settings["editor.tabSize"] != 4 {
		t.Errorf("Settings mismatch")
	}
}

// TestUpdateWorkspaceFileWithNilNodes tests with nil nodes
func TestUpdateWorkspaceFileWithNilNodes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-vscode-nil-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	var nodes []string = nil

	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Get expected file name
	projectName := strings.TrimSuffix(filepath.Base(tmpDir), "_swarm")
	expectedFile := filepath.Join(tmpDir, projectName+".code-workspace")

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read workspace file: %v", err)
	}

	var workspace WorkspaceFile
	if err := json.Unmarshal(content, &workspace); err != nil {
		t.Fatalf("Failed to parse workspace file: %v", err)
	}

	// Should only have repo folder
	if len(workspace.Folders) != 1 {
		t.Errorf("Expected 1 folder (repo only), got %d", len(workspace.Folders))
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
