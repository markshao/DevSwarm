package cmd

import (
	"fmt"
	"os"

	"orion/internal/workflow"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm [node_names...]",
	Short: "Remove one or more development nodes",
	Long: `Removes one or more nodes and cleans up their resources:
- Kills the tmux session
- Removes the git worktree
- Deletes the shadow branch
- Updates the state file

Examples:
  orion rm node1 node2 node3
  orion ls | awk '{print $1}' | xargs orion rm -f`,
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: CompleteNodeNames,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(cwd)
		if err != nil {
			fmt.Printf("Failed to load workspace: %v\n", err)
			os.Exit(1)
		}

		var nodeNames []string
		if len(args) == 0 {
			// Interactive mode: select a single node
			nodeName, err := SelectNode(wm, "remove", true)
			if err != nil {
				fmt.Printf("%v\n", err)
				return
			}
			nodeNames = append(nodeNames, nodeName)
		} else {
			nodeNames = args
		}

		// Collect unapplied workflow runs for all nodes (for warning display)
		if !force {
			engine := workflow.NewEngine(wm)
			runs, _ := engine.ListRuns()

			nodeWarnings := make(map[string][]string)
			for _, nodeName := range nodeNames {
				if node, exists := wm.State.Nodes[nodeName]; exists {
					var unapplied []string
					for _, run := range runs {
						if run.TriggeredByNode == nodeName && run.Status == workflow.StatusSuccess {
							isApplied := false
							for _, appliedID := range node.AppliedRuns {
								if appliedID == run.ID {
									isApplied = true
									break
								}
							}
							if !isApplied {
								unapplied = append(unapplied, run.ID)
							}
						}
					}
					if len(unapplied) > 0 {
						nodeWarnings[nodeName] = unapplied
					}
				}
			}

			if len(nodeWarnings) > 0 {
				fmt.Println()
				for nodeName, unapplied := range nodeWarnings {
					color.Yellow("Warning: Node '%s' has %d unapplied successful workflow run(s).", nodeName, len(unapplied))
					fmt.Printf("  Unapplied runs: %v\n", unapplied)
				}
				fmt.Println()
				fmt.Print("Are you sure you want to remove these node(s)? [y/N]: ")
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Aborted.")
					return
				}
			}
		}

		// Process removal
		var failed []string
		for _, nodeName := range nodeNames {
			if _, exists := wm.State.Nodes[nodeName]; !exists {
				fmt.Printf("❌ Node '%s' not found.\n", nodeName)
				failed = append(failed, nodeName)
				continue
			}

			if len(nodeNames) == 1 {
				fmt.Printf("Removing node '%s'...\n", nodeName)
			}
			if err := wm.RemoveNode(nodeName); err != nil {
				fmt.Printf("❌ Failed to remove node '%s': %v\n", nodeName, err)
				failed = append(failed, nodeName)
			} else {
				if len(nodeNames) > 1 {
					fmt.Printf("✅ Removed '%s'\n", nodeName)
				} else {
					fmt.Printf("✅ Node '%s' removed successfully.\n", nodeName)
				}
			}
		}

		if len(failed) > 0 {
			os.Exit(1)
		}
	},
}

func init() {
	rmCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
	rootCmd.AddCommand(rmCmd)
}
