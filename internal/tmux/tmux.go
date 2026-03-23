package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type PaneMeta struct {
	SessionName        string
	WindowIndex        int
	PaneIndex          int
	PaneID             string
	PaneCurrentCommand string
	PaneTitle          string
	PaneDead           bool
	AlternateOn        bool
}

// SessionExists checks if a tmux session exists.
func SessionExists(sessionName string) bool {
	return exec.Command("tmux", "has-session", "-t", sessionName).Run() == nil
}

// NewSession creates a new detached tmux session.
func NewSession(sessionName, cwd string) error {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", cwd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create tmux session: %s: %w", string(output), err)
	}
	return nil
}

// SendKeys sends keys to a tmux session.
func SendKeys(sessionName, keys string) error {
	cmd := exec.Command("tmux", "send-keys", "-t", sessionName, keys, "C-m")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to send keys to session %s: %s: %w", sessionName, string(output), err)
	}
	return nil
}

// PaneExists checks if a tmux pane exists.
func PaneExists(target string) bool {
	return exec.Command("tmux", "display-message", "-p", "-t", target, "#{pane_id}").Run() == nil
}

// GetPaneMeta returns formatted metadata for a pane target.
func GetPaneMeta(target string) (*PaneMeta, error) {
	cmd := exec.Command("tmux", "display-message", "-p", "-t", target, "#{session_name}\t#{window_index}\t#{pane_index}\t#{pane_id}\t#{pane_current_command}\t#{pane_title}\t#{pane_dead}\t#{alternate_on}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get pane metadata for %s: %w", target, err)
	}

	parts := strings.Split(strings.TrimSpace(string(output)), "\t")
	if len(parts) != 8 {
		return nil, fmt.Errorf("unexpected tmux pane metadata format for %s", target)
	}

	windowIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid window index for %s: %w", target, err)
	}
	paneIndex, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid pane index for %s: %w", target, err)
	}

	return &PaneMeta{
		SessionName:        parts[0],
		WindowIndex:        windowIndex,
		PaneIndex:          paneIndex,
		PaneID:             parts[3],
		PaneCurrentCommand: parts[4],
		PaneTitle:          parts[5],
		PaneDead:           parts[6] == "1",
		AlternateOn:        parts[7] == "1",
	}, nil
}

// GetPrimaryPane returns the pane metadata for the initial pane in a session.
func GetPrimaryPane(sessionName string) (*PaneMeta, error) {
	return GetPaneMeta(fmt.Sprintf("%s:0.0", sessionName))
}

// CapturePane captures pane contents, preferring alternate screen if requested.
func CapturePane(target string, alternate bool, lines int) (string, error) {
	start := "-"
	if lines > 0 {
		start = fmt.Sprintf("-%d", lines)
	}

	args := []string{"capture-pane"}
	if alternate {
		args = append(args, "-a")
	}
	args = append(args, "-p", "-J", "-S", start, "-E", "-", "-t", target)

	cmd := exec.Command("tmux", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane %s: %w", target, err)
	}
	return string(output), nil
}

// AttachSession replaces the current process with tmux attach.
// WARNING: This function does not return if successful!
func AttachSession(sessionName string) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	// Exec replaces the current process
	args := []string{"tmux", "attach", "-t", sessionName}
	if err := syscall.Exec(tmuxPath, args, os.Environ()); err != nil {
		return fmt.Errorf("failed to attach to session: %w", err)
	}

	return nil // Should not reach here
}

// EnsureSession ensures a session exists (creating it if needed) and then attaches to it.
func EnsureAndAttach(sessionName, cwd string) error {
	if !SessionExists(sessionName) {
		fmt.Printf("Creating new tmux session '%s'...\n", sessionName)
		if err := NewSession(sessionName, cwd); err != nil {
			return err
		}
	} else {
		fmt.Printf("Attaching to existing session '%s'...\n", sessionName)
	}

	return AttachSession(sessionName)
}

// IsInsideTmux checks if the current process is running inside tmux.
func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// GetCurrentSessionName returns the name of the current tmux session.
func GetCurrentSessionName() (string, error) {
	if !IsInsideTmux() {
		return "", fmt.Errorf("not inside tmux")
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#S")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// SwitchClient switches the current tmux client to another session.
func SwitchClient(sessionName string) error {
	cmd := exec.Command("tmux", "switch-client", "-t", sessionName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// KillSession kills a tmux session.
func KillSession(sessionName string) error {
	// Ignore error if session doesn't exist
	if !SessionExists(sessionName) {
		return nil
	}

	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Sometimes has-session returns true but kill fails if race condition, so double check
		if strings.Contains(string(output), "no server running on") {
			return nil
		}
		return fmt.Errorf("failed to kill session: %s: %w", string(output), err)
	}
	return nil
}
