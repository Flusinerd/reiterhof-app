package diary_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/diary"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const (
	otherStable = "00000000-0000-4000-8000-0000000009a1"
	otherUser   = "00000000-0000-4000-8000-0000000009a2"
)

func setup(t *testing.T) (*pgxpool.Pool, http.Handler) {
	t.Helper()
	pool := dbtest.NewSeeded(t)
	for _, q := range []string{
		`INSERT INTO stables (id, name) VALUES ('` + otherStable + `', 'Anderer Stall')`,
		`INSERT INTO users (id, stable_id, name, email) VALUES ('` + otherUser + `', '` + otherStable + `', 'Fremd', 'fremd@example.org')`,
	} {
		if _, err := pool.Exec(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
	return pool, httpapi.NewHandler(httpapi.Deps{Pool: pool})
}

func get(t *testing.T, pool *pgxpool.Pool, h http.Handler, user, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	if user != "" {
		authtest.Authorize(t, pool, req, user)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func page(t *testing.T, pool *pgxpool.Pool, h http.Handler, user, path string) diary.Page {
	t.Helper()
	rec := get(t, pool, h, user, path)
	if rec.Code != 200 {
		t.Fatalf("GET %s as %s = %d: %s", path, user, rec.Code, rec.Body)
	}
	var p diary.Page
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode %q: %v", rec.Body, err)
	}
	return p
}

func count(p diary.Page) (sessions, observations int) {
	for _, e := range p.Items {
		switch e.Type {
		case "session":
			sessions++
		case "observation":
			observations++
		}
	}
	return
}

func TestDiaryRoles(t *testing.T) {
	pool, h := setup(t)
	base := "/api/v1/horses/" + seed.HorseLuna + "/diary"

	// The seed has nine sessions of Luna (one hidden from riders) and two observations.
	// Owner and admin see everything, merged and newest first.
	p := page(t, pool, h, seed.UserJan, base+"?limit=100")
	if s, o := count(p); s != 9 || o != 2 {
		t.Fatalf("owner sees %d sessions, %d observations, want 9 and 2", s, o)
	}
	if p.NextBefore != nil {
		t.Errorf("next_before = %s on the last page", *p.NextBefore)
	}
	for i := 1; i < len(p.Items); i++ {
		if p.Items[i].At.After(p.Items[i-1].At) {
			t.Fatalf("not newest first at %d: %s after %s", i, p.Items[i].At, p.Items[i-1].At)
		}
	}
	for _, e := range p.Items {
		switch e.Type {
		case "session":
			if e.Session == nil || e.Observation != nil || e.Session.Label == "" || e.Session.Minutes == 0 || e.User.Name == "" {
				t.Errorf("session entry = %+v %+v", e, e.Session)
			}
		case "observation":
			if e.Observation == nil || e.Session != nil || e.Observation.Urgency == "" {
				t.Errorf("observation entry = %+v", e)
			}
		}
		if len(e.Day) != len("2006-01-02") {
			t.Errorf("day = %q", e.Day)
		}
	}
	// Sessions linked to a library exercise carry its title.
	var withExercise bool
	for _, e := range p.Items {
		if e.Session != nil && e.Session.ExerciseTitle != nil && *e.Session.ExerciseTitle != "" {
			withExercise = true
		}
	}
	if !withExercise {
		t.Error("no session carries an exercise title")
	}

	// A rider sees the sessions marked visible plus their own, and all observations.
	if s, o := count(page(t, pool, h, seed.UserMia, base+"?limit=100")); s != 8 || o != 2 {
		t.Errorf("rider sees %d sessions, %d observations, want 8 and 2", s, o)
	}
	// A plain member of the stable only sees observations.
	if s, o := count(page(t, pool, h, seed.UserTom, base+"?limit=100")); s != 0 || o != 2 {
		t.Errorf("member sees %d sessions, %d observations, want 0 and 2", s, o)
	}

	// Other stable, unknown horse, anonymous, bad parameters.
	for _, c := range []struct {
		user, path string
		want       int
	}{
		{otherUser, base, 404},
		{seed.UserJan, "/api/v1/horses/00000000-0000-4000-8000-0000000009ff/diary", 404},
		{seed.UserJan, "/api/v1/horses/not-a-uuid/diary", 404},
		{"", base, 401},
		{seed.UserJan, base + "?limit=0", 400},
		{seed.UserJan, base + "?limit=101", 400},
		{seed.UserJan, base + "?limit=x", 400},
		{seed.UserJan, base + "?before=yesterday", 400},
	} {
		if rec := get(t, pool, h, c.user, c.path); rec.Code != c.want {
			t.Errorf("GET %s as %q = %d, want %d", c.path, c.user, rec.Code, c.want)
		}
	}
}

func TestDiaryPagination(t *testing.T) {
	pool, h := setup(t)
	base := "/api/v1/horses/" + seed.HorseLuna + "/diary"

	all := page(t, pool, h, seed.UserJan, base+"?limit=100")
	var got []string
	path := base + "?limit=4"
	pages := 0
	for {
		p := page(t, pool, h, seed.UserJan, path)
		pages++
		if len(p.Items) > 4 {
			t.Fatalf("page has %d items", len(p.Items))
		}
		for _, e := range p.Items {
			got = append(got, e.Type+e.ID)
		}
		if p.NextBefore == nil {
			break
		}
		if pages > 10 {
			t.Fatal("pagination does not end")
		}
		path = base + "?limit=4&before=" + url.QueryEscape(*p.NextBefore)
	}
	if pages != 3 || len(got) != len(all.Items) {
		t.Fatalf("pages = %d, entries = %d, want 3 and %d", pages, len(got), len(all.Items))
	}
	for i, e := range all.Items {
		if got[i] != e.Type+e.ID {
			t.Errorf("entry %d differs between paged and full list", i)
		}
	}

	// A very old `before` gives an empty page, not null.
	empty := page(t, pool, h, seed.UserJan, base+"?before=2000-01-01T00:00:00Z")
	if empty.Items == nil || len(empty.Items) != 0 || empty.NextBefore != nil {
		t.Errorf("empty page = %+v", empty)
	}
}

func TestDiaryObservationMedia(t *testing.T) {
	pool, h := setup(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO observations (stable_id, horse_id, reported_by, category, description, media, created_at)
		VALUES ($1, $2, $3, 'injury', 'Kratzer am Bein', $4, $5)`,
		seed.StableB, seed.HorseLuna, seed.UserTom,
		[]string{seed.StableB + "/" + "0123456789abcdef0123456789abcdef.jpg", seed.StableB + "/" + "fedcba9876543210fedcba9876543210.png"},
		time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	p := page(t, pool, h, seed.UserAnna, "/api/v1/horses/"+seed.HorseLuna+"/diary")
	first := p.Items[0]
	if first.Type != "observation" || first.Observation.MediaCount != 2 || first.Observation.MediaURL == nil ||
		*first.Observation.MediaURL != files.URLFor(seed.StableB+"/0123456789abcdef0123456789abcdef.jpg") {
		t.Fatalf("first entry = %+v %+v", first, first.Observation)
	}
	if first.User.Name != "Tom" || first.Observation.Description == nil || *first.Observation.Description != "Kratzer am Bein" {
		t.Errorf("author or text wrong: %+v %+v", first.User, first.Observation)
	}
}
