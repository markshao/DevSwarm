package notification

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"orion/internal/ai"
)

var (
	waitingInputPatterns = []string{
		"Would you like to run the following command\\?",
		"Press enter to confirm",
		"press enter to confirm",
		"esc to cancel",
		"Yes, proceed",
		"Yes, and don't ask again",
		"approve",
		"allow",
		"waiting for input",
		"input required",
	}
)

type SnapshotClassifier interface {
	Classify(nodeName, screen string, stableFor time.Duration) (Classification, error)
}

type LLMClassifier struct {
	client *ai.Client
}

func NewLLMClassifier() (*LLMClassifier, error) {
	client, err := ai.NewClient()
	if err != nil {
		return nil, err
	}
	return &LLMClassifier{client: client}, nil
}

func normalizeScreen(screen string) string {
	lines := strings.Split(screen, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func hashScreen(screen string) string {
	sum := sha256.Sum256([]byte(normalizeScreen(screen)))
	return fmt.Sprintf("%x", sum[:])
}

func tail(screen string, lines int) string {
	if lines <= 0 {
		return screen
	}
	parts := strings.Split(screen, "\n")
	if len(parts) <= lines {
		return screen
	}
	return strings.Join(parts[len(parts)-lines:], "\n")
}

func HeuristicClassify(screen string, stableFor, silenceThreshold time.Duration) Classification {
	tailText := strings.ToLower(tail(normalizeScreen(screen), 30))
	for _, pattern := range waitingInputPatterns {
		if strings.Contains(tailText, strings.ToLower(pattern)) {
			return Classification{State: StateWaitingInput, Reason: "prompt_like_text"}
		}
	}
	if stableFor < silenceThreshold {
		return Classification{State: StateRunning, Reason: "recent_output_change"}
	}
	return Classification{State: StateQuietCandidate, Reason: "stable_output"}
}

func (c *LLMClassifier) Classify(nodeName, screen string, stableFor time.Duration) (Classification, error) {
	systemPrompt := `You are classifying a stable terminal snapshot from an interactive coding agent.

Return JSON only:
{
  "state": "waiting_input|completed_idle|still_working|unknown",
  "reason": "short reason"
}

Definitions:
- waiting_input: the screen is clearly asking the human to confirm, choose, approve, or provide input.
- completed_idle: the agent appears finished and is not asking for more input.
- still_working: the agent appears to still be processing or actively working.
- unknown: not enough evidence.

Be conservative. Only choose waiting_input if the terminal clearly requests user action.`

	userPrompt := fmt.Sprintf("Node: %s\nStable for: %s\n\nTerminal snapshot:\n%s", nodeName, stableFor.Round(time.Second), tail(normalizeScreen(screen), 80))
	content, err := c.client.GenerateText(systemPrompt, userPrompt, 0, 128)
	if err != nil {
		return Classification{}, err
	}

	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var out struct {
		State  string `json:"state"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return Classification{}, fmt.Errorf("failed to parse LLM classification: %w", err)
	}

	switch out.State {
	case "waiting_input":
		return Classification{State: StateWaitingInput, Reason: strings.TrimSpace(out.Reason)}, nil
	case "completed_idle":
		return Classification{State: StateCompletedIdle, Reason: strings.TrimSpace(out.Reason)}, nil
	case "still_working":
		return Classification{State: StateRunning, Reason: strings.TrimSpace(out.Reason)}, nil
	case "unknown":
		return Classification{State: StateUnknown, Reason: strings.TrimSpace(out.Reason)}, nil
	default:
		return Classification{}, fmt.Errorf("unexpected LLM state %q", out.State)
	}
}
