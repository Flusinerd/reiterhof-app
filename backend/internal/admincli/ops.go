package admincli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/blankets"
	"github.com/Flusinerd/reiterhof-app/backend/internal/db"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/stables"
	"github.com/Flusinerd/reiterhof-app/backend/internal/weather"
)

func runSessionsRevoke(ctx context.Context, e *Env, args []string) error {
	pos, err := parse(e.flags("sessions revoke"), args, 1, 1)
	if err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	u, err := findUser(ctx, pool, pos[0])
	if err != nil {
		return err
	}
	tag, err := pool.Exec(ctx, `DELETE FROM auth_sessions WHERE user_id = $1`, u.ID)
	if err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "revoked %d session(s) of %s; they must sign in again\n", tag.RowsAffected(), u.label())
	return nil
}

func runMailTest(ctx context.Context, e *Env, args []string) error {
	fs := e.flags("mail test")
	to := fs.String("to", "", "recipient address (required)")
	if _, err := parse(fs, args, 0, 0); err != nil {
		return err
	}
	if *to == "" {
		return usagef("--to is required")
	}
	rcpt, err := validEmail(*to)
	if err != nil {
		return err
	}
	cfg := e.Config.Auth
	mailer := e.Mailer
	if mailer == nil {
		// auth.NewMailer would silently fall back to logging; a test mail must really be sent.
		if cfg.SMTPHost == "" {
			return errors.New("REITERHOF_SMTP_HOST is not set: login mails would only be logged, nothing to test (check /etc/reiterhof/api.env)")
		}
		if cfg.SMTPFrom == "" {
			return errors.New("REITERHOF_SMTP_FROM is not set (e.g. Stallfunk <login@example.org>)")
		}
		mailer = auth.NewMailer(cfg, e.log())
	}
	host, _ := os.Hostname()
	msg, err := auth.MailContent{
		Preheader: "Der Mailversand funktioniert.",
		Heading:   "Testnachricht",
		Intro: []string{
			"Testmail von stallfunk-admin. Wenn du sie liest, funktioniert der Mailversand.",
		},
		Note: "Gesendet am " + e.now().UTC().Format("2006-01-02 15:04:05") + " UTC von " + host + ".",
	}.Message(rcpt, "Stallfunk: Testnachricht")
	if err != nil {
		return err
	}
	server := "the configured mailer"
	if cfg.SMTPHost != "" {
		server = cfg.SMTPHost + ":" + strconv.Itoa(cfg.SMTPPort)
	}
	if err := mailer.Send(ctx, msg); err != nil {
		return fmt.Errorf("sending the test mail to %s via %s failed: %w\n(check REITERHOF_SMTP_HOST/PORT/USER/PASSWORD/FROM in the env file; port 465 = implicit TLS, others = STARTTLS)", rcpt, server, err)
	}
	fmt.Fprintf(e.Out, "test mail to %s accepted by %s (from %s)\n", rcpt, server, orDash(cfg.SMTPFrom))
	return nil
}

func runWeatherRefresh(ctx context.Context, e *Env, args []string) error {
	if _, err := parse(e.flags("weather refresh"), args, 0, 0); err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	located, err := stables.ListLocated(ctx, pool)
	if err != nil {
		return err
	}
	if len(located) == 0 {
		fmt.Fprintln(e.Err, "warning: no stable has coordinates, so there is nothing to fetch (stable update ID --lat N --lng N)")
	}
	var fetcher weather.Fetcher = &weather.DWD{}
	if e.Fetcher != nil {
		fetcher = e.Fetcher
	}
	svc := &weather.Service{Pool: pool, Fetcher: fetcher, Log: e.log(), Now: e.now, StationID: e.Config.WeatherStation}
	refreshErr := svc.Refresh(ctx)
	if refreshErr == nil {
		fmt.Fprintf(e.Out, "weather refreshed for %d stable(s)\n", len(located))
	}

	// Same as the hourly job in cmd/api: a fresh forecast may change tonight's blanket
	// recommendation, which sends the weather change push.
	native, err := push.NewClientFromEnv(e.log())
	if err != nil {
		fmt.Fprintf(e.Err, "warning: native push: %v\n", err)
	}
	var notify httpx.Notifier = push.NewNotifier(pool, native, e.log())
	if e.Notify != nil {
		notify = e.Notify
	}
	blanketSvc := blankets.NewService(httpx.Deps{Pool: pool, Config: e.Config, Log: e.log(), Now: e.now, Notify: notify})
	checkErr := blanketSvc.CheckWeatherChange(ctx)
	if checkErr == nil {
		fmt.Fprintln(e.Out, "blanket weather-change check done")
	}
	return errors.Join(refreshErr, checkErr)
}

func runMigrateStatus(ctx context.Context, e *Env, args []string) error {
	if _, err := parse(e.flags("migrate status"), args, 0, 0); err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	states, err := db.Status(ctx, pool, e.Migrations)
	if err != nil {
		return err
	}
	if e.JSON {
		return e.writeJSON(states)
	}
	pending := 0
	table := make([][]string, 0, len(states))
	for _, s := range states {
		state := "pending"
		if s.Applied {
			state = "applied"
		} else {
			pending++
		}
		table = append(table, []string{fmt.Sprintf("%04d", s.Version), s.Name, state, fmtTimePtr(s.AppliedAt)})
	}
	writeTable(e.Out, []string{"VERSION", "FILE", "STATE", "APPLIED (UTC)"}, table)
	fmt.Fprintf(e.Out, "%d applied, %d pending\n", len(states)-pending, pending)
	return nil
}

func runMigrateUp(ctx context.Context, e *Env, args []string) error {
	if _, err := parse(e.flags("migrate up"), args, 0, 0); err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	before, err := db.Status(ctx, pool, e.Migrations)
	if err != nil {
		return err
	}
	if err := db.Migrate(ctx, pool, e.Migrations); err != nil {
		return err
	}
	n := 0
	for _, s := range before {
		if !s.Applied {
			fmt.Fprintf(e.Out, "applied %s\n", s.Name)
			n++
		}
	}
	if n == 0 {
		fmt.Fprintln(e.Out, "database is up to date")
	}
	return nil
}
