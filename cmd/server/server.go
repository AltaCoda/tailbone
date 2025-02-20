package server

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "Server management commands",
	Long: `Server management commands for the Tailbone identity server.
These commands allow you to start, stop, and manage the server.`,
}
