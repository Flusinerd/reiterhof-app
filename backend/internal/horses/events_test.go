package horses_test

import (
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/horses"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime/realtimetest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func TestHorseChangedEvents(t *testing.T) {
	e := setup(t)
	w := realtimetest.Watch(t, e.pool, seed.StableB)
	expect := func(what, horseID string) {
		t.Helper()
		data := w.Next(horses.EventChanged)
		if data["horse_id"] != horseID || len(data) != 1 {
			t.Fatalf("%s: data = %v, want only horse_id %s", what, data, horseID)
		}
	}

	rec := e.call(seed.UserJan, "POST", "/api/v1/horses", map[string]any{"name": "Stella"})
	e.expect(rec, 201)
	id := decode[horses.Horse](t, rec).ID
	expect("create", id)
	path := "/api/v1/horses/" + id
	e.expect(e.call(seed.UserJan, "PATCH", path, map[string]any{"name": "Stella Bella"}), 200)
	expect("patch", id)
	e.expect(e.call(seed.UserJan, "PUT", path+"/riders/"+seed.UserTom, map[string]any{}), 200)
	expect("rider set", id)
	e.expect(e.call(seed.UserJan, "DELETE", path+"/riders/"+seed.UserTom, nil), 204)
	expect("rider removed", id)

	rec = e.call(seed.UserJan, "POST", path+"/emergency-contacts", map[string]any{"label": "Tierarzt", "name": "Dr. Weber", "phone": "0171 1"})
	e.expect(rec, 201)
	expect("contact added", id)
	contact := decode[horses.Contact](t, rec).ID
	e.expect(e.call(seed.UserJan, "PATCH", path+"/emergency-contacts/"+contact, map[string]any{"phone": "0171 2"}), 200)
	expect("contact edited", id)
	e.expect(e.call(seed.UserJan, "DELETE", path+"/emergency-contacts/"+contact, nil), 204)
	expect("contact removed", id)

	// Nothing changed (empty patch), refused (no rights) or not found: nothing is published.
	e.expect(e.call(seed.UserJan, "PATCH", path, map[string]any{}), 200)
	e.expect(e.call(seed.UserTom, "PATCH", path, map[string]any{"name": "x"}), 403)
	e.expect(e.call(seed.UserJan, "DELETE", path+"/riders/"+seed.UserTom, nil), 404)
	e.expect(e.call(seed.UserJan, "DELETE", path+"/emergency-contacts/"+contact, nil), 404)
	w.Quiet()
}
