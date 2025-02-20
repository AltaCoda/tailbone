package cmd

import (
	"fmt"

	"github.com/altacoda/tailbone/utils"
	"github.com/spf13/cobra"
)

// runVersion implements the version command
func runVersion(_ *cobra.Command, _ []string) error {
	fmt.Printf("Version %s\n", utils.Version)
	fmt.Printf("Commit %s\n", utils.Commit)
	return nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runVersion(cmd, args); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
