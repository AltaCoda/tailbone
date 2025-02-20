package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"tailscale.com/tsnet"

	"github.com/altacoda/tailbone/utils"
)

type Server struct {
	server utils.IServer
	client utils.IClient
	issuer Issuer
	logger zerolog.Logger
}

func NewServer() (*Server, error) {
	// Configure global logger
	logger := utils.GetLogger("server")

	issuer, err := NewTokenIssuer(context.Background(), IssuerConfig{
		KeyDir: viper.GetString("keys.dir"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create token issuer: %w", err)
	}

	return &Server{
		issuer: issuer,
		logger: logger,
	}, nil
}

func (s *Server) Start() error {
	logger := s.logger
	logger.Info().Msg("initializing tailscale server")

	if viper.GetString("server.tailscale.authkey") == "" {
		return fmt.Errorf("tailscale auth key is not set")
	}

	if viper.GetString("server.tailscale.dir") == "" {
		return fmt.Errorf("tailscale state directory is not set")
	}

	if viper.GetString("server.tailscale.hostname") == "" {
		return fmt.Errorf("tailscale hostname is not set")
	}

	// if tsdir does not exist, create it
	if _, err := os.Stat(viper.GetString("server.tailscale.dir")); os.IsNotExist(err) {
		os.MkdirAll(viper.GetString("server.tailscale.dir"), 0755)
	}

	tsLogger := utils.GetLogger("tsnet")

	s.server = &tsnet.Server{
		Hostname: viper.GetString("server.tailscale.hostname"),
		AuthKey:  viper.GetString("server.tailscale.authkey"),
		Logf:     func(msg string, v ...interface{}) { tsLogger.Trace().Msgf(msg, v...) },
		UserLogf: func(msg string, v ...interface{}) { tsLogger.Debug().Msgf(msg, v...) },
		Dir:      viper.GetString("server.tailscale.dir"),
	}

	// Create local client for Tailscale operations
	var err error
	s.client, err = s.server.LocalClient()
	if err != nil {
		return fmt.Errorf("failed to create local client: %w", err)
	}

	addr := fmt.Sprintf(":%d", viper.GetInt("server.port"))
	logger.Info().Str("addr", addr).Msg("creating listener")

	ln, err := s.server.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create signal handler for graceful shutdown
	go func() {
		utils.WaitForSignal()
		logger.Info().Msg("received shutdown signal")
		cancel()
	}()

	defer cancel()

	// Create HTTP server
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			reqLogger := logger.With().
				Str("remote_addr", r.RemoteAddr).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Logger()

			reqLogger.Info().Msg("handling request")

			switch r.URL.Path {
			case "/_healthz":
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ok":      true,
					"version": utils.Version,
					"commit":  utils.Commit,
				})
				return

			case "/issue":
				// Only allow POST requests
				if r.Method != http.MethodPost {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				// Authenticate and issue token
				who, err := s.client.WhoIs(ctx, r.RemoteAddr)
				if err != nil {
					reqLogger.Error().Err(err).Msg("failed to identify user")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				token, err := s.issuer.IssueToken(ctx, who.UserProfile.LoginName, who.UserProfile.DisplayName)
				if err != nil {
					reqLogger.Error().Err(err).Msg("failed to issue token")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				reqLogger.Info().
					Str("user", who.UserProfile.LoginName).
					Str("display_name", who.UserProfile.DisplayName).
					Msg("issued token")

				json.NewEncoder(w).Encode(map[string]string{
					"token": token,
				})

			default:
				http.NotFound(w, r)
			}
		}),
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		logger.Info().Str("addr", addr).Msg("starting server")
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error().Err(err).Msg("server error")
			errChan <- err
		}
		close(errChan)
	}()

	// Wait for either context cancellation or server error
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		logger.Info().Msg("shutting down server")

		// Shutdown server gracefully
		if err := server.Shutdown(context.Background()); err != nil {
			logger.Error().Err(err).Msg("error during shutdown")
			return err
		}
	}

	logger.Info().Msg("server shutdown complete")
	return nil
}
