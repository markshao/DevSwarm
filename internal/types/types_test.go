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

func TestNodeStatusJSONSerialization(t *testing.T) {
	tests := []struct {
		name         string
		status       NodeStatus
		expectedJSON string
	}{
		{
			name:         "StatusWorking",
			status:       StatusWorking,
			expectedJSON: `"WORKING"`,
		},
		{
			name:         "StatusReadyToPush",
			status:       StatusReadyToPush,
			expectedJSON: `"READY_TO_PUSH"`,
		},
		{
			name:         "StatusFail",
			status:       StatusFail,
			expectedJSON: `"FAIL"`,
		},
		{
			name:         "StatusPushed",
			status:       StatusPushed,
			expectedJSON: `"PUSHED"`,
		},
		{
			name:         "EmptyStatus",
			status:       "",
			expectedJSON: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				ShadowBranch:  "orion/test/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        tt.status,
				CreatedAt:     time.Now(),
			}

			data, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			// For empty status, it should be omitted due to omitempty
			if tt.status == "" {
				if containsStatusField(string(data)) {
					t.Errorf("empty status should be omitted from JSON, got: %s", string(data))
				}
			} else {
				if !containsSubstring(string(data), tt.expectedJSON) {
					t.Errorf("expected JSON to contain %s, got: %s", tt.expectedJSON, string(data))
				}
			}
		})
	}
}

func TestNodeWithStatusDeserialization(t *testing.T) {
	tests := []struct {
		name         string
		jsonInput    string
		expectedStatus NodeStatus
	}{
		{
			name:         "StatusWorking",
			jsonInput:    `{"name":"test","logical_branch":"feat","shadow_branch":"orion/test/feat","worktree_path":"/tmp","status":"WORKING","created_at":"2024-01-01T00:00:00Z"}`,
			expectedStatus: StatusWorking,
		},
		{
			name:         "StatusReadyToPush",
			jsonInput:    `{"name":"test","logical_branch":"feat","shadow_branch":"orion/test/feat","worktree_path":"/tmp","status":"READY_TO_PUSH","created_at":"2024-01-01T00:00:00Z"}`,
			expectedStatus: StatusReadyToPush,
		},
		{
			name:         "StatusFail",
			jsonInput:    `{"name":"test","logical_branch":"feat","shadow_branch":"orion/test/feat","worktree_path":"/tmp","status":"FAIL","created_at":"2024-01-01T00:00:00Z"}`,
			expectedStatus: StatusFail,
		},
		{
			name:         "StatusPushed",
			jsonInput:    `{"name":"test","logical_branch":"feat","shadow_branch":"orion/test/feat","worktree_path":"/tmp","status":"PUSHED","created_at":"2024-01-01T00:00:00Z"}`,
			expectedStatus: StatusPushed,
		},
		{
			name:         "NoStatusField",
			jsonInput:    `{"name":"test","logical_branch":"feat","shadow_branch":"orion/test/feat","worktree_path":"/tmp","created_at":"2024-01-01T00:00:00Z"}`,
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node Node
			err := json.Unmarshal([]byte(tt.jsonInput), &node)
			if err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}

			if node.Status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, node.Status)
			}
		})
	}
}

