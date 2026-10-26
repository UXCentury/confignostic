// Package server owns the HTTP server lifecycle, including request draining.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Serve takes ownership of listener and returns only after shutdown completes.
// Cancellation stops admission without canceling active request contexts. A drain
// deadline force-closes outstanding connections and returns an error to the caller.
func Serve(ctx context.Context, listener net.Listener, handler http.Handler, shutdownTimeout time.Duration, logger *slog.Logger) error {
	defer listener.Close()
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(listener) }()

	var serveErr error
	var finished bool
	select {
	case <-ctx.Done():
	case serveErr = <-served:
		finished = true
	}

	logger.Info("API draining", "timeout", shutdownTimeout.String())
	// The signal context is canceled; use a fresh deadline for active requests.
	drainCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	shutdownErr := srv.Shutdown(drainCtx)
	if shutdownErr != nil {
		shutdownErr = errors.Join(fmt.Errorf("HTTP drain failed: %w", shutdownErr), srv.Close())
	}
	if !finished {
		serveErr = <-served
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	if err := errors.Join(serveErr, shutdownErr); err != nil {
		return err
	}
	logger.Info("API stopped")
	return nil
}
