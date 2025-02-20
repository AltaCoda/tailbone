package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/altacoda/tailbone/core"
	"github.com/altacoda/tailbone/utils"
)

// runStart implements the server start command
func runStart(cmd *cobra.Command, args []string) error {
	// Initialize global logger
	utils.InitLogger()
	logger := utils.GetLogger("server")

	srv, err := core.NewServer()
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	logger.Info().Msg("starting tailbone server")
	return srv.Start()
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Tailbone identity server",
	Long: `Start the Tailbone identity server which provides JWT-based authentication
using Tailscale for user verification.`,
	RunE: runStart,
}

func init() {
	Cmd.AddCommand(startCmd)
	// Server flags
	startCmd.Flags().IntP("port", "p", 80, "Port to run the server on")
	startCmd.Flags().StringP("host", "H", "0.0.0.0", "Host address to bind to")
	startCmd.Flags().String("ts-authkey", "", "Tailscale auth key")
	startCmd.Flags().String("log-level", "info", "Log level (trace, debug, info, warn, error)")
	startCmd.Flags().String("log-format", "console", "Log format (console, json)")
	startCmd.Flags().String("dir", "keys", "Directory containing the JWK files")
	startCmd.Flags().String("issuer", "tailbone", "Issuer name for JWT tokens")
	startCmd.Flags().Duration("expiry", 20*time.Minute, "Token expiry duration")
	startCmd.Flags().String("ts-dir", ".tsnet", "Tailscale state directory")
	startCmd.Flags().String("ts-hostname", "tailbone", "Tailscale hostname")
	// Bind flags to viper
	viper.BindPFlag("server.port", startCmd.Flags().Lookup("port"))
	viper.BindPFlag("server.host", startCmd.Flags().Lookup("host"))
	viper.BindPFlag("server.tailscale.authkey", startCmd.Flags().Lookup("ts-authkey"))
	viper.BindPFlag("log.level", startCmd.Flags().Lookup("log-level"))
	viper.BindPFlag("log.format", startCmd.Flags().Lookup("log-format"))
	viper.BindPFlag("keys.dir", startCmd.Flags().Lookup("dir"))
	viper.BindPFlag("keys.issuer", startCmd.Flags().Lookup("issuer"))
	viper.BindPFlag("keys.expiry", startCmd.Flags().Lookup("expiry"))
	viper.BindPFlag("server.tailscale.dir", startCmd.Flags().Lookup("ts-dir"))
	viper.BindPFlag("server.tailscale.hostname", startCmd.Flags().Lookup("ts-hostname"))

	// Set environment variable bindings
	viper.SetEnvPrefix("TB")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}
