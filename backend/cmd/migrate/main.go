// Command migrate applies all pending SQL migrations.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
	"github.com/Flusinerd/reiterhof-app/backend/internal/db"
	"github.com/Flusinerd/reiterhof-app/backend/migrations"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	pool, err := db.Open(ctx, config.FromEnv().DatabaseURL)
	if err == nil {
		defer pool.Close()
		err = db.Migrate(ctx, pool, migrations.FS)
	}
	if err != nil {
		log.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	log.Info("migrations applied")
}
