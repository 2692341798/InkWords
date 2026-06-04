package httpx

import (
	"context"
	"net/http"
	"time"
)

// ShutdownableServer describes the minimal shutdown contract shared by HTTP services.
type ShutdownableServer interface {
	Shutdown(context.Context) error
}

// NewServer returns the default HTTP server configuration shared by service-owned entrypoints.
func NewServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}
}

// ShutdownOnContextDone waits for the signal context and performs a bounded graceful shutdown.
func ShutdownOnContextDone(signalContext context.Context, server ShutdownableServer, timeout time.Duration) error {
	<-signalContext.Done()

	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return server.Shutdown(shutdownContext)
}
