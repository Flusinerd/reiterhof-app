package health_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/health"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func (e *env) bundleOf(user, horse, itemID string) *health.Bundle {
	e.t.Helper()
	rec := e.call(user, "GET", "/api/v1/horses/"+horse+"/health", nil)
	e.expect(rec, 200)
	resp := decode[health.Response](e.t, rec)
	for _, it := range resp.Items {
		if it.ID == itemID {
			// The summary tile carries the same annotation as the list.
			if s := resp.Summary[it.Kind]; s != nil && s.ID == it.ID && (s.Bundle == nil) != (it.Bundle == nil) {
				e.t.Errorf("summary and items disagree on the bundle of %s", it.Label)
			}
			return it.Bundle
		}
	}
	e.t.Fatalf("item %s not in response", itemID)
	return nil
}

func TestBundleCountsOtherHorsesInAisle(t *testing.T) {
	e := setup(t)
	// "Today" is 2026-10-01. Luna's vet appointment is in 4 days.
	lunaVet := e.addItem(seed.HorseLuna, "vet", "Tierarzt", "2026-10-05", 0, "")
	e.addItem(seed.HorseFanta, "vet", "Tierarzt", "2026-10-15", 0, "")  // +10 days: counts
	e.addItem(seed.HorseCookie, "vet", "Tierarzt", "2026-09-25", 0, "") // -10 days (overdue): counts
	e.addItem(seed.HorseNala, "vet", "Tierarzt", "2026-10-06", 0, "")   // counts once ...
	e.addItem(seed.HorseNala, "vet", "Nachkontrolle", "2026-10-07", 0, "")
	e.addItem(seed.HorseBalu, "vet", "Tierarzt", "2026-10-25", 0, "")            // +20 days: too far
	e.addItem(seed.HorseMerlin, "vaccination", "Influenza", "2026-10-05", 0, "") // other kind
	e.addItem(seed.HorsePepe, "vet", "Kur", "2026-10-05", 0, "07:30")            // daily medication course, not an appointment
	// A horse of another stable never counts.
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO horses (id, stable_id, name) VALUES ('00000000-0000-4000-8000-0000000009a3', $1, 'Fremdpferd')`, otherStable); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO health_items (stable_id, horse_id, kind, label, due_date) VALUES ($1, '00000000-0000-4000-8000-0000000009a3', 'vet', 'Tierarzt', '2026-10-05')`, otherStable); err != nil {
		t.Fatal(err)
	}

	for _, u := range []string{seed.UserJan, seed.UserKai} { // every member sees it
		b := e.bundleOf(u, seed.HorseLuna, lunaVet)
		if b == nil {
			t.Fatalf("%s: no bundle", u)
		}
		if b.Kind != "vet" || b.Count != 3 || len(b.Horses) != 3 {
			t.Fatalf("bundle = %+v", b)
		}
		want := []struct{ id, name string }{{seed.HorseCookie, "Cookie"}, {seed.HorseFanta, "Fanta"}, {seed.HorseNala, "Nala"}}
		for i, w := range want {
			if b.Horses[i].ID != w.id || b.Horses[i].Name != w.name {
				t.Errorf("horses[%d] = %+v, want %s", i, b.Horses[i], w.name)
			}
		}
	}

	// The other direction: Fanta's vet item is 10 days from Luna's and 20 from Balu's.
	fantaVet := e.addItem(seed.HorseFanta, "vet", "Kontrolle", "2026-10-08", 0, "")
	if b := e.bundleOf(seed.UserJan, seed.HorseFanta, fantaVet); b == nil || b.Count < 2 {
		t.Errorf("fanta bundle = %+v", b)
	}
}

func TestBundleOnlyForApproachingVetKinds(t *testing.T) {
	e := setup(t)
	// Everybody has the same appointments in 20 days, plus a partner for every kind at +5 days.
	for _, kind := range []string{"vet", "vaccination", "dentist", "farrier", "deworming", "physio"} {
		e.addItem(seed.HorseFanta, kind, "Andere", "2026-10-06", 0, "")
	}
	near := map[string]string{}
	far := map[string]string{}
	overdue := map[string]string{}
	for _, kind := range []string{"vet", "vaccination", "dentist", "farrier", "deworming", "physio"} {
		near[kind] = e.addItem(seed.HorseLuna, kind, "Bald", "2026-10-10", 0, "")
		far[kind] = e.addItem(seed.HorseLuna, kind, "Spaeter", "2026-10-16", 0, "") // 15 days ahead
		overdue[kind] = e.addItem(seed.HorseLuna, kind, "Ueberfaellig", "2026-09-29", 0, "")
	}
	for _, kind := range health.BundleKinds {
		if b := e.bundleOf(seed.UserJan, seed.HorseLuna, near[kind]); b == nil || b.Count != 1 || b.Kind != kind {
			t.Errorf("%s within 14 days: bundle = %+v", kind, b)
		}
		if b := e.bundleOf(seed.UserJan, seed.HorseLuna, far[kind]); b != nil {
			t.Errorf("%s in 15 days must not bundle yet: %+v", kind, b)
		}
		if b := e.bundleOf(seed.UserJan, seed.HorseLuna, overdue[kind]); b != nil {
			t.Errorf("%s overdue must not bundle: %+v", kind, b)
		}
	}
	for _, kind := range []string{"farrier", "deworming", "physio"} {
		if b := e.bundleOf(seed.UserJan, seed.HorseLuna, near[kind]); b != nil {
			t.Errorf("%s must never bundle: %+v", kind, b)
		}
	}
}

func TestBundleAbsentWithoutOtherHorses(t *testing.T) {
	e := setup(t)
	id := e.addItem(seed.HorseLuna, "vet", "Tierarzt", "2026-10-05", 0, "")
	rec := e.call(seed.UserJan, "GET", "/api/v1/horses/"+seed.HorseLuna+"/health", nil)
	e.expect(rec, 200)
	if b := e.bundleOf(seed.UserJan, seed.HorseLuna, id); b != nil {
		t.Errorf("bundle = %+v", b)
	}
	if strings.Contains(rec.Body.String(), `"bundle"`) {
		t.Errorf("no bundle key expected: %s", rec.Body)
	}
}
