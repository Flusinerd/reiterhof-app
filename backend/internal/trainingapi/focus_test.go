package trainingapi_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func strs(v any) []string {
	out := []string{}
	for _, x := range list(v) {
		out = append(out, str(x))
	}
	return out
}

func TestFocusCatalogRoute(t *testing.T) {
	e := newEnv(t)
	e.call("", http.MethodGet, "/api/v1/training/focuses", "", http.StatusUnauthorized)
	for _, user := range []string{seed.UserJan, seed.UserMia, seed.UserSarah} { // any stable member
		got := e.call(user, http.MethodGet, "/api/v1/training/focuses", "", http.StatusOK)
		focuses := list(got["focuses"])
		if len(focuses) != 15 {
			t.Fatalf("%d focuses, want 15", len(focuses))
		}
		first := obj(focuses[0])
		if first["key"] != "takt" || first["label"] != "Takt" || first["group"] != "scale" || num(first["phase"]) != 1 ||
			str(first["description"]) == "" || str(first["signs"]) == "" || !reflect.DeepEqual(strs(first["fits"]), []string{"hall", "arena", "lunge", "hack"}) {
			t.Fatalf("first focus = %v", first)
		}
		if len(first) != 7 {
			t.Errorf("a focus has %d fields, want key, label, group, phase, description, signs, fits: %v", len(first), first)
		}
	}
	byKey := map[string]map[string]any{}
	for _, f := range list(e.call(seed.UserJan, http.MethodGet, "/api/v1/training/focuses", "", http.StatusOK)["focuses"]) {
		byKey[str(obj(f)["key"])] = obj(f)
	}
	if v := byKey["versammlung"]; num(v["phase"]) != 3 || v["group"] != "scale" {
		t.Errorf("versammlung = %v", v)
	}
	if v := byKey["turnier"]; v["label"] != "Turniervorbereitung" || v["group"] != "practice" || num(v["phase"]) != 0 {
		t.Errorf("turnier = %v", v)
	}
}

