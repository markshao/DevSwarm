package types

import (
	"testing"
	"time"
)

// TestNodeStructure verifies Node struct fields
func TestNodeStructure(t *testing.T) {
	now := time.Now()
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/path/to/worktree",
		TmuxSession:   "orion-test-node",
		Label:         "test",
		CreatedBy:     "user",
		AppliedRuns:   []string{"run-1", "run-2"},
		CreatedAt:     now,
	}

	if node.Name != "test-node" {
		t.Errorf("Name = %q, want %q", node.Name, "test-node")
	}
	if node.LogicalBranch != "feature/test" {
		t.Errorf("LogicalBranch = %q, want %q", node.LogicalBranch, "feature/test")
	}
	if node.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want %q", node.BaseBranch, "main")
	}
	if node.ShadowBranch != "orion-shadow/test-node/feature/test" {
		t.Errorf("ShadowBranch = %q, want %q", node.ShadowBranch, "orion-shadow/test-node/feature/test")
	}
	if node.WorktreePath != "/path/to/worktree" {
		t.Errorf("WorktreePath = %q, want %q", node.WorktreePath, "/path/to/worktree")
	}
	if node.TmuxSession != "orion-test-node" {
		t.Errorf("TmuxSession = %q, want %q", node.TmuxSession, "orion-test-node")
	}
	if node.Label != "test" {
		t.Errorf("Label = %q, want %q", node.Label, "test")
	}
	if node.CreatedBy != "user" {
		t.Errorf("CreatedBy = %q, want %q", node.CreatedBy, "user")
	}
	if len(node.AppliedRuns) != 2 {
		t.Errorf("AppliedRuns length = %d, want %d", len(node.AppliedRuns), 2)
	}
}

// TestNodeOptionalFields verifies optional Node fields can be empty
func TestNodeOptionalFields(t *testing.T) {
	node := Node{
		Name:          "minimal-node",
		LogicalBranch: "main",
		ShadowBranch:  "main",
		WorktreePath:  "/path",
		CreatedAt:     time.Now(),
		// Optional fields left empty
	}

	if node.BaseBranch != "" {
		t.Errorf("BaseBranch should be empty by default, got %q", node.BaseBranch)
	}
	if node.TmuxSession != "" {
		t.Errorf("TmuxSession should be empty by default, got %q", node.TmuxSession)
	}
	if node.Label != "" {
		t.Errorf("Label should be empty by default, got %q", node.Label)
	}
	if node.CreatedBy != "" {
		t.Errorf("CreatedBy should be empty by default, got %q", node.CreatedBy)
	}
	if node.AppliedRuns != nil {
		t.Errorf("AppliedRuns should be nil by default, got %v", node.AppliedRuns)
	}
}

// TestStateStructure verifies State struct fields
func TestStateStructure(t *testing.T) {
	state := State{
		RepoURL:  "https://github.com/user/repo.git",
		RepoPath: "/path/to/repo",
		Nodes: map[string]Node{
			"node1": {
				Name:          "node1",
				LogicalBranch: "feature/1",
				ShadowBranch:  "orion/node1",
				WorktreePath:  "/path/node1",
				CreatedAt:     time.Now(),
			},
		},
	}

	if state.RepoURL != "https://github.com/user/repo.git" {
		t.Errorf("RepoURL = %q, want %q", state.RepoURL, "https://github.com/user/repo.git")
	}
	if state.RepoPath != "/path/to/repo" {
		t.Errorf("RepoPath = %q, want %q", state.RepoPath, "/path/to/repo")
	}
	if len(state.Nodes) != 1 {
		t.Errorf("Nodes length = %d, want %d", len(state.Nodes), 1)
	}
}

