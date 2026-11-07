package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/UXCentury/confignostic/internal/config"
	"github.com/UXCentury/confignostic/internal/httpapi"
	"github.com/UXCentury/confignostic/internal/server"
	"github.com/UXCentury/confignostic/internal/storage"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var ready func(context.Context) error
	if cfg.DatabaseURL != "" {
		startup, cancel := context.WithTimeout(ctx, 10*time.Second)
		db, err := storage.Open(startup, cfg.DatabaseURL)
		if err == nil {
			err = db.Ready(startup)
		}
		cancel()
		if err != nil {
			if db != nil {
				db.Close()
			}
			logger.Error("database startup failed; check connectivity and run database migrations")
			return 1
		}
		defer db.Close()
		ready = db.Ready
	}
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		logger.Error("API listen failed", "error", err)
		return 1
	}
	logger.Info("API listening", "address", listener.Addr().String())
	if err := server.Serve(ctx, listener, httpapi.NewHandlerWithReadiness(ready), cfg.ShutdownTimeout, logger); err != nil {
		logger.Error("API stopped with an error", "error", err)
		return 1
	}
	return 0
}