func TestProfileFocuses(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/training-profile"
	with := func(extra string) string {
		return strings.Replace(validProfile, `"status": "fit"`, `"status": "fit", `+extra, 1)
	}
	keysOf := func(got map[string]any) []string {
		var keys []string
		for _, f := range list(got["focuses"]) {
			keys = append(keys, str(obj(f)["key"]))
		}
		return keys
	}

	// A profile without focuses (older clients): none, no block start, phase from level L and age.
	got := e.call(seed.UserJan, http.MethodPut, path, validProfile, http.StatusOK)
	if got["focuses"] == nil || len(list(got["focuses"])) != 0 || got["cycle_start"] != nil || got["phase"] != "schubkraft" {
		t.Fatalf("profile without focuses = %v", got)
	}
	if got["warnings"] == nil || len(list(got["warnings"])) != 0 {
		t.Fatalf("warnings = %v", got["warnings"])
	}
	if want := []string{"schwung", "geraderichten", "durchlaessigkeit"}; !reflect.DeepEqual(strs(got["focus_hints"]), want) {
		t.Fatalf("hints = %v, want the scale focuses of the phase %v", got["focus_hints"], want)
	}

	// The first focuses start the blocks: cycle_start is the Monday of the current week (Wed 25th).
	got = e.call(seed.UserJan, http.MethodPut, path, with(`"focuses": [{"key": "losgelassenheit"}, {"key": "geraderichten", "until": "2026-04-30"}]`), http.StatusOK)
	f := list(got["focuses"])
	if len(f) != 2 || obj(f[0])["key"] != "losgelassenheit" || obj(f[0])["until"] != nil || obj(f[1])["key"] != "geraderichten" || obj(f[1])["until"] != "2026-04-30" {
		t.Fatalf("focuses = %v", got["focuses"])
	}
	if got["cycle_start"] != "2026-03-23" {
		t.Fatalf("cycle_start = %v, want the Monday 2026-03-23", got["cycle_start"])
	}
	if want := []string{"schwung", "durchlaessigkeit"}; !reflect.DeepEqual(strs(got["focus_hints"]), want) {
		t.Fatalf("hints = %v, want %v", got["focus_hints"], want)
	}
	if len(list(got["warnings"])) != 0 {
		t.Fatalf("warnings = %v", got["warnings"])
	}

	// A request without focuses keeps them and the block start; the rider sees them read-only.
	got = e.call(seed.UserJan, http.MethodPut, path, validProfile, http.StatusOK)
	if !reflect.DeepEqual(keysOf(got), []string{"losgelassenheit", "geraderichten"}) || got["cycle_start"] != "2026-03-23" {
		t.Fatalf("omitted focuses must be kept: %v", got)
	}
	if mia := e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK); !reflect.DeepEqual(keysOf(mia), []string{"losgelassenheit", "geraderichten"}) || mia["phase"] != "schubkraft" {
		t.Fatalf("rider view = %v", mia)
	}
	e.errCode(seed.UserMia, http.MethodPut, path, with(`"focuses": []`), http.StatusForbidden)

	// cycle_start sent: any day of a week gives its Monday; null clears it and the first-focuses
	// rule does not start it again while the focuses stay.
	got = e.call(seed.UserJan, http.MethodPut, path, with(`"cycle_start": "2026-03-11"`), http.StatusOK)
	if got["cycle_start"] != "2026-03-09" || len(list(got["focuses"])) != 2 {
		t.Fatalf("explicit cycle_start = %v", got["cycle_start"])
	}
	got = e.call(seed.UserJan, http.MethodPut, path, with(`"cycle_start": null`), http.StatusOK)
	if got["cycle_start"] != nil || len(list(got["focuses"])) != 2 {
		t.Fatalf("cleared cycle_start = %v", got["cycle_start"])
	}
	// Removing all focuses and choosing new ones is "first set" again.
	got = e.call(seed.UserJan, http.MethodPut, path, with(`"focuses": []`), http.StatusOK)
	if len(list(got["focuses"])) != 0 || got["cycle_start"] != nil {
		t.Fatalf("cleared focuses = %v", got)
	}
	got = e.call(seed.UserJan, http.MethodPut, path, with(`"focuses": [{"key": "takt", "until": null}]`), http.StatusOK)
	if got["cycle_start"] != "2026-03-23" || obj(list(got["focuses"])[0])["until"] != nil {
		t.Fatalf("new focuses = %v", got)
	}

	// A focus above the horse's phase is accepted with a warning (Luna: Schubkraft, so only
	// Versammlung is above it); a hint for each of them names what it needs.
	got = e.call(seed.UserJan, http.MethodPut, path, with(`"focuses": [{"key": "versammlung"}, {"key": "turnier", "until": "2026-03-28"}]`), http.StatusOK)
	w := list(got["warnings"])
	if len(w) != 1 || obj(w[0])["key"] != "versammlung" || obj(w[0])["text"] != "Versammlung setzt Schwung und Geraderichtung voraus." {
		t.Fatalf("warnings = %v", got["warnings"])
	}
	// The same horse at level A is in the Gewöhnungsphase: the hints are the first three scale
	// steps, Geraderichtung and Seitengänge are above the phase.
	beginner := strings.Replace(with(`"focuses": [{"key": "geraderichten"}, {"key": "seitengaenge"}, {"key": "takt"}]`), `"level": "L"`, `"level": "A"`, 1)
	got = e.call(seed.UserJan, http.MethodPut, path, beginner, http.StatusOK)
	if got["phase"] != "gewoehnung" || !reflect.DeepEqual(strs(got["focus_hints"]), []string{"losgelassenheit", "anlehnung"}) {
		t.Fatalf("beginner = %v / %v", got["phase"], got["focus_hints"])
	}
	w = list(got["warnings"])
	if len(w) != 2 || obj(w[0])["key"] != "geraderichten" || obj(w[0])["text"] != "Geraderichtung setzt Anlehnung und Schwung voraus." ||
		obj(w[1])["key"] != "seitengaenge" || obj(w[1])["text"] != "Seitengänge setzen Losgelassenheit und Anlehnung voraus." {
		t.Fatalf("warnings = %v", got["warnings"])
	}
	// A horse that is not fit is in the Gewöhnungsphase whatever its level.
	got = e.call(seed.UserJan, http.MethodPut, path, strings.Replace(with(`"focuses": [{"key": "schwung"}]`), `"status": "fit"`, `"status": "pause"`, 1), http.StatusOK)
	if got["phase"] != "gewoehnung" || len(list(got["warnings"])) != 1 {
		t.Fatalf("paused horse = %v / %v", got["phase"], got["warnings"])
	}
	// A young horse too: Luna is 11, Nala born 2018 is 8; make Luna 4.
	e.exec(`UPDATE horses SET birth_year = 2022 WHERE id = $1`, luna)
	got = e.call(seed.UserJan, http.MethodPut, path, with(`"focuses": [{"key": "schwung"}]`), http.StatusOK)
	if got["phase"] != "gewoehnung" {
		t.Fatalf("young horse phase = %v", got["phase"])
	}
}

