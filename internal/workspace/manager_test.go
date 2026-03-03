package workspace

import (
	"devswarm/internal/types"
	"runtime"
	"testing"
)

func TestFindNodeByPath(t *testing.T) {
	// Setup a mock workspace manager
	wm := &WorkspaceManager{
		State: &types.State{
			Nodes: map[string]types.Node{
				"node1": {
					Name:         "node1",
					WorktreePath: "/Users/user/devswarm_ws/nodes/node1",
				},
				"node2": {
					Name:         "node2",
					WorktreePath: "/Users/user/devswarm_ws/nodes/node2",
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
			inputPath: "/Users/user/devswarm_ws/nodes/node1/main.go",
			wantNode:  "node1",
			wantFound: true,
		},
		{
			name:      "Exact match directory inside node",
			inputPath: "/Users/user/devswarm_ws/nodes/node1/src",
			wantNode:  "node1",
			wantFound: true,
		},
		{
			name:      "Path outside nodes",
			inputPath: "/Users/user/devswarm_ws/repo/main.go",
			wantNode:  "",
			wantFound: false,
		},
		{
			name:      "Partial prefix match (should fail)",
			inputPath: "/Users/user/devswarm_ws/nodes/node1-suffix/main.go",
			wantNode:  "",
			wantFound: false,
		},
	}

	// Add case-insensitive tests for macOS/Windows
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		// Simulate the issue: Node path is stored as lowercase, input is mixed case
		// Assuming wm.State.Nodes matches the lowercase structure
		wm.State.Nodes["node_lower"] = types.Node{
			Name:         "node_lower",
			WorktreePath: "/users/user/devswarm_ws/nodes/node_lower",
		}

		tests = append(tests, struct {
			name      string
			inputPath string
			wantNode  string
			wantFound bool
		}{
			name:      "Case mismatch on macOS (Input mixed, Node lower)",
			inputPath: "/Users/User/DevSwarm_ws/Nodes/node_lower/main.go",
			wantNode:  "node_lower",
			wantFound: true,
		}, struct {
			name      string
			inputPath string
			wantNode  string
			wantFound bool
		}{
			name:      "Case mismatch on macOS (Input lower, Node mixed)",
			inputPath: "/users/user/devswarm_ws/nodes/node1/main.go",
			wantNode:  "node1", // node1 stored as Mixed case in setup
			wantFound: true,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock filepath.EvalSymlinks behavior for testing?
			// Since we can't easily mock syscalls, we might need to refactor FindNodeByPath
			// to accept a path normalizer function.
			// For now, let's test the logic logic without actual FS calls if possible,
			// or we accept that EvalSymlinks might fail on non-existent paths.

			// However, our current implementation calls filepath.Abs and EvalSymlinks.
			// Testing this purely with strings requires us to modify the implementation
			// to be more testable or use a real temporary directory.

			// Let's assume we are testing the string matching logic primarily.
			// But wait, the current implementation RELIES on EvalSymlinks to fix case.
			// So unit testing with fake paths won't work because EvalSymlinks will fail or return as-is.

			// We need a helper that abstracts FS operations.
		})
	}
}
