package cmd

import (
	"fmt"
	"os"
	"strings"
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

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NODE\tLABEL\tSESSION\tPANE\tSTATE\tPENDING\tMUTED\tLAST CHANGE\tLAST NOTIFY\tNOTIFY COUNT\tREASON\tLAST BLOCK")
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
			pending := "-"
			if notification.HasPendingWaitEvent(watcher) {
				pending = "wait-input"
			}
			muted := "-"
			if watcher.MutedWaitEventID >= watcher.WaitEventID && watcher.WaitEventID > 0 {
				muted = "muted"
			}
			lastBlock := "-"
			if watcher.LastAgentBlock != "" {
				lastBlock = strings.ReplaceAll(watcher.LastAgentBlock, "\n", "\\n")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
				watcher.NodeName,
				label,
				watcher.SessionName,
				watcher.PaneID,
				watcher.State,
				pending,
				muted,
				lastChange,
				lastNotify,
				watcher.NotifyCount,
				watcher.LastReason,
				lastBlock,
			)
		}
		w.Flush()
		return nil
	},
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
	notificationServiceCmd.AddCommand(notificationServiceListWatchersCmd)
	notificationServiceCmd.AddCommand(notificationServiceCleanWatchersCmd)
	notificationServiceCmd.AddCommand(notificationServiceRunCmd)
	rootCmd.AddCommand(notificationServiceCmd)
}