func TestProfileFocusValidation(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/training-profile"
	with := func(extra string) string {
		return strings.Replace(validProfile, `"status": "fit"`, `"status": "fit", `+extra, 1)
	}
	for name, body := range map[string]string{
		"four focuses":        with(`"focuses": [{"key":"takt"},{"key":"schwung"},{"key":"anlehnung"},{"key":"turnier"}]`),
		"unknown key":         with(`"focuses": [{"key":"tempo"}]`),
		"empty key":           with(`"focuses": [{"key":""}]`),
		"duplicate":           with(`"focuses": [{"key":"takt"},{"key":"takt","until":"2026-04-01"}]`),
		"until not a date":    with(`"focuses": [{"key":"takt","until":"28.03."}]`),
		"until with a time":   with(`"focuses": [{"key":"takt","until":"2026-03-28T10:00:00Z"}]`),
		"focuses not a list":  with(`"focuses": "takt"`),
		"focus unknown field": with(`"focuses": [{"key":"takt","weight":2}]`),
		"cycle_start":         with(`"cycle_start": "morgen"`),
		"cycle_start number":  with(`"cycle_start": 5`),
		"goal_level":          with(`"goal_level": "beginner"`),
		"goal_level other":    with(`"goal_level": "expert"`),
		"goal_until":          with(`"goal_until": "bald"`),
	} {
		if code := e.errCode(seed.UserJan, http.MethodPut, path, body, http.StatusBadRequest); code != "validation_failed" && code != "invalid_json" {
			t.Errorf("%s: code = %q", name, code)
		}
	}
	// Nothing of a refused request was stored.
	if got := e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK); len(list(got["focuses"])) != 0 || got["cycle_start"] != nil {
		t.Fatalf("a refused request changed the profile: %v", got)
	}
	// Three focuses are the limit; the order is kept.
	got := e.call(seed.UserJan, http.MethodPut, path, with(`"focuses": [{"key":"turnier","until":"2026-03-28"},{"key":"takt"},{"key":"anlehnung"}]`), http.StatusOK)
	var keys []string
	for _, f := range list(got["focuses"]) {
		keys = append(keys, str(obj(f)["key"]))
	}
	if !reflect.DeepEqual(keys, []string{"turnier", "takt", "anlehnung"}) {
		t.Fatalf("focuses = %v", keys)
	}
}

