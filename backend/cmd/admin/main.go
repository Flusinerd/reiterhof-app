// Command admin is stallfunk-admin, the operator tool of the Stallfunk backend: first
// stable and admins, invite codes, user and horse maintenance, test mail, weather refresh
// and migrations. It reads the same environment as the API (REITERHOF_DATABASE_URL, ...),
// optionally from a systemd EnvironmentFile (--env-file, default /etc/reiterhof/api.env).
// The commands live in internal/admincli. Run "stallfunk-admin help" for the overview.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/admincli"
	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
	"github.com/Flusinerd/reiterhof-app/backend/internal/db"
	"github.com/Flusinerd/reiterhof-app/backend/migrations"

	// The static binary must find time zones (stable --timezone) without system tzdata.
	_ "time/tzdata"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	env := &admincli.Env{In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Migrations: migrations.FS}

	fs := flag.NewFlagSet("stallfunk-admin", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { admincli.Usage(os.Stderr) }
	envFile := fs.String("env-file", "", "systemd EnvironmentFile to read (default "+admincli.DefaultEnvFile+" if it exists)")
	fs.BoolVar(&env.JSON, "json", false, "machine-readable output for list commands")
	fs.BoolVar(&env.Yes, "yes", false, "skip confirmations")
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	args := fs.Args()

	// "help" and an empty command line must work without any configuration.
	if len(args) > 0 && args[0] != "help" {
		var err error
		if *envFile != "" {
			_, err = admincli.LoadEnvFile(*envFile, false)
		} else {
			_, err = admincli.LoadEnvFile(admincli.DefaultEnvFile, true)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
	}
	env.Config = config.FromEnv()
	env.Open = func(ctx context.Context) (*pgxpool.Pool, error) {
		octx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		pool, err := db.Open(octx, env.Config.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("cannot reach the database (REITERHOF_DATABASE_URL): %w", err)
		}
		return pool, nil
	}

	err := admincli.Run(ctx, env, args)
	if env.Pool != nil {
		env.Pool.Close()
	}
	switch {
	case err == nil:
		return 0
	case errors.Is(err, admincli.ErrUsage):
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	case errors.Is(err, admincli.ErrAborted):
		fmt.Fprintln(os.Stderr, "aborted, nothing changed")
		return 1
	default:
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
}
