package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// WaitForSignal blocks until an OS interrupt signal is received
func WaitForSignal(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigChan:
	case <-ctx.Done():
	}
}
