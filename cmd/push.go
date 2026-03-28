package cmd

import (
	"fmt"
	"os"

	"orion/internal/git"
	"orion/internal/types"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [node_name]",
	Short: "Push a node's branch to remote repository",
	Long: `Push a node's shadow branch to the remote repository.

This command must be run from the Orion workspace root.

Examples:
  # Push a specific node
  orion push my-feature

  # Select a node interactively
  orion push`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: CompleteHumanNodeNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}

		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			return fmt.Errorf("not in an orion workspace: %w", err)
		}

		if cwd != rootPath {
			return fmt.Errorf("`orion push` must run from workspace root: %s", rootPath)
		}

		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			return fmt.Errorf("load workspace: %w", err)
		}

		// Determine target node
		var targetNodeName string

		if len(args) > 0 {
			targetNodeName = args[0]
			node, exists := wm.State.Nodes[targetNodeName]
			if !exists {
				return fmt.Errorf("node '%s' does not exist", targetNodeName)
			}
			if node.CreatedBy != types.NodeCreatedByUser {
				return fmt.Errorf("node '%s' is not a human node and cannot be pushed", targetNodeName)
			}
		} else {
			selectedName, err := SelectNode(wm, "push", true)
			if err != nil {
				color.Yellow("%v", err)
				return nil
			}
			targetNodeName = selectedName
		}
		targetNode := wm.State.Nodes[targetNodeName]

		// Push the branch
		fmt.Printf("Pushing branch '%s' to remote...\n", targetNode.ShadowBranch)

		if err := git.PushBranch(wm.State.RepoPath, targetNode.ShadowBranch); err != nil {
			return fmt.Errorf("push branch: %w", err)
		}

		color.Green("🚀 Successfully pushed '%s' to remote", targetNodeName)
		fmt.Printf("Branch: %s\n", targetNode.ShadowBranch)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
