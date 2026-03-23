package notification

import (
	"fmt"
	"os/exec"
	"runtime"
)

var lookPath = exec.LookPath

func NotifyWatcher(nodeName, reason string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("mac notifications are only supported on darwin")
	}

	body := "Waiting for input"
	if reason != "" {
		body = fmt.Sprintf("Waiting for input: %s", reason)
	}

	cmd, err := buildNotificationCommand(nodeName, body)
	if err != nil {
		return err
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to send mac notification: %s: %w", string(output), err)
	}
	return nil
}

func buildNotificationCommand(nodeName, body string) (*exec.Cmd, error) {
	subtitle := fmt.Sprintf("Node %s", nodeName)

	if _, err := lookPath("terminal-notifier"); err == nil {
		cmd := exec.Command("terminal-notifier",
			"-title", "Orion",
			"-subtitle", subtitle,
			"-message", body,
			"-sound", "default",
		)
		return cmd, nil
	}

	script := fmt.Sprintf("display notification %q with title %q subtitle %q", body, "Orion", subtitle)
	if _, err := lookPath("osascript"); err == nil {
		return exec.Command("osascript", "-e", script), nil
	}

	return nil, fmt.Errorf("no supported mac notification command found")
}
