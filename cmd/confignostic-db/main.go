// confignostic-db applies the embedded forward-only database migrations.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/UXCentury/confignostic/internal/config"
	"github.com/UXCentury/confignostic/internal/storage"
)

func main() { os.Exit(run()) }
func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if len(os.Args) != 2 || os.Args[1] != "migrate" {
		logger.Error("usage: confignostic-db migrate")
		return 1
	}
	cfg, err := config.Load(os.LookupEnv)
	if err != nil || cfg.DatabaseURL == "" {
		logger.Error("set a valid CONFIGNOSTIC_DATABASE_URL")
		return 1
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, time.Minute)
	defer cancel()
	db, err := storage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed")
		return 1
	}
	defer db.Close()
	if err = db.Migrate(ctx); err != nil {
		logger.Error("database migration failed; verify connectivity and schema compatibility")
		return 1
	}
	logger.Info("database migrations complete", "schema_version", 2)
	return 0
}
