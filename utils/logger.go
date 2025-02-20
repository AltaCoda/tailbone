package utils

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

var Logger zerolog.Logger

// InitLogger initializes the global logger with configuration from viper
func InitLogger() {
	// Set time format
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Configure log level
	level := zerolog.InfoLevel // default level
	switch viper.GetString("log.level") {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	case "trace":
		level = zerolog.TraceLevel
	}
	zerolog.SetGlobalLevel(level)

	logFormat := viper.GetString("log.format")
	if logFormat == "json" {
		// Create file writer
		Logger = zerolog.New(os.Stdout)
	} else {
		// Create console writer
		Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:     os.Stdout,
			NoColor: viper.GetBool("log.no_color"),
		}).
			With().
			Timestamp().
			Caller().
			Logger()
	}

}

// GetLogger returns a child logger with additional context
func GetLogger(component string) zerolog.Logger {
	return Logger.With().Str("component", component).Logger()
}

// TSLogWriter implements io.Writer interface to adapt Tailscale logs to zerolog
type TSLogWriter struct {
	logger zerolog.Logger
}

func (w *TSLogWriter) Write(p []byte) (n int, err error) {
	w.logger.Debug().Msg(string(p))
	return len(p), nil
}

// NewTSLogWriter creates a new TSLogWriter that forwards to the given logger
func NewTSLogWriter(logger zerolog.Logger) *TSLogWriter {
	return &TSLogWriter{
		logger: logger.With().Str("source", "tailscale").Logger(),
	}
}
