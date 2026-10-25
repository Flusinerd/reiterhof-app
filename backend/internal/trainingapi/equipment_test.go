package trainingapi_test

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const equipmentPath = "/api/v1/stables/training-equipment"

// location reads a location of the equipment answer.
func location(got map[string]any, name string) (equipment []string, shared bool) {
	l := obj(obj(got["locations"])[name])
	return strs(l["equipment"]), l["shared"] == true
}

func TestStableEquipmentDefaults(t *testing.T) {
	e := newEnv(t)
	e.exec(`UPDATE stables SET training_equipment = '{}'`) // the demo stable has saved its equipment: start without
	e.errCode("", http.MethodGet, equipmentPath, "", http.StatusUnauthorized)

	// Every member reads it; the defaults fill every location, set is false until it is saved.
	for user, canEdit := range map[string]bool{seed.UserSarah: false, seed.UserMia: false, seed.UserAnna: false, seed.UserJan: true} {
		got := e.call(user, http.MethodGet, equipmentPath, "", http.StatusOK)
		if got["set"] != false || got["can_edit"] != canEdit {
			t.Fatalf("%s: set %v, can_edit %v", short(user), got["set"], got["can_edit"])
		}
		for name, want := range map[string][]string{
			"hall": {"poles"}, "arena": {"poles"}, "lunge": {}, "jumping": {"jumps", "poles"}, "groundwork": {"poles"},
		} {
			if eq, shared := location(got, name); !reflect.DeepEqual(eq, want) || shared {
				t.Errorf("%s: %s = %v shared %v, want %v not shared", short(user), name, eq, shared, want)
			}
		}
		if len(obj(got["locations"])) != 5 {
			t.Errorf("locations = %v", got["locations"])
		}
		keys := obj(got["keys"])
		if !reflect.DeepEqual(strs(keys["facility"]), []string{"poles", "cavaletti", "jumps", "obstacles"}) || len(list(keys["personal"])) != 11 {
			t.Errorf("keys = %v", keys)
		}
	}
	// An empty location is an empty list, not null.
	raw := e.do(seed.UserJan, http.MethodGet, equipmentPath, "")
	if want := `"lunge":{"equipment":[],"shared":false}`; !strings.Contains(raw.Body.String(), want) {
		t.Errorf("answer %s lacks %s", raw.Body, want)
	}
}

