package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `Print version information for ky.

Displays the version, commit hash, and build date.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ky %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  date:   %s\n", date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