func TestMyFocus(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/my-focus"

	e.errCode("", http.MethodGet, path, "", http.StatusUnauthorized)
	// Owner and rider (legacy rule "ride") may; a member without role, the owner of another horse
	// and a rider without log_sessions and take_week_slots may not.
	got := e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK)
	if got["keys"] == nil || len(list(got["keys"])) != 0 || num(got["catalog_phase"]) != 2 || got["phase"] != "schubkraft" {
		t.Fatalf("empty focus = %v", got)
	}
	e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK)
	e.errCode(seed.UserSarah, http.MethodGet, path, "", http.StatusForbidden)
	e.errCode(seed.UserAnna, http.MethodGet, path, "", http.StatusForbidden)
	e.errCode(seed.UserSarah, http.MethodPut, path, `{"keys":["takt"]}`, http.StatusForbidden)
	e.errCode(seed.UserSarah, http.MethodDelete, path, "", http.StatusForbidden)
	e.exec(`UPDATE horse_riders SET rules = '["groom"]' WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia)
	e.errCode(seed.UserMia, http.MethodGet, path, "", http.StatusForbidden)
	e.errCode(seed.UserMia, http.MethodPut, path, `{"keys":["takt"]}`, http.StatusForbidden)
	for _, rule := range []string{"log_sessions", "take_week_slots"} {
		e.exec(`UPDATE horse_riders SET rules = $3::jsonb WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia, `["`+rule+`"]`)
		e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK)
	}
	e.errCode(seed.UserJan, http.MethodGet, "/api/v1/horses/00000000-0000-4000-8000-00000000ffff/my-focus", "", http.StatusNotFound)

	// Everybody keeps an own focus: two keys at most, known and distinct.
	got = e.call(seed.UserMia, http.MethodPut, path, `{"keys":["gelassenheit","takt"]}`, http.StatusOK)
	if !reflect.DeepEqual(strs(got["keys"]), []string{"gelassenheit", "takt"}) {
		t.Fatalf("keys = %v", got["keys"])
	}
	e.call(seed.UserJan, http.MethodPut, path, `{"keys":["geraderichten"]}`, http.StatusOK)
	if got := e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK); !reflect.DeepEqual(strs(got["keys"]), []string{"gelassenheit", "takt"}) {
		t.Fatalf("Mia's focus = %v", got["keys"])
	}
	if got := e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK); !reflect.DeepEqual(strs(got["keys"]), []string{"geraderichten"}) {
		t.Fatalf("Jan's focus = %v", got["keys"])
	}
	for name, body := range map[string]string{
		"three":     `{"keys":["takt","anlehnung","schwung"]}`,
		"unknown":   `{"keys":["tempo"]}`,
		"duplicate": `{"keys":["takt","takt"]}`,
		"no list":   `{"keys":"takt"}`,
		"extra":     `{"keys":["takt"],"horse":"x"}`,
	} {
		if code := e.errCode(seed.UserMia, http.MethodPut, path, body, http.StatusBadRequest); code != "validation_failed" && code != "invalid_json" {
			t.Errorf("%s: code = %q", name, code)
		}
	}
	// A new list replaces the old one; an empty list and DELETE clear it.
	got = e.call(seed.UserMia, http.MethodPut, path, `{"keys":["vertrauen"]}`, http.StatusOK)
	if !reflect.DeepEqual(strs(got["keys"]), []string{"vertrauen"}) {
		t.Fatalf("replaced keys = %v", got["keys"])
	}
	got = e.call(seed.UserMia, http.MethodPut, path, `{"keys":[]}`, http.StatusOK)
	if len(list(got["keys"])) != 0 || e.count(`SELECT count(*) FROM rider_focuses WHERE user_id = $1`, seed.UserMia) != 0 {
		t.Fatalf("empty list = %v", got)
	}
	e.call(seed.UserMia, http.MethodPut, path, `{"keys":["takt"]}`, http.StatusOK)
	if rec := e.do(seed.UserMia, http.MethodDelete, path, ""); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body)
	}
	if rec := e.do(seed.UserMia, http.MethodDelete, path, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete without a focus = %d", rec.Code)
	}
	if got := e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK); len(list(got["keys"])) != 0 {
		t.Fatalf("keys after delete = %v", got["keys"])
	}
	if got := e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK); len(list(got["keys"])) != 1 {
		t.Fatalf("Jan's focus was removed too: %v", got["keys"])
	}
	// The catalog phase follows the horse: a beginner may pick the first three steps.
	e.exec(`UPDATE training_profiles SET level = 'A' WHERE horse_id = $1`, luna)
	if got := e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK); num(got["catalog_phase"]) != 1 || got["phase"] != "gewoehnung" {
		t.Fatalf("beginner = %v", got)
	}
}

func TestSessionFocusKey(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/sessions"

	// Optional: older clients send none and get null back.
	old := obj(e.call(seed.UserJan, http.MethodPost, path, `{"activity":"hall","minutes":30,"started_at":"2026-03-23T17:00:00+01:00"}`, http.StatusCreated)["session"])
	if v, ok := old["focus_key"]; !ok || v != nil {
		t.Fatalf("session without focus_key = %v", old)
	}
	s := obj(e.call(seed.UserJan, http.MethodPost, path, `{"activity":"hall","minutes":45,"focus_key":"losgelassenheit"}`, http.StatusCreated)["session"])
	if s["focus_key"] != "losgelassenheit" {
		t.Fatalf("created = %v", s)
	}
	if e.count(`SELECT count(*) FROM sessions WHERE focus_key = 'losgelassenheit'`) != 1 {
		t.Fatal("focus_key not stored")
	}
	// A rider logs one too; the list returns it.
	e.call(seed.UserMia, http.MethodPost, path, `{"activity":"walker","minutes":30,"focus_key":"mobilitaet"}`, http.StatusCreated)
	keys := map[string]int{}
	for _, x := range list(e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK)["sessions"]) {
		if k, ok := obj(x)["focus_key"].(string); ok {
			keys[k]++
		}
	}
	if keys["losgelassenheit"] != 1 || keys["mobilitaet"] != 1 || len(keys) != 2 {
		t.Fatalf("listed focus keys = %v", keys)
	}
	// An empty string is "not given"; anything outside the catalog is refused.
	if e := obj(e.call(seed.UserJan, http.MethodPost, path, `{"activity":"hall","minutes":30,"focus_key":""}`, http.StatusCreated)["session"]); e["focus_key"] != nil {
		t.Fatalf("empty focus_key = %v", e["focus_key"])
	}
	for _, bad := range []string{`"tempo"`, `"Takt"`, `5`} {
		code := e.errCode(seed.UserJan, http.MethodPost, path, `{"activity":"hall","minutes":30,"focus_key":`+bad+`}`, http.StatusBadRequest)
		if code != "validation_failed" && code != "invalid_json" {
			t.Errorf("focus_key %s: code = %q", bad, code)
		}
	}
}
