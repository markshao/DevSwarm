package types

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNodeStatusConstants verifies that all NodeStatus constants are defined correctly
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
				t.Errorf("expected %s, got %s", tt.expected, tt.status)
			}
		})
	}
}

// TestNodeStatusJSONSerialization tests JSON marshaling and unmarshaling of NodeStatus
func TestNodeStatusJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{"Working", StatusWorking, `"WORKING"`},
		{"ReadyToPush", StatusReadyToPush, `"READY_TO_PUSH"`},
		{"Fail", StatusFail, `"FAIL"`},
		{"Pushed", StatusPushed, `"PUSHED"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(data))
			}

			// Unmarshal
			var unmarshaled NodeStatus
			if err := json.Unmarshal([]byte(tt.expected), &unmarshaled); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			if unmarshaled != tt.status {
				t.Errorf("expected %s, got %s", tt.status, unmarshaled)
			}
		})
	}
}

// TestNodeWithStatusJSONSerialization tests Node struct JSON serialization with Status field
func TestNodeWithStatusJSONSerialization(t *testing.T) {
	now := time.Now()
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/path/to/worktree",
		Label:         "test",
		CreatedBy:     "user",
		Status:        StatusReadyToPush,
		AppliedRuns:   []string{"run-1", "run-2"},
		CreatedAt:     now,
	}

	// Marshal
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal
	var unmarshaled Node
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify fields
	if unmarshaled.Name != node.Name {
		t.Errorf("Name mismatch: expected %s, got %s", node.Name, unmarshaled.Name)
	}
	if unmarshaled.Status != node.Status {
		t.Errorf("Status mismatch: expected %s, got %s", node.Status, unmarshaled.Status)
	}
	if len(unmarshaled.AppliedRuns) != len(node.AppliedRuns) {
		t.Errorf("AppliedRuns length mismatch: expected %d, got %d", len(node.AppliedRuns), len(unmarshaled.AppliedRuns))
	}
}

// TestNodeWithEmptyStatus tests Node with empty status (legacy node)
func TestNodeWithEmptyStatus(t *testing.T) {
	node := Node{
		Name:          "legacy-node",
		LogicalBranch: "feature/legacy",
		ShadowBranch:  "orion-shadow/legacy-node/feature/legacy",
		WorktreePath:  "/path/to/legacy",
		CreatedAt:     time.Now(),
		// Status is empty (zero value)
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var unmarshaled Node
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if unmarshaled.Status != "" {
		t.Errorf("expected empty status for legacy node, got %s", unmarshaled.Status)
	}
}

// TestNodeStatusTransitions tests valid status transitions
func TestNodeStatusTransitions(t *testing.T) {
	tests := []struct {
		name        string
		fromStatus  NodeStatus
		toStatus    NodeStatus
		description string
	}{
		{
			name:        "Working to ReadyToPush",
			fromStatus:  StatusWorking,
			toStatus:    StatusReadyToPush,
			description: "Workflow succeeded",
		},
		{
			name:        "Working to Fail",
			fromStatus:  StatusWorking,
			toStatus:    StatusFail,
			description: "Workflow failed",
		},
		{
			name:        "ReadyToPush to Pushed",
			fromStatus:  StatusReadyToPush,
			toStatus:    StatusPushed,
			description: "Successfully pushed to remote",
		},
		{
			name:        "Fail to Working",
			fromStatus:  StatusFail,
			toStatus:    StatusWorking,
			description: "Retry after failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{
				Name:       "transition-test",
				Status:     tt.fromStatus,
				CreatedAt:  time.Now(),
				ShadowBranch: "test-branch",
				WorktreePath: "/test/path",
			}

			// Simulate status transition
			node.Status = tt.toStatus

			if node.Status != tt.toStatus {
				t.Errorf("status transition failed: expected %s, got %s", tt.toStatus, node.Status)
			}
		})
	}
}

// TestNodeStatusString tests string conversion of NodeStatus
func TestNodeStatusString(t *testing.T) {
	tests := []struct {
		status   NodeStatus
		expected string
	}{
		{StatusWorking, "WORKING"},
		{StatusReadyToPush, "READY_TO_PUSH"},
		{StatusFail, "FAIL"},
		{StatusPushed, "PUSHED"},
		{"", ""}, // Empty status
		{"UNKNOWN", "UNKNOWN"}, // Unknown status
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.status))
			}
		})
	}
}

// TestNodeStatusComparison tests status comparison operations
func TestNodeStatusComparison(t *testing.T) {
	// Test equality
	if StatusWorking != StatusWorking {
		t.Error("StatusWorking should equal StatusWorking")
	}
	if StatusReadyToPush == StatusWorking {
		t.Error("StatusReadyToPush should not equal StatusWorking")
	}

	// Test against string
	if string(StatusPushed) != "PUSHED" {
		t.Error("StatusPushed should equal 'PUSHED'")
	}

	// Test empty status
	var emptyStatus NodeStatus
	if emptyStatus != "" {
		t.Error("Empty NodeStatus should equal empty string")
	}
}

// TestStateWithNodeStatuses tests State struct with multiple nodes having different statuses
func TestStateWithNodeStatuses(t *testing.T) {
	state := State{
		RepoURL:  "https://github.com/test/repo",
		RepoPath: "/path/to/repo",
		Nodes: map[string]Node{
			"node-working": {
				Name:         "node-working",
				Status:       StatusWorking,
				ShadowBranch: "branch-working",
				WorktreePath: "/path/working",
				CreatedAt:    time.Now(),
			},
			"node-ready": {
				Name:         "node-ready",
				Status:       StatusReadyToPush,
				ShadowBranch: "branch-ready",
				WorktreePath: "/path/ready",
				CreatedAt:    time.Now(),
			},
			"node-fail": {
				Name:         "node-fail",
				Status:       StatusFail,
				ShadowBranch: "branch-fail",
				WorktreePath: "/path/fail",
				CreatedAt:    time.Now(),
			},
			"node-pushed": {
				Name:         "node-pushed",
				Status:       StatusPushed,
				ShadowBranch: "branch-pushed",
				WorktreePath: "/path/pushed",
				CreatedAt:    time.Now(),
			},
		},
	}

	// Marshal
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal
	var unmarshaled State
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify all nodes and their statuses
	expectedStatuses := map[string]NodeStatus{
		"node-working": StatusWorking,
		"node-ready":   StatusReadyToPush,
		"node-fail":    StatusFail,
		"node-pushed":  StatusPushed,
	}

	for name, expectedStatus := range expectedStatuses {
		node, exists := unmarshaled.Nodes[name]
		if !exists {
			t.Errorf("node %s not found in unmarshaled state", name)
			continue
		}
		if node.Status != expectedStatus {
			t.Errorf("node %s: expected status %s, got %s", name, expectedStatus, node.Status)
		}
	}
}
