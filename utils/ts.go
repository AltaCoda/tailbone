package utils

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"tailscale.com/tsnet"
)

type TsServer struct {
	logger zerolog.Logger
	server *tsnet.Server
}

func NewTsServer() *TsServer {
	return &TsServer{
		logger: GetLogger("ts"),
	}
}

func (t *TsServer) Start() error {
	logger := t.logger
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
		err = os.MkdirAll(viper.GetString("server.tailscale.dir"), 0755)
		if err != nil {
			return fmt.Errorf("failed to create tailscale dir: %w", err)
		}
	}

	tsLogger := GetLogger("tsnet")

	t.server = &tsnet.Server{
		Hostname:  viper.GetString("server.tailscale.hostname"),
		AuthKey:   viper.GetString("server.tailscale.authkey"),
		Logf:      func(msg string, v ...interface{}) { tsLogger.Trace().Msgf(msg, v...) },
		UserLogf:  func(msg string, v ...interface{}) { tsLogger.Debug().Msgf(msg, v...) },
		Dir:       viper.GetString("server.tailscale.dir"),
		Ephemeral: true,
	}

	return t.server.Start()
}

func (t *TsServer) Stop() {
	if t.server == nil {
		return
	}

	t.server.Close()
}

func (t *TsServer) Server() *tsnet.Server {
	if t.server == nil {
		panic("server not initialized")
	}

	return t.server
}

var _ IServer = &TsServer{}