func TestStableEquipmentSave(t *testing.T) {
	e := newEnv(t)
	e.exec(`UPDATE stables SET training_equipment = '{}'`)

	// Only admins save it.
	for _, user := range []string{seed.UserSarah, seed.UserMia, seed.UserAnna} {
		e.errCode(user, http.MethodPut, equipmentPath, `{"locations":{}}`, http.StatusForbidden)
	}
	e.errCode("", http.MethodPut, equipmentPath, `{"locations":{}}`, http.StatusUnauthorized)

	// Duplicates collapse, a location without equipment takes its default, one that is not sent too.
	got := e.call(seed.UserJan, http.MethodPut, equipmentPath, `{"locations": {
		"hall": {"equipment": ["poles", "cavaletti", "poles"], "shared": true},
		"lunge": {"equipment": ["poles", "lunge", "side_reins"]},
		"arena": {"shared": false},
		"groundwork": {"equipment": []}}}`, http.StatusOK)
	if got["set"] != true || got["can_edit"] != true {
		t.Fatalf("saved = %v", got)
	}
	for name, want := range map[string]struct {
		equipment []string
		shared    bool
	}{
		"hall": {[]string{"poles", "cavaletti"}, true}, "lunge": {[]string{"poles", "lunge", "side_reins"}, false},
		"arena": {[]string{"poles"}, false}, "groundwork": {[]string{}, false}, "jumping": {[]string{"jumps", "poles"}, false},
	} {
		if eq, shared := location(got, name); !reflect.DeepEqual(eq, want.equipment) || shared != want.shared {
			t.Errorf("%s = %v shared %v, want %v shared %v", name, eq, shared, want.equipment, want.shared)
		}
	}
	// A member sees what the admin saved; the stable's column holds only what was sent.
	again := e.call(seed.UserSarah, http.MethodGet, equipmentPath, "", http.StatusOK)
	if !reflect.DeepEqual(again["locations"], got["locations"]) || again["set"] != true {
		t.Fatalf("member view = %v", again)
	}
	var stored int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM jsonb_object_keys((SELECT training_equipment FROM stables WHERE id = $1))`, seed.StableB).Scan(&stored); err != nil || stored != 4 {
		t.Fatalf("stored locations = %d (%v), want the 4 that were sent", stored, err)
	}

	// The next request replaces it: what is not sent is the default again.
	got = e.call(seed.UserJan, http.MethodPut, equipmentPath, `{"locations": {"jumping": {"equipment": ["jumps"], "shared": true}}}`, http.StatusOK)
	if eq, shared := location(got, "hall"); !reflect.DeepEqual(eq, []string{"poles"}) || shared {
		t.Fatalf("hall after replacing = %v %v", eq, shared)
	}
	if eq, shared := location(got, "jumping"); !reflect.DeepEqual(eq, []string{"jumps"}) || !shared {
		t.Fatalf("jumping = %v %v", eq, shared)
	}
	// An empty object resets everything.
	got = e.call(seed.UserJan, http.MethodPut, equipmentPath, `{"locations": {}}`, http.StatusOK)
	if got["set"] != false {
		t.Fatalf("reset = %v", got)
	}

	// Unknown locations (hack and walker have none) and keys, and malformed bodies are refused.
	for name, body := range map[string]string{
		"unknown location": `{"locations": {"pool": {"equipment": []}}}`,
		"hack":             `{"locations": {"hack": {"equipment": []}}}`,
		"walker":           `{"locations": {"walker": {}}}`,
		"unknown key":      `{"locations": {"hall": {"equipment": ["trampoline"]}}}`,
		"empty key":        `{"locations": {"hall": {"equipment": [""]}}}`,
		"no locations":     `{}`,
		"null locations":   `{"locations": null}`,
		"not an object":    `{"locations": []}`,
		"equipment":        `{"locations": {"hall": {"equipment": "poles"}}}`,
		"shared":           `{"locations": {"hall": {"shared": "yes"}}}`,
		"unknown field":    `{"locations": {"hall": {"size": 3}}}`,
		"unknown top":      `{"locations": {}, "name": "x"}`,
	} {
		if code := e.errCode(seed.UserJan, http.MethodPut, equipmentPath, body, http.StatusBadRequest); code != "validation_failed" && code != "invalid_json" {
			t.Errorf("%s: code = %q", name, code)
		}
	}
	if got := e.call(seed.UserJan, http.MethodGet, equipmentPath, "", http.StatusOK); got["set"] != false {
		t.Fatalf("a refused request saved something: %v", got)
	}
}

// The equipment belongs to the stable: an admin of another stable changes only their own.
func TestStableEquipmentIsPerStable(t *testing.T) {
	e := newEnv(t)
	const otherStable, otherAdmin = "00000000-0000-4000-8000-0000000001f7", "00000000-0000-4000-8000-0000000002f7"
	e.exec(`INSERT INTO stables (id, name) VALUES ($1, 'Fremder Stall')`, otherStable)
	e.exec(`INSERT INTO users (id, stable_id, name, email, is_admin) VALUES ($1, $2, 'Fremd', 'fremd7@example.org', true)`, otherAdmin, otherStable)

	e.call(seed.UserJan, http.MethodPut, equipmentPath, `{"locations": {"hall": {"equipment": ["cavaletti"], "shared": true}}}`, http.StatusOK)
	got := e.call(otherAdmin, http.MethodGet, equipmentPath, "", http.StatusOK)
	if got["set"] != false {
		t.Fatalf("other stable sees %v", got)
	}
	e.call(otherAdmin, http.MethodPut, equipmentPath, `{"locations": {"arena": {"equipment": ["jumps"]}}}`, http.StatusOK)
	if eq, shared := location(e.call(seed.UserJan, http.MethodGet, equipmentPath, "", http.StatusOK), "hall"); !reflect.DeepEqual(eq, []string{"cavaletti"}) || !shared {
		t.Fatalf("Jan's hall = %v %v", eq, shared)
	}
	if eq, _ := location(e.call(seed.UserJan, http.MethodGet, equipmentPath, "", http.StatusOK), "arena"); !reflect.DeepEqual(eq, []string{"poles"}) {
		t.Fatalf("Jan's arena = %v", eq)
	}
}
