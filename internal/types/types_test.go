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

func TestNodeStatusJSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		validate func(*testing.T, []byte)
	}{
		{
			name:     "Working status",
			status:   StatusWorking,
			validate: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatalf("json.Unmarshal failed: %v", err)
				}
				if result["status"] != "WORKING" {
					t.Errorf("expected status %q, got %q", "WORKING", result["status"])
				}
			},
		},
		{
			name:     "ReadyToPush status",
			status:   StatusReadyToPush,
			validate: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatalf("json.Unmarshal failed: %v", err)
				}
				if result["status"] != "READY_TO_PUSH" {
					t.Errorf("expected status %q, got %q", "READY_TO_PUSH", result["status"])
				}
			},
		},
		{
			name:     "Fail status",
			status:   StatusFail,
			validate: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatalf("json.Unmarshal failed: %v", err)
				}
				if result["status"] != "FAIL" {
					t.Errorf("expected status %q, got %q", "FAIL", result["status"])
				}
			},
		},
		{
			name:     "Pushed status",
			status:   StatusPushed,
			validate: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatalf("json.Unmarshal failed: %v", err)
				}
				if result["status"] != "PUSHED" {
					t.Errorf("expected status %q, got %q", "PUSHED", result["status"])
				}
			},
		},
		{
			name:     "Empty status (omitempty)",
			status:   "",
			validate: func(t *testing.T, data []byte) {
				// Empty status should be omitted due to omitempty
				var result map[string]interface{}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatalf("json.Unmarshal failed: %v", err)
				}
				if _, exists := result["status"]; exists {
					t.Errorf("empty status should be omitted due to omitempty, but got: %v", result["status"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{
				Name:   "test-node",
				Status: tt.status,
			}

			data, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			tt.validate(t, data)
		})
	}
}

func TestNodeStatusJSONUnmarshaling(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected NodeStatus
	}{
		{
			name:     "Working status",
			json:     `{"name":"test","status":"WORKING"}`,
			expected: StatusWorking,
		},
		{
			name:     "ReadyToPush status",
			json:     `{"name":"test","status":"READY_TO_PUSH"}`,
			expected: StatusReadyToPush,
		},
		{
			name:     "Fail status",
			json:     `{"name":"test","status":"FAIL"}`,
			expected: StatusFail,
		},
		{
			name:     "Pushed status",
			json:     `{"name":"test","status":"PUSHED"}`,
			expected: StatusPushed,
		},
		{
			name:     "Missing status (defaults to empty)",
			json:     `{"name":"test"}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node Node
			if err := json.Unmarshal([]byte(tt.json), &node); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}

			if node.Status != tt.expected {
				t.Errorf("expected status %q, got %q", tt.expected, node.Status)
			}
		})
	}
}

func TestNodeWithStatus(t *testing.T) {
	now := time.Now()

	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/path/to/worktree",
		TmuxSession:   "orion-test-node",
		Label:         "testing",
		CreatedBy:     "user",
		AppliedRuns:   []string{"run-1", "run-2"},
		Status:        StatusReadyToPush,
		CreatedAt:     now,
	}

	// Test JSON serialization
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Test JSON deserialization
	var loadedNode Node
	if err := json.Unmarshal(data, &loadedNode); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify all fields
	if loadedNode.Name != node.Name {
		t.Errorf("Name mismatch: expected %q, got %q", node.Name, loadedNode.Name)
	}
	if loadedNode.LogicalBranch != node.LogicalBranch {
		t.Errorf("LogicalBranch mismatch: expected %q, got %q", node.LogicalBranch, loadedNode.LogicalBranch)
	}
	if loadedNode.BaseBranch != node.BaseBranch {
		t.Errorf("BaseBranch mismatch: expected %q, got %q", node.BaseBranch, loadedNode.BaseBranch)
	}
	if loadedNode.ShadowBranch != node.ShadowBranch {
		t.Errorf("ShadowBranch mismatch: expected %q, got %q", node.ShadowBranch, loadedNode.ShadowBranch)
	}
	if loadedNode.WorktreePath != node.WorktreePath {
		t.Errorf("WorktreePath mismatch: expected %q, got %q", node.WorktreePath, loadedNode.WorktreePath)
	}
	if loadedNode.TmuxSession != node.TmuxSession {
		t.Errorf("TmuxSession mismatch: expected %q, got %q", node.TmuxSession, loadedNode.TmuxSession)
	}
	if loadedNode.Label != node.Label {
		t.Errorf("Label mismatch: expected %q, got %q", node.Label, loadedNode.Label)
	}
	if loadedNode.CreatedBy != node.CreatedBy {
		t.Errorf("CreatedBy mismatch: expected %q, got %q", node.CreatedBy, loadedNode.CreatedBy)
	}
	if len(loadedNode.AppliedRuns) != len(node.AppliedRuns) {
		t.Errorf("AppliedRuns length mismatch: expected %d, got %d", len(node.AppliedRuns), len(loadedNode.AppliedRuns))
	}
	if loadedNode.Status != node.Status {
		t.Errorf("Status mismatch: expected %q, got %q", node.Status, loadedNode.Status)
	}
}

func TestNodeStatusTransitions(t *testing.T) {
	node := Node{
		Name:   "test-node",
		Status: StatusWorking,
	}

	// Simulate status transitions
	transitions := []struct {
		from NodeStatus
		to   NodeStatus
	}{
		{StatusWorking, StatusReadyToPush},
		{StatusReadyToPush, StatusPushed},
		{StatusWorking, StatusFail},
		{StatusFail, StatusWorking},
	}

	for _, tr := range transitions {
		t.Run(string(tr.from)+"_to_"+string(tr.to), func(t *testing.T) {
			node.Status = tr.from
			// Simulate transition
			node.Status = tr.to
			if node.Status != tr.to {
				t.Errorf("transition failed: expected %q, got %q", tr.to, node.Status)
			}
		})
	}
}

func TestEmptyNodeStatus(t *testing.T) {
	// Test that empty NodeStatus is handled correctly
	var status NodeStatus
	if status != "" {
		t.Errorf("empty NodeStatus should be empty string, got %q", status)
	}

	// Test JSON marshaling of empty status
	node := Node{Name: "test"}
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Empty status should be omitted
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if _, exists := result["status"]; exists {
		t.Errorf("empty status should be omitted due to omitempty, but got: %v", result["status"])
	}
}
