package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"orion/internal/notification"
	"orion/internal/workspace"

	"github.com/spf13/cobra"
)

func resolveWorkspaceRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return workspace.FindWorkspaceRoot(cwd)
}

var notificationServiceCmd = &cobra.Command{
	Use:   "notification-service",
	Short: "Manage the Orion notification service",
}

var notificationServiceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the workspace notification service",
	RunE: func(cmd *cobra.Command, args []string) error {
		rootPath, err := resolveWorkspaceRoot()
		if err != nil {
			return fmt.Errorf("resolve workspace: %w", err)
		}
		if err := notification.EnsureStarted(rootPath); err != nil {
			return fmt.Errorf("start notification service: %w", err)
		}
		fmt.Println("Notification service is running.")
		return nil
	},
}

var notificationServiceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the workspace notification service",
	RunE: func(cmd *cobra.Command, args []string) error {
		rootPath, err := resolveWorkspaceRoot()
		if err != nil {
			return fmt.Errorf("resolve workspace: %w", err)
		}
		if err := notification.Stop(rootPath); err != nil {
			return fmt.Errorf("stop notification service: %w", err)
		}
		fmt.Println("Notification service stopped.")
		return nil
	},
}

var notificationServiceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show notification service status",
	RunE: func(cmd *cobra.Command, args []string) error {
		rootPath, err := resolveWorkspaceRoot()
		if err != nil {
			return fmt.Errorf("resolve workspace: %w", err)
		}

		status, running, err := notification.GetServiceStatus(rootPath)
		if err != nil {
			return fmt.Errorf("load notification service status: %w", err)
		}

		registry, err := notification.ReadRegistry(rootPath)
		if err != nil {
			return fmt.Errorf("load watcher registry: %w", err)
		}

		fmt.Printf("Workspace: %s\n", rootPath)
		fmt.Printf("Status:    %s\n", map[bool]string{true: "running", false: "stopped"}[running])
		fmt.Printf("PID:       %d\n", status.PID)
		if !status.StartedAt.IsZero() {
			fmt.Printf("Started:   %s\n", status.StartedAt.Format(time.RFC3339))
		}
		if !status.LastLoopAt.IsZero() {
			fmt.Printf("Last Loop: %s\n", status.LastLoopAt.Format(time.RFC3339))
		}
		fmt.Printf("Watchers:  %d\n", len(registry.Watchers))
		if status.LastError != "" {
			fmt.Printf("Last Err:  %s\n", status.LastError)
		}
		return nil
	},
}

