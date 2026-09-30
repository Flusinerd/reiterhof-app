// Command api starts the Reiterhof cloud API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
	"github.com/Flusinerd/reiterhof-app/backend/internal/db"
	"github.com/Flusinerd/reiterhof-app/backend/internal/health"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/presence"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
	"github.com/Flusinerd/reiterhof-app/backend/internal/requests"
	"github.com/Flusinerd/reiterhof-app/backend/internal/scheduler"
	"github.com/Flusinerd/reiterhof-app/backend/internal/weather"
	"github.com/Flusinerd/reiterhof-app/backend/migrations"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.FromEnv()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database unavailable", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		log.Error("migrations failed", "err", err)
		os.Exit(1)
	}

	var jobs sync.WaitGroup
	hub := realtime.NewHub(pool, log)
	jobs.Add(1)
	go func() {
		defer jobs.Done()
		hub.Run(ctx)
	}()
	(&scheduler.Scheduler{Log: log}).Go(ctx, &jobs, presence.StaleJob(pool, log, time.Now))
	if cfg.WeatherEnabled {
		weatherSvc := &weather.Service{
			Pool:      pool,
			Fetcher:   &weather.DWD{},
			Log:       log,
			Now:       time.Now,
			StationID: cfg.WeatherStation,
		}
		sched := &scheduler.Scheduler{Log: log}
		sched.Go(ctx, &jobs, scheduler.Job{
			Name:       "weather-snapshot",
			Schedule:   scheduler.Every(time.Hour),
			RunOnStart: true,
			Run:        weatherSvc.Refresh,
		})
	}

	notifier := push.NewNotifier(pool, push.NewClientFromEnv(), log)
	healthReminders := &health.Reminders{Pool: pool, Notify: notifier, Log: log, Now: time.Now}
	(&scheduler.Scheduler{Log: log}).Go(ctx, &jobs, scheduler.Job{
		Name:       "health-reminders",
		Schedule:   scheduler.Every(15 * time.Minute),
		RunOnStart: true,
		Run:        healthReminders.Run,
	})

	deps := httpapi.Deps{
		Pool: pool, Config: cfg, Log: log, Now: time.Now,
		Notify: notifier,
		Events: hub,
	}
	// Requests publish their changes after commit; Register reads this hook.
	requests.DefaultPublish = func(ctx context.Context, e requests.Event) {
		data := map[string]any{"id": e.RequestID, "kind": e.Kind}
		if err := realtime.Publish(ctx, pool, e.StableID, "request.changed", data); err != nil {
			log.Warn("publish request event", "err", err)
		}
	}
	requestJobs := &scheduler.Scheduler{Log: log}
	for _, job := range requests.NewService(deps).Jobs() {
		requestJobs.Go(ctx, &jobs, job)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.NewHandler(deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("API starting", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server stopped unexpectedly", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown failed", "err", err)
		os.Exit(1)
	}
	jobs.Wait()
	log.Info("API stopped")
}
