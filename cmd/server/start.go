package server

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/altacoda/tailbone/core"
	"github.com/altacoda/tailbone/utils"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Tailbone identity server",
	Long: `Start the Tailbone identity server which provides JWT-based authentication
using Tailscale for user verification.`,
	RunE: runStart,
	PreRun: func(cmd *cobra.Command, _ []string) {
		// Bind flags to viper
		viper.BindPFlag("server.port", cmd.Flags().Lookup("port"))
		viper.BindPFlag("server.binding", cmd.Flags().Lookup("binding"))
		viper.BindPFlag("server.tailscale.authkey", cmd.Flags().Lookup("ts-authkey"))
		viper.BindPFlag("server.tailscale.joinTimeout", cmd.Flags().Lookup("ts-join-timeout"))
		viper.BindPFlag("server.tailscale.joinRetry", cmd.Flags().Lookup("ts-join-retry"))
		viper.BindPFlag("keys.issuer", cmd.Flags().Lookup("issuer"))
		viper.BindPFlag("keys.expiry", cmd.Flags().Lookup("expiry"))
		viper.BindPFlag("server.tailscale.dir", cmd.Flags().Lookup("ts-dir"))
		viper.BindPFlag("server.tailscale.hostname", cmd.Flags().Lookup("ts-hostname"))
		viper.BindPFlag("admin.port", cmd.Flags().Lookup("admin-port"))
		viper.BindPFlag("admin.binding", cmd.Flags().Lookup("admin-binding"))
		viper.BindPFlag("components", cmd.Flags().Lookup("components"))
	},
}

// runStart implements the server start command
func runStart(cmd *cobra.Command, args []string) error {
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

	ip, err := tsServer.LocalIp()
	if err != nil {
		return fmt.Errorf("failed to get local IP: %w", err)
	}

	logger.Info().
		Str("ip", ip.String()).
		Msg("Tailscale server started")

	if slices.Contains(components, "issuer") {
		if viper.GetString("server.binding") == "auto" {
			viper.Set("server.binding", ip.String())
		}

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
		if viper.GetString("admin.binding") == "auto" {
			viper.Set("admin.binding", ip.String())
		}
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

func init() {
	Cmd.AddCommand(startCmd)
	// IssuerListener flags
	startCmd.Flags().IntP("port", "p", 80, "Port to run the server on (issuer)")
	startCmd.Flags().StringP("binding", "b", "auto", "Binding address for the server (issuer)")
	startCmd.Flags().String("ts-authkey", "", "Tailscale auth key")
	startCmd.Flags().Duration("ts-join-timeout", 60*time.Second, "Tailscale join timeout")
	startCmd.Flags().Duration("ts-join-retry", 1*time.Second, "Tailscale join retry interval")
	startCmd.Flags().String("issuer", "tailbone", "Issuer name for JWT tokens")
	startCmd.Flags().Duration("expiry", 20*time.Minute, "Token expiry duration")
	startCmd.Flags().String("ts-dir", ".tsnet", "Tailscale state directory")
	startCmd.Flags().String("ts-hostname", "tailbone", "Tailscale hostname")
	startCmd.Flags().String("admin-binding", "auto", "Admin server binding address")
	startCmd.Flags().Int("admin-port", 50051, "Admin server port")
	startCmd.Flags().StringSlice("components", []string{"issuer", "admin"}, "Components to start")
}
