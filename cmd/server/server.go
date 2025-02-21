package server

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "Server management commands",
	Long: `Server management commands for the Tailbone identity server.
These commands allow you to start, stop, and manage the server.`,
}

func init() {
	Cmd.PersistentFlags().String("log-level", "info", "Log level (trace, debug, info, warn, error)")
	Cmd.PersistentFlags().String("log-format", "console", "Log format (console, json)")
	Cmd.PersistentFlags().String("dir", "keys", "Directory containing the JWK files")
	Cmd.PersistentFlags().String("bucket", "", "S3 bucket for JWKS storage")
	Cmd.PersistentFlags().String("key-path", ".well-known/jwks.json", "Path/key for the JWKS file in S3")

	viper.BindPFlag("log.level", Cmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log.format", Cmd.PersistentFlags().Lookup("log-format"))
	viper.BindPFlag("keys.dir", Cmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("keys.bucket", Cmd.PersistentFlags().Lookup("bucket"))
	viper.BindPFlag("keys.keyPath", Cmd.PersistentFlags().Lookup("key-path"))
}
