package server

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/altacoda/tailbone/core"
	"github.com/altacoda/tailbone/utils"
)

// runStart implements the server start command
func runStart(cmd *cobra.Command, args []string) error {
	// Bind flags to viper
	viper.BindPFlag("server.port", cmd.Flags().Lookup("port"))
	viper.BindPFlag("server.host", cmd.Flags().Lookup("host"))
	viper.BindPFlag("server.tailscale.authkey", cmd.Flags().Lookup("ts-authkey"))
	viper.BindPFlag("log.level", cmd.Flags().Lookup("log-level"))
	viper.BindPFlag("log.format", cmd.Flags().Lookup("log-format"))
	viper.BindPFlag("keys.dir", cmd.Flags().Lookup("dir"))
	viper.BindPFlag("keys.issuer", cmd.Flags().Lookup("issuer"))
	viper.BindPFlag("keys.expiry", cmd.Flags().Lookup("expiry"))
	viper.BindPFlag("keys.bucket", cmd.Flags().Lookup("bucket"))
	viper.BindPFlag("keys.keyPath", cmd.Flags().Lookup("key-path"))
	viper.BindPFlag("server.tailscale.dir", cmd.Flags().Lookup("ts-dir"))
	viper.BindPFlag("server.tailscale.hostname", cmd.Flags().Lookup("ts-hostname"))
	viper.BindPFlag("admin.address", cmd.Flags().Lookup("admin-address"))
	viper.BindPFlag("components", cmd.Flags().Lookup("components"))

	utils.InitLogger()
	logger := utils.GetLogger("server")

	ctx := context.Background()
	logger.Info().Msg("starting tailbone server")

	var servers []utils.IServer
	components := viper.GetStringSlice("components")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tsServer := utils.NewTsServer()
	err := tsServer.Start()
	if err != nil {
		return fmt.Errorf("failed to start Tailscale server: %w", err)
	}

	if slices.Contains(components, "issuer") {
		srv, err := core.NewIssuerListener(tsServer.Server())
		if err != nil {
			return fmt.Errorf("failed to create issuer listener: %w", err)
		}
		go func() {
			if err := srv.Start(); err != nil {
				logger.Error().Err(err).Msg("issuer listener error")
				cancel()
				return
			}
		}()
		servers = append(servers, srv)
	}

	if slices.Contains(components, "admin") {
		adminSrv, err := core.NewAdminListener(ctx, tsServer.Server())
		if err != nil {
			return fmt.Errorf("failed to create admin listener: %w", err)
		}
		go func() {
			if err := adminSrv.Start(); err != nil {
				logger.Error().Err(err).Msg("admin listener error")
				cancel()
				return
			}
		}()
		servers = append(servers, adminSrv)
	}

	utils.WaitForSignal(ctx)
	logger.Info().Msg("shutting down listeners")

	// Stop all servers
	for _, srv := range servers {
		srv.Stop()
	}

	logger.Info().Msg("shutting down Tailscale server")
	tsServer.Stop()

	return nil
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
	// IssuerListener flags
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
	startCmd.Flags().String("admin-address", ":50051", "Address of the admin server")
	startCmd.Flags().StringSlice("components", []string{"issuer", "admin"}, "Components to start")
	startCmd.Flags().String("bucket", "", "S3 bucket for JWKS storage")
	startCmd.Flags().String("key-path", ".well-known/jwks.json", "Path/key for the JWKS file in S3")

	// Set environment variable bindings
	viper.SetEnvPrefix("TB")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
}
