package utils

import (
	"os"
	"os/signal"
	"syscall"
)

// WaitForSignal blocks until an OS interrupt signal is received
func WaitForSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
