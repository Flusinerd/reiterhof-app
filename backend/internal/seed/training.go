package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/load"
)

// Training seed (M6): profiles for all demo horses, a reha plan for Fanta, about two weeks of
// sessions relative to the day of seeding and a few planned week slots. The exercise library is
// not demo data: migration 0250 inserts it for every installation (IDs 8YY).
//
// ID scheme continues the one in ids.go: 8YY exercises (migration 0250), 9YY sessions, a01.. week slots,
// b01 reha plans; training profiles are 701-707.

func seedID(table string, n int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-000000000%s%02d", table, n)
}

type profileSeed struct {
	id, horse, discipline, level, status string
	allowed                              []training.AllowedActivity
	rbRules                              map[string]any
}

func on(a training.Activity) training.AllowedActivity {
	return training.AllowedActivity{Activity: a, Mode: training.ModeOn}
}
func off(a training.Activity) training.AllowedActivity {
	return training.AllowedActivity{Activity: a, Mode: training.ModeOff}
}
func cond(a training.Activity, note string) training.AllowedActivity {
	return training.AllowedActivity{Activity: a, Mode: training.ModeConditional, Note: note}
}

func insertTraining(ctx context.Context, tx pgx.Tx, now time.Time) error {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		loc = time.UTC
	}
	today := training.Day(now.In(loc))
	at := func(dayOffset, hour int) time.Time { // local wall clock time on today+dayOffset
		d := today.AddDate(0, 0, dayOffset)
		return time.Date(d.Year(), d.Month(), d.Day(), hour, 0, 0, 0, loc)
	}
	date := func(dayOffset int) string { return today.AddDate(0, 0, dayOffset).Format("2006-01-02") }

	var b pgx.Batch
	q := func(sql string, args ...any) { b.Queue(sql+" ON CONFLICT DO NOTHING", args...) }

	// --- profiles (fill rows that are still empty, keep local edits) ------------------
	rhythm, _ := json.Marshal(training.DefaultRhythm())
	lunaShows, _ := json.Marshal([]map[string]string{
		{"date": date(-9), "name": "Turnier Dorsten", "classes": "Dressur A"},
		{"date": date(3), "name": "Turnier", "classes": "Dressur L, Dressurreiter L", "helper": "Anna hilft beim Fertigmachen"},
	})
	noShows := []byte("[]")
	profiles := []profileSeed{
		{seedID("7", 1), HorseLuna, "dressage", "L", "fit",
			[]training.AllowedActivity{on(training.ActivityHall), on(training.ActivityArena), on(training.ActivityHack),
				on(training.ActivityLunge), off(training.ActivityJumping), on(training.ActivityGroundwork), on(training.ActivityWalker)},
			map[string]any{UserMia: map[string]any{
				"allowed_activities": []string{"hall", "arena", "lunge", "groundwork", "walker"},
				"max_intensity":      "medium", "may_hack_alone": false, "may_ride_shows": false}}},
		{seedID("7", 2), HorseFanta, "leisure", "A", "reha",
			[]training.AllowedActivity{on(training.ActivityWalker), on(training.ActivityGroundwork),
				cond(training.ActivityLunge, "Nur im Schritt"), off(training.ActivityHall), off(training.ActivityArena),
				off(training.ActivityHack), off(training.ActivityJumping)},
			map[string]any{UserLea: map[string]any{
				"allowed_activities": []string{"walker", "groundwork"},
				"max_intensity":      "light", "may_hack_alone": false, "may_ride_shows": false}}},
		{seedID("7", 3), HorseBalu, "jumping", "L", "fit",
			[]training.AllowedActivity{on(training.ActivityHall), on(training.ActivityArena), on(training.ActivityHack),
				on(training.ActivityJumping), on(training.ActivityLunge), on(training.ActivityGroundwork), on(training.ActivityWalker)},
			map[string]any{}},
		{seedID("7", 4), HorseCookie, "western", "", "fit",
			[]training.AllowedActivity{on(training.ActivityArena), on(training.ActivityHack), on(training.ActivityGroundwork),
				on(training.ActivityWalker), off(training.ActivityHall), off(training.ActivityLunge), off(training.ActivityJumping)},
			map[string]any{}},
		{seedID("7", 5), HorseMerlin, "eventing", "L", "fit",
			[]training.AllowedActivity{on(training.ActivityHall), on(training.ActivityArena), on(training.ActivityHack),
				on(training.ActivityJumping), on(training.ActivityLunge), on(training.ActivityWalker), off(training.ActivityGroundwork)},
			map[string]any{}},
		{seedID("7", 6), HorsePepe, "leisure", "", "fit",
			[]training.AllowedActivity{on(training.ActivityHack), on(training.ActivityArena), on(training.ActivityWalker),
				cond(training.ActivityJumping, "Nur kleine Sprünge"), off(training.ActivityHall), off(training.ActivityLunge), off(training.ActivityGroundwork)},
			map[string]any{}},
		{seedID("7", 7), HorseNala, "young_horse", "", "fit",
			[]training.AllowedActivity{on(training.ActivityGroundwork), on(training.ActivityLunge), on(training.ActivityHack),
				on(training.ActivityWalker), off(training.ActivityJumping), off(training.ActivityHall), off(training.ActivityArena)},
			map[string]any{}},
	}
	for _, p := range profiles {
		allowed, _ := json.Marshal(p.allowed)
		rb, _ := json.Marshal(p.rbRules)
		shows := noShows
		if p.horse == HorseLuna {
			shows = lunaShows
		}
		b.Queue(`INSERT INTO training_profiles (id, stable_id, horse_id, discipline, level, allowed_activities, shows, season_end, rhythm, rb_rules, status)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, $10, $11)
			ON CONFLICT (horse_id) DO UPDATE SET
				discipline = EXCLUDED.discipline, level = EXCLUDED.level, allowed_activities = EXCLUDED.allowed_activities,
				shows = EXCLUDED.shows, season_end = EXCLUDED.season_end, rhythm = EXCLUDED.rhythm, rb_rules = EXCLUDED.rb_rules
			WHERE training_profiles.discipline IS NULL`,
			p.id, StableB, p.horse, p.discipline, p.level, allowed, shows, today.AddDate(0, 0, 60), rhythm, rb, p.status)
	}

	// --- reha plan for Fanta ----------------------------------------------------------
	phases, _ := json.Marshal([]map[string]any{
		{"name": "Schrittführen", "days": 14, "activity": "walker", "min_minutes": 15, "max_minutes": 20, "conditions": "Nur auf ebenem Boden"},
		{"name": "Longieren im Schritt", "days": 14, "activity": "lunge", "min_minutes": 10, "max_minutes": 15, "conditions": "Große Kreise, kein Trab"},
	})
	q(`INSERT INTO reha_plans (id, stable_id, horse_id, diagnosis, vet, start_date, phases, checkup_date, active, abort_criteria)
	   VALUES ($1, $2, $3, 'Sehnenzerrung vorne links', 'Dr. Berger', $4, $5, $6, true,
	           'Bei Lahmheit, Wärme oder Schwellung im Bein sofort abbrechen und Dr. Berger anrufen.')`,
		seedID("b", 1), StableB, HorseFanta, today.AddDate(0, 0, -10), phases, today.AddDate(0, 0, 20))

	// --- sessions of the last two weeks -----------------------------------------------
	type sessionSeed struct {
		horse, user, activity string
		offset, hour, minutes int
		canter                float64
		feel                  string
		focus                 int
		visible               bool
	}
	sessions := []sessionSeed{
		{HorseLuna, UserJan, "hall", -13, 17, 45, 0.2, "loose", 2, true},
		{HorseLuna, UserJan, "hack", -12, 16, 60, 0.15, "fresh", 0, true},
		{HorseLuna, UserMia, "lunge", -10, 17, 25, 0.1, "loose", 0, true},
		{HorseLuna, UserJan, "hall", -8, 17, 40, 0.25, "loose", 3, true},
		{HorseLuna, UserJan, "arena", -7, 16, 45, 0.3, "tense", 1, false},
		{HorseLuna, UserMia, "walker", -5, 18, 30, 0, "fresh", 0, true},
		{HorseLuna, UserJan, "hall", -4, 17, 50, 0.2, "loose", 2, true},
		{HorseLuna, UserJan, "groundwork", -2, 17, 30, 0, "loose", 0, true},
		{HorseLuna, UserMia, "hall", -1, 17, 40, 0.15, "tired", 0, true},
		{HorseBalu, UserJonas, "jumping", -11, 16, 40, 0.3, "fresh", 2, true},
		{HorseBalu, UserJonas, "hack", -9, 16, 60, 0.2, "loose", 0, true},
		{HorseBalu, UserJonas, "arena", -6, 16, 45, 0.3, "loose", 0, true},
		{HorseBalu, UserJonas, "jumping", -3, 16, 40, 0.3, "fresh", 3, true},
		{HorseFanta, UserAnna, "walker", -9, 15, 20, 0, "loose", 0, true},
		{HorseFanta, UserLea, "walker", -6, 15, 20, 0, "loose", 0, true},
		{HorseFanta, UserAnna, "walker", -3, 15, 20, 0, "fresh", 0, true},
		{HorseCookie, UserTom, "arena", -4, 18, 45, 0.3, "fresh", 0, true},
		{HorseMerlin, UserSarah, "hack", -5, 16, 60, 0.25, "fresh", 0, true},
		{HorsePepe, UserKai, "hack", -2, 17, 60, 0.2, "loose", 0, true},
		{HorseNala, UserAnna, "groundwork", -3, 15, 30, 0, "loose", 0, true},
	}
	for i, s := range sessions {
		gait, _ := json.Marshal(map[string]float64{"canter": s.canter})
		score := load.Score(s.minutes, training.Activity(s.activity), s.canter)
		var focus *int
		var exercise *string
		if s.focus > 0 {
			focus = &s.focus
			if s.horse == HorseLuna && s.activity != "groundwork" {
				id := seedID("8", 10) // Schulterherein
				exercise = &id
			}
		}
		q(`INSERT INTO sessions (id, stable_id, horse_id, user_id, activity, started_at, duration_min, gait_shares, feel,
		                         focus_rating, visible_to_rider, load_score, source, exercise_id)
		   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), $10, $11, $12, 'quick', $13)`,
			seedID("9", i+1), StableB, s.horse, s.user, s.activity, at(s.offset, s.hour), s.minutes, gait, s.feel,
			focus, s.visible, score, exercise)
	}

	// --- planned week slots for Luna --------------------------------------------------
	slots := []struct {
		n, offset int
		user      string
		activity  string
		status    string
	}{
		{1, 1, UserMia, "hall", "planned"},
		{2, 2, UserJan, "arena", "planned"},
		{3, 5, "", "", "rest"},
	}
	for _, s := range slots {
		var user, activity *string
		if s.user != "" {
			user = &s.user
		}
		if s.activity != "" {
			activity = &s.activity
		}
		q(`INSERT INTO week_slots (id, stable_id, horse_id, day, user_id, activity, status) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			seedID("a", s.n), StableB, HorseLuna, today.AddDate(0, 0, s.offset), user, activity, s.status)
	}

	res := tx.SendBatch(ctx, &b)
	for range b.Len() {
		if _, err := res.Exec(); err != nil {
			_ = res.Close()
			return fmt.Errorf("seed training: %w", err)
		}
	}
	return res.Close()
}
