package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Run inserts the example data in one transaction. It is idempotent: rows that
// already exist (by fixed ID) are left untouched, so local edits survive.
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	if err := insert(ctx, tx); err != nil {
		return err
	}
	if err := insertHorseCare(ctx, tx); err != nil {
		return err
	}
	if err := insertBlankets(ctx, tx); err != nil {
		return err
	}
	if err := insertTraining(ctx, tx, time.Now()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func insert(ctx context.Context, tx pgx.Tx) error {
	var b pgx.Batch
	q := func(sql string, args ...any) { b.Queue(sql+" ON CONFLICT DO NOTHING", args...) }

	q(`INSERT INTO stables (id, name, farm_name, city, lat, lng) VALUES ($1, 'Stallgasse B', 'Hof Ahlers', 'Dorsten', 51.66, 6.96)`, StableB)

	users := []struct{ id, name, color string }{
		{UserJan, "Jan", "blue"}, {UserAnna, "Anna", "rose"}, {UserJonas, "Jonas", "green"},
		{UserTom, "Tom", "orange"}, {UserSarah, "Sarah", "purple"}, {UserKai, "Kai", "teal"},
		{UserMia, "Mia", "yellow"}, {UserLea, "Lea", "red"},
	}
	for _, u := range users {
		q(`INSERT INTO users (id, stable_id, name, email, avatar_color, is_admin) VALUES ($1, $2, $3, lower($3) || '@example.org', $4, $5)`,
			u.id, StableB, u.name, u.color, u.id == UserJan)
	}

	horses := []struct {
		id, name, box, owner, sex, color string
		birthYear                        int
	}{
		{HorseLuna, "Luna", "12", UserJan, "mare", "brown", 2015},
		{HorseFanta, "Fanta", "14", UserAnna, "mare", "chestnut", 2012},
		{HorseBalu, "Balu", "16", UserJonas, "gelding", "black", 2010},
		{HorseCookie, "Cookie", "15", UserTom, "mare", "palomino", 2017},
		{HorseMerlin, "Merlin", "11", UserSarah, "gelding", "grey", 2014},
		{HorsePepe, "Pepe", "10", UserKai, "gelding", "bay", 2011},
		{HorseNala, "Nala", "13", UserAnna, "mare", "bay", 2018},
	}
	for _, h := range horses {
		q(`INSERT INTO horses (id, stable_id, name, box, owner_id, sex, color_key, birth_year) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			h.id, StableB, h.name, h.box, h.owner, h.sex, h.color, h.birthYear)
	}

	q(`INSERT INTO horse_riders (id, stable_id, horse_id, user_id, rules) VALUES ('00000000-0000-4000-8000-000000000601', $1, $2, $3, '["ride","groom"]')`, StableB, HorseLuna, UserMia)
	q(`INSERT INTO horse_riders (id, stable_id, horse_id, user_id, rules) VALUES ('00000000-0000-4000-8000-000000000602', $1, $2, $3, '["ride","groom"]')`, StableB, HorseFanta, UserLea)

	blankets := []struct {
		id, horse, name string
		fillG           int
	}{
		{BlanketLunaRain, HorseLuna, "Regendecke", 0},
		{BlanketLuna100, HorseLuna, "Decke 100 g", 100},
		{BlanketLuna200, HorseLuna, "Decke 200 g", 200},
		{BlanketBalu150, HorseBalu, "Decke 150 g", 150},
	}
	for _, bl := range blankets {
		q(`INSERT INTO blankets (id, stable_id, horse_id, name, fill_g) VALUES ($1, $2, $3, $4, $5)`, bl.id, StableB, bl.horse, bl.name, bl.fillG)
	}

	// Rules match half-open: temp_min <= temp < temp_max. NULL blanket = none.
	rules := []struct {
		id, horse string
		pos       int
		min, max  *float64
		rain      *bool
		blanket   *string
		note      string
	}{
		{"00000000-0000-4000-8000-000000000501", HorseLuna, 1, f(12), nil, nil, nil, "Keine Decke"},
		{"00000000-0000-4000-8000-000000000502", HorseLuna, 2, f(5), f(12), t(true), s(BlanketLunaRain), "Nur bei Regen: Regendecke"},
		{"00000000-0000-4000-8000-000000000503", HorseLuna, 3, f(5), f(12), nil, nil, "Keine Decke"},
		{"00000000-0000-4000-8000-000000000504", HorseLuna, 4, f(0), f(5), nil, s(BlanketLuna100), "Bei Regen zusätzlich Regendecke"},
		{"00000000-0000-4000-8000-000000000505", HorseLuna, 5, nil, f(0), nil, s(BlanketLuna200), ""},
		{"00000000-0000-4000-8000-000000000506", HorseBalu, 1, nil, f(-5), nil, s(BlanketBalu150), ""},
		{"00000000-0000-4000-8000-000000000507", HorseBalu, 2, nil, nil, nil, nil, "Keine Decke"},
	}
	for _, r := range rules {
		q(`INSERT INTO blanket_rules (id, stable_id, horse_id, position, temp_min, temp_max, rain, blanket_id, note) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''))`,
			r.id, StableB, r.horse, r.pos, r.min, r.max, r.rain, r.blanket, r.note)
	}

	profiles := []struct{ id, horse, status string }{
		{"00000000-0000-4000-8000-000000000701", HorseLuna, "fit"},
		{"00000000-0000-4000-8000-000000000702", HorseFanta, "reha"},
		{"00000000-0000-4000-8000-000000000703", HorseBalu, "fit"},
	}
	for _, p := range profiles {
		q(`INSERT INTO training_profiles (id, stable_id, horse_id, status) VALUES ($1, $2, $3, $4)`, p.id, StableB, p.horse, p.status)
	}

	res := tx.SendBatch(ctx, &b)
	for range b.Len() {
		if _, err := res.Exec(); err != nil {
			_ = res.Close()
			return fmt.Errorf("seed: %w", err)
		}
	}
	return res.Close()
}

func f(v float64) *float64 { return &v }
func t(v bool) *bool       { return &v }
func s(v string) *string   { return &v }
