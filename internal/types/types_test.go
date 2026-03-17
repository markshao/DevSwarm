package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNodeStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{
			name:     "StatusWorking",
			status:   StatusWorking,
			expected: "WORKING",
		},
		{
			name:     "StatusReadyToPush",
			status:   StatusReadyToPush,
			expected: "READY_TO_PUSH",
		},
		{
			name:     "StatusFail",
			status:   StatusFail,
			expected: "FAIL",
		},
		{
			name:     "StatusPushed",
			status:   StatusPushed,
			expected: "PUSHED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.status)
			}
		})
	}
}

func TestNodeJSONSerialization(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		node     Node
		validate func(t *testing.T, decoded Node)
	}{
		{
			name: "Node with all fields",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				BaseBranch:    "main",
				ShadowBranch:  "orion/test-123/feature/test",
				WorktreePath:  "/path/to/worktree",
				TmuxSession:   "test-session",
				Label:         "testing",
				CreatedBy:     "user",
				AppliedRuns:   []string{"run-1", "run-2"},
				Status:        StatusReadyToPush,
				CreatedAt:     now,
			},
			validate: func(t *testing.T, decoded Node) {
				if decoded.Name != "test-node" {
					t.Errorf("Name mismatch: got %q", decoded.Name)
				}
				if decoded.Status != StatusReadyToPush {
					t.Errorf("Status mismatch: got %q", decoded.Status)
				}
				if len(decoded.AppliedRuns) != 2 {
					t.Errorf("AppliedRuns length mismatch: got %d", len(decoded.AppliedRuns))
				}
			},
		},
		{
			name: "Node with minimal fields",
			node: Node{
				Name:          "minimal-node",
				LogicalBranch: "feature/minimal",
				ShadowBranch:  "orion/min-123/feature/minimal",
				WorktreePath:  "/path/to/minimal",
				CreatedAt:     now,
				// Status is empty (zero value)
			},
			validate: func(t *testing.T, decoded Node) {
				if decoded.Name != "minimal-node" {
					t.Errorf("Name mismatch: got %q", decoded.Name)
				}
				if decoded.Status != "" {
					t.Errorf("Expected empty status, got %q", decoded.Status)
				}
				if decoded.AppliedRuns != nil {
					t.Errorf("Expected nil AppliedRuns, got %v", decoded.AppliedRuns)
				}
			},
		},
		{
			name: "Node with StatusWorking",
			node: Node{
				Name:          "working-node",
				LogicalBranch: "feature/working",
				ShadowBranch:  "orion/work-123/feature/working",
				WorktreePath:  "/path/to/working",
				Status:        StatusWorking,
				CreatedAt:     now,
			},
			validate: func(t *testing.T, decoded Node) {
				if decoded.Status != StatusWorking {
					t.Errorf("Status mismatch: got %q", decoded.Status)
				}
			},
		},
		{
			name: "Node with StatusFail",
			node: Node{
				Name:          "fail-node",
				LogicalBranch: "feature/fail",
				ShadowBranch:  "orion/fail-123/feature/fail",
				WorktreePath:  "/path/to/fail",
				Status:        StatusFail,
				CreatedAt:     now,
			},
			validate: func(t *testing.T, decoded Node) {
				if decoded.Status != StatusFail {
					t.Errorf("Status mismatch: got %q", decoded.Status)
				}
			},
		},
		{
			name: "Node with StatusPushed",
			node: Node{
				Name:          "pushed-node",
				LogicalBranch: "feature/pushed",
				ShadowBranch:  "orion/pushed-123/feature/pushed",
				WorktreePath:  "/path/to/pushed",
				Status:        StatusPushed,
				CreatedAt:     now,
			},
			validate: func(t *testing.T, decoded Node) {
				if decoded.Status != StatusPushed {
					t.Errorf("Status mismatch: got %q", decoded.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.node)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Unmarshal
			var decoded Node
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Validate
			tt.validate(t, decoded)
		})
	}
}

func TestNodeStatusJSON(t *testing.T) {
	tests := []struct {
		name         string
		status       NodeStatus
		expectedJSON string
	}{
		{
			name:         "ReadyToPush",
			status:       StatusReadyToPush,
			expectedJSON: `"READY_TO_PUSH"`,
		},
		{
			name:         "Working",
			status:       StatusWorking,
			expectedJSON: `"WORKING"`,
		},
		{
			name:         "Fail",
			status:       StatusFail,
			expectedJSON: `"FAIL"`,
		},
		{
			name:         "Pushed",
			status:       StatusPushed,
			expectedJSON: `"PUSHED"`,
		},
		{
			name:         "Empty",
			status:       "",
			expectedJSON: `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if string(data) != tt.expectedJSON {
				t.Errorf("expected %s, got %s", tt.expectedJSON, string(data))
			}
		})
	}
}

func TestStateJSONSerialization(t *testing.T) {
	state := &State{
		RepoURL:  "https://github.com/test/repo.git",
		RepoPath: "/path/to/repo",
		Nodes: map[string]Node{
			"node1": {
				Name:          "node1",
				LogicalBranch: "feature/one",
				ShadowBranch:  "orion/run-1/feature/one",
				WorktreePath:  "/path/to/node1",
				Status:        StatusWorking,
			},
			"node2": {
				Name:          "node2",
				LogicalBranch: "feature/two",
				ShadowBranch:  "orion/run-2/feature/two",
				WorktreePath:  "/path/to/node2",
				Status:        StatusReadyToPush,
			},
		},
	}

	// Marshal
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var decoded State
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Validate
	if decoded.RepoURL != state.RepoURL {
		t.Errorf("RepoURL mismatch: got %q", decoded.RepoURL)
	}
	if decoded.RepoPath != state.RepoPath {
		t.Errorf("RepoPath mismatch: got %q", decoded.RepoPath)
	}
	if len(decoded.Nodes) != len(state.Nodes) {
		t.Errorf("Nodes count mismatch: got %d, want %d", len(decoded.Nodes), len(state.Nodes))
	}
	if decoded.Nodes["node1"].Status != StatusWorking {
		t.Errorf("node1 Status mismatch: got %q", decoded.Nodes["node1"].Status)
	}
	if decoded.Nodes["node2"].Status != StatusReadyToPush {
		t.Errorf("node2 Status mismatch: got %q", decoded.Nodes["node2"].Status)
	}
}
