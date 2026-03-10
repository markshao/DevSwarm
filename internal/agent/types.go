package agent

import (
	"context"
)

// Provider defines the interface for a code agent provider (e.g., Qwen, Trae, Claude).
type Provider interface {
	// Name returns the provider name (e.g. "qwen")
	Name() string

	// Run executes the agent with the given prompt in the specified directory.
	// It returns the output (e.g. logs, result summary) and an error.
	Run(ctx context.Context, prompt string, workdir string, env []string) (string, error)
}

// Config holds configuration for a specific provider.
type Config struct {
	Provider string            `yaml:"provider"` // e.g. "qwen"
	Model    string            `yaml:"model"`
	APIKey   string            `yaml:"api_key"`     // Or load from env
	Endpoint string            `yaml:"endpoint"`    // Optional
	Params   map[string]string `yaml:"params"`      // Extra params
}
