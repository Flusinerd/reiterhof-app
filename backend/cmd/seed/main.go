// Command seed loads the example data (Stallgasse B) into the database.
// It is idempotent. Migrations are applied first.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
	"github.com/Flusinerd/reiterhof-app/backend/internal/db"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
	"github.com/Flusinerd/reiterhof-app/backend/migrations"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(context.Background(), log); err != nil {
		log.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	pool, err := db.Open(ctx, config.FromEnv().DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		return err
	}
	if err := seed.Run(ctx, pool); err != nil {
		return err
	}
	log.Info("seed applied")
	return nil
}
