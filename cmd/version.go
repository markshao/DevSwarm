package cmd

import (
	"fmt"

	"devswarm/internal/version"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of DevSwarm",
	Long:  `All software has versions. This is DevSwarm's.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("DevSwarm version %s\n", version.Version)
		fmt.Printf("Commit: %s\n", version.Commit)
		fmt.Printf("Date: %s\n", version.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