// TestConfigStructure verifies Config struct fields
func TestConfigStructure(t *testing.T) {
	config := Config{
		Version:   1,
		Workspace: "workspaces",
		Git: GitConfig{
			MainBranch: "main",
			User:       "Test User",
			Email:      "test@example.com",
		},
		Agents: AgentsConfig{
			DefaultProvider: "qwen",
			Providers: map[string]ProviderSettings{
				"qwen": {
					Model: "qwen-max",
				},
			},
		},
		Runtime: RuntimeConfig{
			ArtifactDir: ".orion/runs",
		},
	}

	if config.Version != 1 {
		t.Errorf("Version = %d, want %d", config.Version, 1)
	}
	if config.Workspace != "workspaces" {
		t.Errorf("Workspace = %q, want %q", config.Workspace, "workspaces")
	}
	if config.Git.MainBranch != "main" {
		t.Errorf("Git.MainBranch = %q, want %q", config.Git.MainBranch, "main")
	}
	if config.Agents.DefaultProvider != "qwen" {
		t.Errorf("Agents.DefaultProvider = %q, want %q", config.Agents.DefaultProvider, "qwen")
	}
}

// TestWorkflowStructure verifies Workflow struct fields
func TestWorkflowStructure(t *testing.T) {
	workflow := Workflow{
		Name: "default",
		Trigger: WorkflowTrigger{
			Event: "commit",
		},
		Pipeline: []PipelineStep{
			{
				ID:     "ut",
				Agent:  "ut-agent",
				Branch: "shadow",
				Suffix: "ut",
			},
			{
				ID:        "cr",
				Agent:     "cr-agent",
				Branch:    "shadow",
				Suffix:    "cr",
				DependsOn: []string{"ut"},
			},
		},
	}

	if workflow.Name != "default" {
		t.Errorf("Name = %q, want %q", workflow.Name, "default")
	}
	if workflow.Trigger.Event != "commit" {
		t.Errorf("Trigger.Event = %q, want %q", workflow.Trigger.Event, "commit")
	}
	if len(workflow.Pipeline) != 2 {
		t.Errorf("Pipeline length = %d, want %d", len(workflow.Pipeline), 2)
	}
	if len(workflow.Pipeline[1].DependsOn) != 1 {
		t.Errorf("Pipeline[1].DependsOn length = %d, want %d", len(workflow.Pipeline[1].DependsOn), 1)
	}
}

// TestAgentStructure verifies Agent struct fields
func TestAgentStructure(t *testing.T) {
	agent := Agent{
		Name: "ut-agent",
		Runtime: AgentRuntime{
			Provider: "qwen",
			Model:    "qwen-max",
			Params: map[string]string{
				"temperature": "0.7",
			},
		},
		Prompt: "ut.md",
		Env:    []string{"API_KEY", "MODEL_NAME"},
	}

	if agent.Name != "ut-agent" {
		t.Errorf("Name = %q, want %q", agent.Name, "ut-agent")
	}
	if agent.Runtime.Provider != "qwen" {
		t.Errorf("Runtime.Provider = %q, want %q", agent.Runtime.Provider, "qwen")
	}
	if agent.Runtime.Model != "qwen-max" {
		t.Errorf("Runtime.Model = %q, want %q", agent.Runtime.Model, "qwen-max")
	}
	if len(agent.Runtime.Params) != 1 {
		t.Errorf("Runtime.Params length = %d, want %d", len(agent.Runtime.Params), 1)
	}
	if agent.Prompt != "ut.md" {
		t.Errorf("Prompt = %q, want %q", agent.Prompt, "ut.md")
	}
}

// TestProviderSettingsStructure verifies ProviderSettings struct fields
func TestProviderSettingsStructure(t *testing.T) {
	settings := ProviderSettings{
		APIKeyEnv: "API_KEY",
		Model:     "qwen-max",
		Endpoint:  "https://api.example.com",
		Command:   "qwen {{.Prompt}} -y",
		Params: map[string]string{
			"max_tokens": "4096",
		},
	}

	if settings.APIKeyEnv != "API_KEY" {
		t.Errorf("APIKeyEnv = %q, want %q", settings.APIKeyEnv, "API_KEY")
	}
	if settings.Model != "qwen-max" {
		t.Errorf("Model = %q, want %q", settings.Model, "qwen-max")
	}
	if settings.Endpoint != "https://api.example.com" {
		t.Errorf("Endpoint = %q, want %q", settings.Endpoint, "https://api.example.com")
	}
	if settings.Command != "qwen {{.Prompt}} -y" {
		t.Errorf("Command = %q, want %q", settings.Command, "qwen {{.Prompt}} -y")
	}
}