func TestNodeStatusRoundTrip(t *testing.T) {
	originalNode := Node{
		Name:          "roundtrip-node",
		LogicalBranch: "feature/roundtrip",
		BaseBranch:    "main",
		ShadowBranch:  "orion/roundtrip/feature/roundtrip",
		WorktreePath:  "/tmp/roundtrip",
		TmuxSession:   "roundtrip-session",
		Label:         "test-label",
		CreatedBy:     "test-user",
		AppliedRuns:   []string{"run-1", "run-2"},
		Status:        StatusReadyToPush,
		CreatedAt:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	// Marshal to JSON
	data, err := json.Marshal(originalNode)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal from JSON
	var loadedNode Node
	err = json.Unmarshal(data, &loadedNode)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify all fields
	if loadedNode.Name != originalNode.Name {
		t.Errorf("Name mismatch: %q vs %q", loadedNode.Name, originalNode.Name)
	}
	if loadedNode.LogicalBranch != originalNode.LogicalBranch {
		t.Errorf("LogicalBranch mismatch: %q vs %q", loadedNode.LogicalBranch, originalNode.LogicalBranch)
	}
	if loadedNode.BaseBranch != originalNode.BaseBranch {
		t.Errorf("BaseBranch mismatch: %q vs %q", loadedNode.BaseBranch, originalNode.BaseBranch)
	}
	if loadedNode.ShadowBranch != originalNode.ShadowBranch {
		t.Errorf("ShadowBranch mismatch: %q vs %q", loadedNode.ShadowBranch, originalNode.ShadowBranch)
	}
	if loadedNode.WorktreePath != originalNode.WorktreePath {
		t.Errorf("WorktreePath mismatch: %q vs %q", loadedNode.WorktreePath, originalNode.WorktreePath)
	}
	if loadedNode.TmuxSession != originalNode.TmuxSession {
		t.Errorf("TmuxSession mismatch: %q vs %q", loadedNode.TmuxSession, originalNode.TmuxSession)
	}
	if loadedNode.Label != originalNode.Label {
		t.Errorf("Label mismatch: %q vs %q", loadedNode.Label, originalNode.Label)
	}
	if loadedNode.CreatedBy != originalNode.CreatedBy {
		t.Errorf("CreatedBy mismatch: %q vs %q", loadedNode.CreatedBy, originalNode.CreatedBy)
	}
	if loadedNode.Status != originalNode.Status {
		t.Errorf("Status mismatch: %q vs %q", loadedNode.Status, originalNode.Status)
	}
	if len(loadedNode.AppliedRuns) != len(originalNode.AppliedRuns) {
		t.Errorf("AppliedRuns length mismatch: %d vs %d", len(loadedNode.AppliedRuns), len(originalNode.AppliedRuns))
	}
	for i, run := range loadedNode.AppliedRuns {
		if run != originalNode.AppliedRuns[i] {
			t.Errorf("AppliedRuns[%d] mismatch: %q vs %q", i, run, originalNode.AppliedRuns[i])
		}
	}
	if !loadedNode.CreatedAt.Equal(originalNode.CreatedAt) {
		t.Errorf("CreatedAt mismatch: %v vs %v", loadedNode.CreatedAt, originalNode.CreatedAt)
	}
}

func TestNodeStatusTransitions(t *testing.T) {
	node := Node{
		Name:          "transition-node",
		LogicalBranch: "feature/transition",
		ShadowBranch:  "orion/transition/feature/transition",
		WorktreePath:  "/tmp/transition",
		Status:        StatusWorking,
		CreatedAt:     time.Now(),
	}

	// Initial status should be WORKING
	if node.Status != StatusWorking {
		t.Errorf("initial status should be WORKING, got %q", node.Status)
	}

	// Simulate workflow success -> READY_TO_PUSH
	node.Status = StatusReadyToPush
	if node.Status != StatusReadyToPush {
		t.Errorf("after workflow success, status should be READY_TO_PUSH, got %q", node.Status)
	}

	// Simulate push -> PUSHED
	node.Status = StatusPushed
	if node.Status != StatusPushed {
		t.Errorf("after push, status should be PUSHED, got %q", node.Status)
	}

	// Simulate workflow failure -> FAIL
	node.Status = StatusFail
	if node.Status != StatusFail {
		t.Errorf("after workflow failure, status should be FAIL, got %q", node.Status)
	}
}

// Helper functions

func containsStatusField(jsonStr string) bool {
	// Check if "status" field exists in JSON
	return containsSubstring(jsonStr, `"status"`)
}

func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
