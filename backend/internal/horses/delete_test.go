package horses_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files/filestest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/horses"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime/realtimetest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func TestNameMatches(t *testing.T) {
	for _, c := range []struct {
		name, typed string
		want        bool
	}{
		{"Luna", "Luna", true},
		{"Luna", "  luna ", true},
		{"Käthe", "KÄTHE", true},
		{"Luna", "Lun", false},
		{"Luna", "", false},
		{"Luna", "   ", false},
		{" Luna ", "luna", true},
	} {
		if got := horses.NameMatches(c.name, c.typed); got != c.want {
			t.Errorf("NameMatches(%q, %q) = %v, want %v", c.name, c.typed, got, c.want)
		}
	}
}

// seedHorse creates a horse of Jan in the seed stable with a row in every table that points to a
// horse, a reminder of every kind that points to one of its records without a foreign key, and
// four stored files (photo, document, observation image, blanket image). It returns the horse id
// and the file paths.
func (e *env) seedHorse(t *testing.T) (string, map[string]string) {
	t.Helper()
	ctx := context.Background()
	save := func(data []byte, ct string) string {
		s, err := files.Save(ctx, seed.StableB, bytes.NewReader(data), ct)
		if err != nil {
			t.Fatal(err)
		}
		return s.Path
	}
	paths := map[string]string{
		"photo":       save(filestest.PNG(), "image/png"),
		"document":    save([]byte("%PDF-1.7\nhello\n"), "application/pdf"),
		"observation": save(filestest.JPEG(), "image/jpeg"),
		"blanket":     save(filestest.PNG(), "image/png"),
	}
	var horse string
	if err := e.pool.QueryRow(ctx, `INSERT INTO horses (stable_id, name, owner_id, photo_path)
		VALUES ($1, 'Tessa', $2, $3) RETURNING id::text`, seed.StableB, seed.UserJan, paths["photo"]).Scan(&horse); err != nil {
		t.Fatal(err)
	}
	// Test constants only, so the ids and paths are spliced in as literals.
	lit := strings.NewReplacer("$1", "'"+seed.StableB+"'", "$2", "'"+horse+"'", "$3", "'"+seed.UserJan+"'", "$4", "'"+seed.UserMia+"'",
		"$5", "'"+paths["document"]+"'", "$6", "'"+paths["blanket"]+"'", "$7", "'"+paths["observation"]+"'")
	stmts := []string{
		`INSERT INTO horse_riders (stable_id, horse_id, user_id) VALUES ($1, $2, $4)`,
		`INSERT INTO rider_focuses (stable_id, horse_id, user_id, keys) VALUES ($1, $2, $4, '{takt}')`,
		`INSERT INTO blankets (id, stable_id, horse_id, name, photo_path) VALUES ('00000000-0000-4000-8000-0000000007b1', $1, $2, 'Regendecke', $6)`,
		`INSERT INTO blanket_rules (stable_id, horse_id, position, blanket_id) VALUES ($1, $2, 1, '00000000-0000-4000-8000-0000000007b1')`,
		`INSERT INTO blanket_states (stable_id, horse_id, day, action, covered_with, changed_by) VALUES ($1, $2, current_date, 'covered', '00000000-0000-4000-8000-0000000007b1', $3)`,
		`INSERT INTO blanket_overrides (stable_id, horse_id, blanket_id, day_from, set_by) VALUES ($1, $2, '00000000-0000-4000-8000-0000000007b1', current_date, $3)`,
		`INSERT INTO requests (id, stable_id, type, horse_id, created_by, date, description) VALUES ('00000000-0000-4000-8000-0000000007b2', $1, 'other', $2, $3, current_date, 'Hilfe')`,
		`INSERT INTO observations (id, stable_id, horse_id, reported_by, description, media) VALUES ('00000000-0000-4000-8000-0000000007b3', $1, $2, $4, 'Schwellung', jsonb_build_array($7))`,
		`INSERT INTO health_items (id, stable_id, horse_id, kind, label, due_date) VALUES ('00000000-0000-4000-8000-0000000007b4', $1, $2, 'farrier', 'Hufschmied', current_date + 3)`,
		`INSERT INTO emergency_contacts (stable_id, horse_id, label, name, phone) VALUES ($1, $2, 'Tierarzt', 'Dr. Weber', '0171 1')`,
		`INSERT INTO horse_documents (stable_id, horse_id, kind, title, file_path, uploaded_by) VALUES ($1, $2, 'passport', 'Pass', $5, $3)`,
		`INSERT INTO training_profiles (stable_id, horse_id) VALUES ($1, $2)`,
		`INSERT INTO sessions (id, stable_id, horse_id, user_id, activity, started_at) VALUES ('00000000-0000-4000-8000-0000000007b5', $1, $2, $3, 'hall', now())`,
		`INSERT INTO gait_windows (session_id, stable_id, window_count, windows) VALUES ('00000000-0000-4000-8000-0000000007b5', $1, 1, '[]')`,
		`INSERT INTO week_slots (stable_id, horse_id, day, user_id) VALUES ($1, $2, current_date, $3)`,
		`INSERT INTO reha_plans (id, stable_id, horse_id, diagnosis, start_date) VALUES ('00000000-0000-4000-8000-0000000007b6', $1, $2, 'Sehnenreizung', current_date)`,
		`INSERT INTO reha_days (stable_id, reha_plan_id, day, done_by) VALUES ($1, '00000000-0000-4000-8000-0000000007b6', current_date, $3)`,
		`INSERT INTO dressage_tests (stable_id, horse_id, created_by, kind, class, arena, title, seed, plan) VALUES ($1, $2, $3, 'practice', 'A', '20x60', 'Probe', 1, '{}')`,
		// Reminders have no foreign key to their source: three that belong to the horse, three that do not.
		`INSERT INTO reminders (stable_id, user_id, kind, title, due_at, source_table, source_id) VALUES
			($1, $3, 'health_due', 'Fällig', now(), 'health_items', '00000000-0000-4000-8000-0000000007b4'),
			($1, $3, 'reha', 'Reha', now(), 'reha_plans', '00000000-0000-4000-8000-0000000007b6'),
			($1, $3, 'blanket_weather', 'Wetter', now(), 'blanket_weather', $2),
			($1, $3, 'helper', 'Hilfe', now(), 'requests', '00000000-0000-4000-8000-0000000007b2'),
			($1, $3, 'blanket_weather', 'Wetter', now(), 'blanket_weather', '00000000-0000-4000-8000-000000000302'),
			($1, $3, 'training', 'Plan', now(), 'training_plan', $1)`,
	}
	for _, q := range stmts {
		if _, err := e.pool.Exec(ctx, lit.Replace(q)); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	return horse, paths
}

func (e *env) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDeleteHorse(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	horse, paths := e.seedHorse(t)

	// Every table whose foreign key points to horses holds a row of this horse (the test fails for a
	// table added later until it is seeded above), and the files are loadable.
	rows, err := e.pool.Query(ctx, `SELECT DISTINCT c.conrelid::regclass::text, a.attname
		FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
		WHERE c.contype = 'f' AND c.confrelid = 'horses'::regclass ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	type ref struct{ table, column string }
	var refs []ref
	for rows.Next() {
		var r ref
		if err := rows.Scan(&r.table, &r.column); err != nil {
			t.Fatal(err)
		}
		refs = append(refs, r)
	}
	rows.Close()
	if len(refs) < 15 {
		t.Fatalf("found only %d foreign keys to horses: %v", len(refs), refs)
	}
	for _, r := range refs {
		if n := e.count(t, `SELECT count(*) FROM `+r.table+` WHERE `+r.column+` = $1`, horse); n == 0 {
			t.Errorf("%s.%s has no row for the test horse", r.table, r.column)
		}
	}
	// Foreign keys of rows below the horse's tables.
	for _, q := range []string{
		`SELECT count(*) FROM gait_windows WHERE session_id = '00000000-0000-4000-8000-0000000007b5'`,
		`SELECT count(*) FROM reha_days WHERE reha_plan_id = '00000000-0000-4000-8000-0000000007b6'`,
	} {
		if e.count(t, q) != 1 {
			t.Fatalf("not seeded: %s", q)
		}
	}
	// A file another horse's record still points to must survive.
	if _, err := e.pool.Exec(ctx, `UPDATE blankets SET photo_path = $2 WHERE id = $1`, seed.BlanketLuna100, paths["observation"]); err != nil {
		t.Fatal(err)
	}
	w := realtimetest.Watch(t, e.pool, seed.StableB)
	path := "/api/v1/horses/" + horse

	// Access: members without rights, riders and strangers are refused; nothing is deleted.
	body := map[string]any{"confirm_name": "Tessa"}
	e.expect(e.call(seed.UserTom, "DELETE", path, body), 403)
	e.expect(e.call(seed.UserMia, "DELETE", path, body), 403)
	e.expect(e.call(otherUser, "DELETE", path, body), 404)
	e.expect(e.call("", "DELETE", path, body), 401)
	e.expect(e.call(seed.UserJan, "DELETE", "/api/v1/horses/"+otherHorse, body), 404)
	e.expect(e.call(seed.UserJan, "DELETE", "/api/v1/horses/not-a-uuid", body), 404)

	// The name must match; a wrong or missing one is a 400 with a German message and deletes nothing.
	for name, b := range map[string]any{
		"wrong":   map[string]any{"confirm_name": "Luna"},
		"partial": map[string]any{"confirm_name": "Tess"},
		"empty":   map[string]any{"confirm_name": "  "},
		"missing": map[string]any{},
		"none":    nil,
	} {
		rec := e.call(seed.UserJan, "DELETE", path, b)
		if rec.Code != 400 {
			t.Fatalf("%s: status = %d, want 400: %s", name, rec.Code, rec.Body)
		}
		if got := rec.Body.String(); !bytes.Contains([]byte(got), []byte("validation_failed")) || !bytes.Contains([]byte(got), []byte("Name")) {
			t.Errorf("%s: body = %s", name, got)
		}
	}
	e.expect(e.call(seed.UserJan, "DELETE", path, map[string]any{"confirm_name": "Tessa", "x": 1}), 400)
	if e.count(t, `SELECT count(*) FROM horses WHERE id = $1`, horse) != 1 {
		t.Fatal("a refused delete removed the horse")
	}
	w.Quiet()

	// The owner deletes it; case and spaces of the name do not matter.
	e.expect(e.call(seed.UserJan, "DELETE", path, map[string]any{"confirm_name": "  tESSA "}), 204)
	data := w.Next(horses.EventChanged)
	if data["horse_id"] != horse || len(data) != 1 {
		t.Errorf("event data = %v, want only horse_id", data)
	}

	// Nothing references the horse any more, in tables with and without a foreign key.
	for _, r := range refs {
		if r.table == "requests" {
			continue
		}
		if n := e.count(t, `SELECT count(*) FROM `+r.table+` WHERE `+r.column+` = $1`, horse); n != 0 {
			t.Errorf("%s.%s still has %d rows", r.table, r.column, n)
		}
	}
	for what, q := range map[string]string{
		"gait_windows": `SELECT count(*) FROM gait_windows WHERE session_id = '00000000-0000-4000-8000-0000000007b5'`,
		"reha_days":    `SELECT count(*) FROM reha_days WHERE reha_plan_id = '00000000-0000-4000-8000-0000000007b6'`,
		"reminders of its health item, reha plan and weather change": `SELECT count(*) FROM reminders WHERE
			(source_table = 'health_items' AND source_id = '00000000-0000-4000-8000-0000000007b4')
			OR (source_table = 'reha_plans' AND source_id = '00000000-0000-4000-8000-0000000007b6')
			OR (source_table = 'blanket_weather' AND source_id = '` + horse + `')`,
	} {
		if n := e.count(t, q); n != 0 {
			t.Errorf("%s: %d rows left", what, n)
		}
	}
	// The help request stays without a horse, with its reminder; other horses' reminders stay.
	if n := e.count(t, `SELECT count(*) FROM requests WHERE id = '00000000-0000-4000-8000-0000000007b2' AND horse_id IS NULL`); n != 1 {
		t.Errorf("request kept = %d, want 1", n)
	}
	if n := e.count(t, `SELECT count(*) FROM reminders WHERE source_table = 'requests' AND source_id = '00000000-0000-4000-8000-0000000007b2'`); n != 1 {
		t.Errorf("request reminder kept = %d, want 1", n)
	}
	if n := e.count(t, `SELECT count(*) FROM reminders WHERE source_table = 'blanket_weather' AND source_id = $1`, seed.HorseFanta); n != 1 {
		t.Errorf("reminder of another horse = %d, want 1", n)
	}
	if n := e.count(t, `SELECT count(*) FROM reminders WHERE source_table = 'training_plan'`); n != 1 {
		t.Errorf("stable reminder = %d, want 1", n)
	}
	if e.count(t, `SELECT count(*) FROM horses WHERE stable_id = $1`, seed.StableB) != 7 {
		t.Error("another horse was deleted")
	}

	// Files: the unreferenced ones are gone, the one another record uses stays.
	for _, k := range []string{"photo", "document", "blanket"} {
		if _, _, err := files.Open(seed.StableB, paths[k]); err != files.ErrNotFound {
			t.Errorf("%s file: %v, want ErrNotFound", k, err)
		}
	}
	if f, _, err := files.Open(seed.StableB, paths["observation"]); err != nil {
		t.Errorf("shared file was removed: %v", err)
	} else {
		f.Close()
	}

	// The horse is gone for everybody, a second delete is a 404.
	e.expect(e.call(seed.UserJan, "GET", path, nil), 404)
	e.expect(e.call(seed.UserJan, "DELETE", path, body), 404)
	w.Quiet()
}

// An admin who does not own the horse may delete it.
func TestDeleteHorseAsAdmin(t *testing.T) {
	e := setup(t)
	horse, _ := e.seedHorse(t)
	if _, err := e.pool.Exec(context.Background(), `UPDATE horses SET owner_id = $1 WHERE id = $2`, seed.UserTom, horse); err != nil {
		t.Fatal(err)
	}
	e.expect(e.call(seed.UserAnna, "DELETE", "/api/v1/horses/"+horse, map[string]any{"confirm_name": "Tessa"}), 403)
	e.expect(e.call(seed.UserJan, "DELETE", "/api/v1/horses/"+horse, map[string]any{"confirm_name": "Tessa"}), 204)
	if e.count(t, `SELECT count(*) FROM horses WHERE id = $1`, horse) != 0 {
		t.Error("horse still there")
	}
}

// A horse without stored files and data is deleted too.
func TestDeleteHorseWithoutFiles(t *testing.T) {
	e := setup(t)
	var horse string
	if err := e.pool.QueryRow(context.Background(), `INSERT INTO horses (stable_id, name, owner_id) VALUES ($1, 'Mini', $2) RETURNING id::text`,
		seed.StableB, seed.UserJan).Scan(&horse); err != nil {
		t.Fatal(err)
	}
	e.expect(e.call(seed.UserJan, "DELETE", "/api/v1/horses/"+horse, map[string]any{"confirm_name": "Mini"}), 204)
}
