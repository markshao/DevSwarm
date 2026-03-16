package workflow

import (
	"testing"
	"time"
)

// TestRunStatusConstants verifies run status constants are defined
func TestRunStatusConstants(t *testing.T) {
	statuses := []RunStatus{
		StatusPending,
		StatusRunning,
		StatusSuccess,
		StatusFailed,
	}

	for _, status := range statuses {
		if status == "" {
			t.Errorf("RunStatus constant should not be empty: %v", status)
		}
	}
}

// TestRunStructure verifies Run struct fields
func TestRunStructure(t *testing.T) {
	now := time.Now()
	run := Run{
		ID:              "run-123456",
		Workflow:        "default",
		Trigger:         "commit",
		TriggerData:     "abc1234",
		BaseBranch:      "feature/test",
		TriggeredByNode: "test-node",
		Status:          StatusRunning,
		StartTime:       now,
		Steps: []StepStatus{
			{
				ID:           "ut",
				Agent:        "ut-agent",
				Status:       StatusPending,
				NodeName:     "run-123456-ut-ut",
				ShadowBranch: "orion/run-123456/ut",
			},
		},
	}

	if run.ID != "run-123456" {
		t.Errorf("ID = %q, want %q", run.ID, "run-123456")
	}
	if run.Workflow != "default" {
		t.Errorf("Workflow = %q, want %q", run.Workflow, "default")
	}
	if run.Trigger != "commit" {
		t.Errorf("Trigger = %q, want %q", run.Trigger, "commit")
	}
	if run.TriggerData != "abc1234" {
		t.Errorf("TriggerData = %q, want %q", run.TriggerData, "abc1234")
	}
	if run.BaseBranch != "feature/test" {
		t.Errorf("BaseBranch = %q, want %q", run.BaseBranch, "feature/test")
	}
	if run.TriggeredByNode != "test-node" {
		t.Errorf("TriggeredByNode = %q, want %q", run.TriggeredByNode, "test-node")
	}
	if run.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", run.Status, StatusRunning)
	}
	if len(run.Steps) != 1 {
		t.Errorf("Steps length = %d, want %d", len(run.Steps), 1)
	}
}

// TestRunOptionalFields verifies optional Run fields
func TestRunOptionalFields(t *testing.T) {
	run := Run{
		ID:         "run-123",
		Workflow:   "default",
		Trigger:    "manual",
		BaseBranch: "main",
		Status:     StatusPending,
		StartTime:  time.Now(),
		// Optional fields left empty/zero
	}

	if run.TriggerData != "" {
		t.Errorf("TriggerData should be empty, got %q", run.TriggerData)
	}
	if run.TriggeredByNode != "" {
		t.Errorf("TriggeredByNode should be empty, got %q", run.TriggeredByNode)
	}
	if !run.EndTime.IsZero() {
		t.Errorf("EndTime should be zero value, got %v", run.EndTime)
	}
}

// TestStepStatusStructure verifies StepStatus struct fields
func TestStepStatusStructure(t *testing.T) {
	now := time.Now()
	step := StepStatus{
		ID:           "ut",
		Agent:        "ut-agent",
		Status:       StatusSuccess,
		StartTime:    now,
		EndTime:      now.Add(1 * time.Minute),
		NodeName:     "run-123-ut-ut",
		ShadowBranch: "orion/run-123/ut",
		LogPath:      ".orion/runs/run-123/logs/ut.log",
		Error:        "",
	}

	if step.ID != "ut" {
		t.Errorf("ID = %q, want %q", step.ID, "ut")
	}
	if step.Agent != "ut-agent" {
		t.Errorf("Agent = %q, want %q", step.Agent, "ut-agent")
	}
	if step.Status != StatusSuccess {
		t.Errorf("Status = %q, want %q", step.Status, StatusSuccess)
	}
	if step.NodeName != "run-123-ut-ut" {
		t.Errorf("NodeName = %q, want %q", step.NodeName, "run-123-ut-ut")
	}
	if step.ShadowBranch != "orion/run-123/ut" {
		t.Errorf("ShadowBranch = %q, want %q", step.ShadowBranch, "orion/run-123/ut")
	}
}

// TestStepStatusOptionalFields verifies optional StepStatus fields
func TestStepStatusOptionalFields(t *testing.T) {
	step := StepStatus{
		ID:     "cr",
		Agent:  "cr-agent",
		Status: StatusPending,
		// Optional fields left empty
	}

	if step.NodeName != "" {
		t.Errorf("NodeName should be empty, got %q", step.NodeName)
	}
	if step.ShadowBranch != "" {
		t.Errorf("ShadowBranch should be empty, got %q", step.ShadowBranch)
	}
	if step.LogPath != "" {
		t.Errorf("LogPath should be empty, got %q", step.LogPath)
	}
	if step.Error != "" {
		t.Errorf("Error should be empty, got %q", step.Error)
	}
	if !step.StartTime.IsZero() {
		t.Errorf("StartTime should be zero value, got %v", step.StartTime)
	}
	if !step.EndTime.IsZero() {
		t.Errorf("EndTime should be zero value, got %v", step.EndTime)
	}
}

// TestStepStatusWithError verifies StepStatus with error
func TestStepStatusWithError(t *testing.T) {
	step := StepStatus{
		ID:     "ut",
		Agent:  "ut-agent",
		Status: StatusFailed,
		Error:  "agent execution failed: exit code 1",
	}

	if step.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", step.Status, StatusFailed)
	}
	if step.Error == "" {
		t.Error("Error should not be empty for failed step")
	}
}

// TestRunStatusTransitions verifies typical run status transitions
func TestRunStatusTransitions(t *testing.T) {
	// A typical run goes: Pending -> Running -> Success/Failed
	run := Run{
		ID:        "run-test",
		Workflow:  "default",
		Trigger:   "manual",
		BaseBranch: "main",
		Status:    StatusPending,
		StartTime: time.Now(),
	}

	// Transition to Running
	run.Status = StatusRunning
	if run.Status != StatusRunning {
		t.Errorf("Failed to transition to Running: %q", run.Status)
	}

	// Transition to Success
	run.Status = StatusSuccess
	run.EndTime = time.Now()
	if run.Status != StatusSuccess {
		t.Errorf("Failed to transition to Success: %q", run.Status)
	}
}

// TestRunWithMultipleSteps verifies Run with multiple pipeline steps
func TestRunWithMultipleSteps(t *testing.T) {
	run := Run{
		ID:         "run-multi",
		Workflow:   "default",
		Trigger:    "commit",
		BaseBranch: "feature/test",
		Status:     StatusRunning,
		StartTime:  time.Now(),
		Steps: []StepStatus{
			{
				ID:     "ut",
				Agent:  "ut-agent",
				Status: StatusSuccess,
			},
			{
				ID:     "cr",
				Agent:  "cr-agent",
				Status: StatusRunning,
			},
			{
				ID:     "build",
				Agent:  "build-agent",
				Status: StatusPending,
			},
		},
	}

	if len(run.Steps) != 3 {
		t.Errorf("Steps length = %d, want %d", len(run.Steps), 3)
	}

	// Verify step statuses
	expectedStatuses := []RunStatus{StatusSuccess, StatusRunning, StatusPending}
	for i, expected := range expectedStatuses {
		if run.Steps[i].Status != expected {
			t.Errorf("Step[%d].Status = %q, want %q", i, run.Steps[i].Status, expected)
		}
	}
}
