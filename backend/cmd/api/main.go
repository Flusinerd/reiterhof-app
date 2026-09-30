// Command api starts the Stallfunk cloud API.
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

	"github.com/Flusinerd/reiterhof-app/backend/internal/blankets"
	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
	"github.com/Flusinerd/reiterhof-app/backend/internal/db"
	"github.com/Flusinerd/reiterhof-app/backend/internal/health"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/presence"
	"github.com/Flusinerd/reiterhof-app/backend/internal/privacy"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
	"github.com/Flusinerd/reiterhof-app/backend/internal/reha"
	"github.com/Flusinerd/reiterhof-app/backend/internal/reminders"
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
	(&scheduler.Scheduler{Log: log}).Go(ctx, &jobs, privacy.Job(pool, log, time.Now))

	native, err := push.NewClientFromEnv(log)
	if err != nil {
		log.Error("native push: credentials rejected, that platform is skipped", "err", err)
	}
	log.Info("native push", "platforms", native.Configured())
	notifier := push.NewNotifier(pool, native, log)
	switch vapid, err := push.VAPIDFromEnv(); {
	case err == nil:
		notifier.WithWeb(push.NewWebClient(vapid))
	case errors.Is(err, push.ErrVAPIDNotConfigured):
		log.Info("web push disabled: no VAPID keys (REITERHOF_VAPID_*)")
	default:
		log.Error("web push disabled: invalid VAPID configuration", "err", err)
	}
	healthReminders := &health.Reminders{Pool: pool, Notify: notifier, Log: log, Now: time.Now}
	(&scheduler.Scheduler{Log: log}).Go(ctx, &jobs, scheduler.Job{
		Name:       "health-reminders",
		Schedule:   scheduler.Every(15 * time.Minute),
		RunOnStart: true,
		Run:        healthReminders.Run,
	})

	rehaReminders := &reha.Reminders{Pool: pool, Notify: notifier, Log: log, Now: time.Now}
	(&scheduler.Scheduler{Log: log}).Go(ctx, &jobs, scheduler.Job{
		Name:       "reha-checkup-reminders",
		Schedule:   scheduler.Every(15 * time.Minute),
		RunOnStart: true,
		Run:        rehaReminders.Run,
	})

	deps := httpapi.Deps{
		Pool: pool, Config: cfg, Log: log, Now: time.Now,
		Notify: notifier,
		Events: hub,
	}
	blanketSvc := blankets.NewService(deps)
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
			Run: func(ctx context.Context) error {
				err := weatherSvc.Refresh(ctx)
				// A fresh forecast may change tonight's recommendation (weather change push).
				return errors.Join(err, blanketSvc.CheckWeatherChange(ctx))
			},
		})
	}
	// The last person leaving the stable gets an immediate blanket reminder.
	presence.AfterCheckOut = func(ctx context.Context, stableID, userID string) {
		go blanketSvc.OnCheckOut(context.WithoutCancel(ctx), stableID, userID)
	}
	blanketJobs := &scheduler.Scheduler{Log: log}
	for _, job := range blanketSvc.Jobs() {
		blanketJobs.Go(ctx, &jobs, job)
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

	reminderJobs := &scheduler.Scheduler{Log: log}
	for _, job := range reminders.NewService(deps).Jobs() {
		reminderJobs.Go(ctx, &jobs, job)
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
