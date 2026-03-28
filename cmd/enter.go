package cmd

import (
	"fmt"
	"os"

	"orion/internal/notification"
	"orion/internal/types"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var enterCmd = &cobra.Command{
	Use:   "enter [node_name]",
	Short: "Enter a node's development environment",
	Long: `Starts or attaches to the tmux session for the specified node.

Features:
  - If [node_name] is provided, it enters that node directly.
  - If [node_name] is OMITTED, an INTERACTIVE MENU will appear to let you select a node.
  - Supports Shell Tab Completion for node names.

If you are already inside tmux, it will switch the current client.
If not, it will start a new client.`,
	Args:              cobra.RangeArgs(0, 1),
	ValidArgsFunction: CompleteNodeNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}

		wm, err := workspace.NewManager(cwd)
		if err != nil {
			return fmt.Errorf("load workspace: %w", err)
		}

		var nodeName string
		if len(args) == 0 {
			// Interactive mode
			var err error
			nodeName, err = SelectNode(wm, "enter", true)
			if err != nil {
				color.Yellow("%v", err)
				return nil
			}
		} else {
			nodeName = args[0]
			// Check if it is an agent node
			if node, exists := wm.State.Nodes[nodeName]; exists && node.CreatedBy != types.NodeCreatedByUser {
				return fmt.Errorf("node '%s' is an agent node; use `orion workflow enter`", nodeName)
			}
		}

		fmt.Printf("Entering node '%s'...\n", nodeName)
		sessionName, err := wm.EnsureNodeSession(nodeName)
		if err != nil {
			return fmt.Errorf("prepare node session: %w", err)
		}

		if err := notification.EnsureStarted(wm.RootPath); err != nil {
			color.Yellow("Warning: Failed to start notification service: %v", err)
		} else if err := notification.EnsureWatcher(wm.RootPath, nodeName, wm.State.Nodes[nodeName].Label, sessionName); err != nil {
			color.Yellow("Warning: Failed to register notification watcher: %v", err)
		} else if err := notification.AcknowledgeWaitEvent(wm.RootPath, nodeName); err != nil {
			color.Yellow("Warning: Failed to clear pending wait-input state: %v", err)
		}

		if err := wm.AttachNodeSession(nodeName); err != nil {
			return fmt.Errorf("enter node: %w", err)
		}

		// Note: If successful, the process is replaced by tmux, so this won't print.
		return nil
	},
}

func init() {
	rootCmd.AddCommand(enterCmd)
}