var notificationServiceListWatchersCmd = &cobra.Command{
	Use:   "list-watchers",
	Short: "List registered notification watchers",
	RunE: func(cmd *cobra.Command, args []string) error {
		rootPath, err := resolveWorkspaceRoot()
		if err != nil {
			return fmt.Errorf("resolve workspace: %w", err)
		}

		registry, err := notification.ReadRegistry(rootPath)
		if err != nil {
			return fmt.Errorf("load watcher registry: %w", err)
		}
		if len(registry.Watchers) == 0 {
			fmt.Println("No watchers registered.")
			return nil
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		showAgentBlock, _ := cmd.Flags().GetBool("show-agent-block")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if verbose {
			if showAgentBlock {
				fmt.Fprintln(w, "NODE\tLABEL\tSESSION\tPANE\tSTATE\tWAIT EVENT\tSHOULD NOTIFY\tLAST CHANGE\tLAST NOTIFY\tNOTIFY COUNT\tREASON\tLAST BLOCK")
			} else {
				fmt.Fprintln(w, "NODE\tLABEL\tSESSION\tPANE\tSTATE\tWAIT EVENT\tSHOULD NOTIFY\tLAST CHANGE\tLAST NOTIFY\tNOTIFY COUNT\tREASON")
			}
		} else {
			fmt.Fprintln(w, "NODE\tSTATE\tLAST CHANGE\tWAIT EVENT\tSHOULD NOTIFY")
		}
		for _, watcher := range registry.Watchers {
			lastChange := "-"
			if !watcher.LastChangeAt.IsZero() {
				lastChange = watcher.LastChangeAt.Format("01-02 15:04:05")
			}
			lastNotify := "-"
			if !watcher.LastNotifyAt.IsZero() {
				lastNotify = watcher.LastNotifyAt.Format("01-02 15:04:05")
			}
			label := watcher.Label
			if label == "" {
				label = "-"
			}
			hasPendingWaitEvent := notification.HasPendingWaitEvent(watcher)
			isMuted := watcher.MutedWaitEventID >= watcher.WaitEventID && watcher.WaitEventID > 0
			waitEventStatus := "-"
			switch {
			case isMuted:
				waitEventStatus = "muted"
			case hasPendingWaitEvent:
				waitEventStatus = "pending"
			}
			shouldNotify := "no"
			if hasPendingWaitEvent && !isMuted {
				shouldNotify = "yes"
			}
			if verbose {
				if showAgentBlock {
					lastBlock := "-"
					if watcher.LastAgentBlock != "" {
						lastBlock = formatSingleLine(watcher.LastAgentBlock)
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
						watcher.NodeName,
						label,
						watcher.SessionName,
						watcher.PaneID,
						watcher.State,
						waitEventStatus,
						shouldNotify,
						lastChange,
						lastNotify,
						watcher.NotifyCount,
						formatSingleLine(watcher.LastReason),
						lastBlock,
					)
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
						watcher.NodeName,
						label,
						watcher.SessionName,
						watcher.PaneID,
						watcher.State,
						waitEventStatus,
						shouldNotify,
						lastChange,
						lastNotify,
						watcher.NotifyCount,
						formatSingleLine(watcher.LastReason),
					)
				}
				continue
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				watcher.NodeName,
				watcher.State,
				lastChange,
				waitEventStatus,
				shouldNotify,
			)
		}
		w.Flush()
		return nil
	},
}

func formatSingleLine(s string) string {
	if s == "" {
		return "-"
	}
	return flattenMultiline(s)
}

func flattenMultiline(s string) string {
	if s == "" {
		return s
	}
	out := make([]rune, 0, len(s))
	lastWasSpace := false
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			if !lastWasSpace {
				out = append(out, ' ')
				lastWasSpace = true
			}
			continue
		}
		out = append(out, r)
		lastWasSpace = false
	}
	return string(out)
}

var notificationServiceCleanWatchersCmd = &cobra.Command{
	Use:   "clean-watchers",
	Short: "Clean all registered notification watchers",
	RunE: func(cmd *cobra.Command, args []string) error {
		rootPath, err := resolveWorkspaceRoot()
		if err != nil {
			return fmt.Errorf("resolve workspace: %w", err)
		}

		removed, err := notification.ClearWatchers(rootPath)
		if err != nil {
			return fmt.Errorf("clean watcher registry: %w", err)
		}

		fmt.Printf("Cleaned %d watcher(s).\n", removed)
		return nil
	},
}

var notificationServiceRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the workspace notification service loop",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		rootPath, _ := cmd.Flags().GetString("workspace")
		if rootPath == "" {
			return fmt.Errorf("--workspace is required")
		}
		if err := notification.Run(rootPath); err != nil {
			return fmt.Errorf("notification service failed: %w", err)
		}
		return nil
	},
}

func init() {
	notificationServiceRunCmd.Flags().String("workspace", "", "Workspace root path")

	notificationServiceCmd.AddCommand(notificationServiceStartCmd)
	notificationServiceCmd.AddCommand(notificationServiceStopCmd)
	notificationServiceCmd.AddCommand(notificationServiceStatusCmd)
	notificationServiceListWatchersCmd.Flags().BoolP("verbose", "v", false, "Show detailed watcher fields")
	notificationServiceListWatchersCmd.Flags().Bool("show-agent-block", false, "Include extracted agent block (verbose output)")
	notificationServiceCmd.AddCommand(notificationServiceListWatchersCmd)
	notificationServiceCmd.AddCommand(notificationServiceCleanWatchersCmd)
	notificationServiceCmd.AddCommand(notificationServiceRunCmd)
	rootCmd.AddCommand(notificationServiceCmd)
}
