package trainingapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const validProfile = `{
  "discipline": "dressage", "level": "L",
  "allowed_activities": [
    {"activity": "hall", "mode": "on"},
    {"activity": "hack", "mode": "conditional", "note": "Nur mit Begleitung"},
    {"activity": "jumping", "mode": "off"}
  ],
  "shows": [{"date": "2026-03-28", "name": "Turnier", "classes": "Dressur L", "helper": "Anna"}],
  "season_end": "2026-10-31",
  "rhythm": {"sessions_min": 4, "sessions_max": 5, "rest_days_min": 1, "rest_days_max": 2, "max_minutes": 60},
  "status": "fit"
}`

func TestProfilePermissions(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/training-profile"

	e.call("", http.MethodGet, path, "", http.StatusUnauthorized)

	// owner, admin and rider may read
	own := e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK)
	if own["can_edit"] != true || own["discipline"] != "dressage" || len(list(own["rider_rules"])) != 1 {
		t.Fatalf("owner view = %v", own)
	}
	admin := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+fanta+"/training-profile", "", http.StatusOK)
	if admin["can_edit"] != true {
		t.Fatalf("admin can_edit = %v", admin["can_edit"])
	}
	rider := e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK)
	if rider["can_edit"] != false {
		t.Fatalf("rider can_edit = %v", rider["can_edit"])
	}
	if rules := list(rider["rider_rules"]); len(rules) != 1 || str(obj(rules[0])["user_id"]) != seed.UserMia {
		t.Fatalf("rider sees rules %v, want only her own", rider["rider_rules"])
	}

	// a plain stable member and the owner of another horse may not
	e.errCode(seed.UserSarah, http.MethodGet, path, "", http.StatusForbidden)
	e.errCode(seed.UserAnna, http.MethodGet, path, "", http.StatusForbidden)

	// writing: riders and members never, owners and admins do
	e.errCode(seed.UserMia, http.MethodPut, path, validProfile, http.StatusForbidden)
	e.errCode(seed.UserSarah, http.MethodPut, path, validProfile, http.StatusForbidden)
	e.errCode(seed.UserAnna, http.MethodPut, path, validProfile, http.StatusForbidden)
	e.call(seed.UserJan, http.MethodPut, path, validProfile, http.StatusOK)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+fanta+"/training-profile", validProfile, http.StatusOK) // admin
	e.call(seed.UserAnna, http.MethodPut, "/api/v1/horses/"+fanta+"/training-profile", validProfile, http.StatusOK)

	// unknown and malformed horse ids
	e.errCode(seed.UserJan, http.MethodGet, "/api/v1/horses/00000000-0000-4000-8000-00000000ffff/training-profile", "", http.StatusNotFound)
	e.errCode(seed.UserJan, http.MethodGet, "/api/v1/horses/not-a-uuid/training-profile", "", http.StatusNotFound)
}

func TestProfileOtherStableSeesNothing(t *testing.T) {
	e := newEnv(t)
	const otherStable, otherUser = "00000000-0000-4000-8000-0000000001f1", "00000000-0000-4000-8000-0000000002f1"
	e.exec(`INSERT INTO stables (id, name) VALUES ($1, 'Fremder Stall')`, otherStable)
	e.exec(`INSERT INTO users (id, stable_id, name, email, is_admin) VALUES ($1, $2, 'Fremd', 'fremd@example.org', true)`, otherUser, otherStable)

	// even an admin of another stable gets 404 (does not reveal the horse)
	for _, p := range []string{
		"/api/v1/horses/" + luna + "/training-profile", "/api/v1/horses/" + luna + "/today",
		"/api/v1/horses/" + luna + "/week", "/api/v1/horses/" + luna + "/sessions",
	} {
		e.errCode(otherUser, http.MethodGet, p, "", http.StatusNotFound)
	}
	e.errCode(otherUser, http.MethodPut, "/api/v1/horses/"+luna+"/training-profile", validProfile, http.StatusNotFound)
	e.errCode(otherUser, http.MethodPost, "/api/v1/horses/"+luna+"/sessions", `{"activity":"hall","minutes":30}`, http.StatusNotFound)
	if got := e.call(otherUser, http.MethodGet, "/api/v1/training/horses", "", http.StatusOK); len(list(got["horses"])) != 0 {
		t.Fatalf("horses of other stable = %v", got["horses"])
	}
}

func TestProfileValidation(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/training-profile"
	patch := func(old, new string) string { return strings.Replace(validProfile, old, new, 1) }

	bad := map[string]string{
		"discipline":         patch(`"dressage"`, `"polo"`),
		"missing discipline": patch(`"discipline": "dressage",`, ``),
		"mode":               patch(`"mode": "on"`, `"mode": "maybe"`),
		"conditional note":   patch(`, "note": "Nur mit Begleitung"`, ``),
		"activity":           patch(`"activity": "hall"`, `"activity": "rest"`),
		"duplicate":          patch(`"activity": "jumping"`, `"activity": "hall"`),
		"status":             patch(`"status": "fit"`, `"status": "sick"`),
		"show date":          patch(`2026-03-28`, `28.03.2026`),
		"show name":          patch(`"name": "Turnier"`, `"name": ""`),
		"season end":         patch(`2026-10-31`, `31.10.`),
		"sessions min":       patch(`"sessions_min": 4`, `"sessions_min": 0`),
		"sessions order":     patch(`"sessions_max": 5`, `"sessions_max": 3`),
		"rest days":          patch(`"rest_days_max": 2`, `"rest_days_max": 9`),
		"max minutes":        patch(`"max_minutes": 60`, `"max_minutes": 500`),
		"unknown field":      patch(`"level": "L"`, `"level": "L", "owner": "x"`),
		"not a rider":        patch(`"status": "fit"`, `"status": "fit", "rider_rules": [{"user_id": "`+seed.UserSarah+`"}]`),
		"rider activity":     patch(`"status": "fit"`, `"status": "fit", "rider_rules": [{"user_id": "`+seed.UserMia+`", "allowed_activities": ["rest"]}]`),
		"rider intensity":    patch(`"status": "fit"`, `"status": "fit", "rider_rules": [{"user_id": "`+seed.UserMia+`", "max_intensity": "extreme"}]`),
	}
	for name, body := range bad {
		if code := e.errCode(seed.UserJan, http.MethodPut, path, body, http.StatusBadRequest); code != "validation_failed" && code != "invalid_json" {
			t.Errorf("%s: code = %q", name, code)
		}
	}

	// A valid save round-trips; the rest day after a show stays on by default; rider rules
	// are kept when omitted and replaced when sent.
	got := e.call(seed.UserJan, http.MethodPut, path, validProfile, http.StatusOK)
	if r := obj(got["rhythm"]); r["rest_after_show"] != true || num(r["sessions_max"]) != 5 {
		t.Fatalf("rhythm = %v", r)
	}
	if rules := list(got["rider_rules"]); len(rules) != 1 || len(list(obj(rules[0])["allowed_activities"])) != 5 {
		t.Fatalf("omitted rider_rules must keep the stored rules: %v", got["rider_rules"])
	}
	if got["season_end"] != "2026-10-31" || len(list(got["shows"])) != 1 {
		t.Fatalf("profile = %v", got)
	}
	withRules := patch(`"status": "fit"`, `"status": "pause", "rider_rules": [{"user_id": "`+seed.UserMia+`", "allowed_activities": ["walker","walker"], "max_intensity": "light", "may_hack_alone": true}]`)
	got = e.call(seed.UserJan, http.MethodPut, path, withRules, http.StatusOK)
	rr := obj(list(got["rider_rules"])[0])
	if got["status"] != "pause" || len(list(rr["allowed_activities"])) != 1 || rr["max_intensity"] != "light" || rr["may_hack_alone"] != true {
		t.Fatalf("rider rules = %v", got)
	}
	// the rider sees her (new) rules read-only
	mia := e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK)
	if obj(list(mia["rider_rules"])[0])["max_intensity"] != "light" {
		t.Fatalf("rider view = %v", mia["rider_rules"])
	}
	// a horse without profile row gets defaults, and PUT creates the row
	nala := "/api/v1/horses/" + seed.HorseNala + "/training-profile"
	e.exec(`DELETE FROM training_profiles WHERE horse_id = $1`, seed.HorseNala)
	empty := e.call(seed.UserAnna, http.MethodGet, nala, "", http.StatusOK)
	if empty["exists"] != false || obj(empty["rhythm"])["rest_after_show"] != true || empty["status"] != "fit" {
		t.Fatalf("default profile = %v", empty)
	}
	e.call(seed.UserAnna, http.MethodPut, nala, validProfile, http.StatusOK)
	if e.call(seed.UserAnna, http.MethodGet, nala, "", http.StatusOK)["exists"] != true {
		t.Fatal("profile not created")
	}
}

func TestTrainingHorses(t *testing.T) {
	e := newEnv(t)
	names := func(user string) string {
		var out []string
		for _, h := range list(e.call(user, http.MethodGet, "/api/v1/training/horses", "", http.StatusOK)["horses"]) {
			out = append(out, str(obj(h)["name"])+":"+str(obj(h)["role"])+":"+str(obj(h)["profile_status"]))
		}
		return strings.Join(out, ",")
	}
	if got := names(seed.UserJan); got != "Luna:owner:fit" {
		t.Errorf("Jan = %q", got)
	}
	if got := names(seed.UserMia); got != "Luna:rider:fit" {
		t.Errorf("Mia = %q", got)
	}
	if got := names(seed.UserAnna); got != "Fanta:owner:reha,Nala:owner:fit" {
		t.Errorf("Anna = %q", got)
	}
	if got := names(seed.UserLea); got != "Fanta:rider:reha" {
		t.Errorf("Lea = %q", got)
	}
}

// profileWithRhythm is validProfile (hall, hack conditional, jumping off) with another rhythm.
func profileWithRhythm(rhythm string) string {
	return strings.Replace(validProfile,
		`"rhythm": {"sessions_min": 4, "sessions_max": 5, "rest_days_min": 1, "rest_days_max": 2, "max_minutes": 60}`,
		`"rhythm": `+rhythm, 1)
}

func TestProfileWeekStructure(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/training-profile"
	days := `[{"kind":"rest"},{"kind":"demanding"},{"kind":"recovery"},{"kind":""},{"kind":"normal"},{"kind":"activity","activity":"hack"},{"kind":"light"}]`
	got := e.call(seed.UserJan, http.MethodPut, path, profileWithRhythm(
		`{"sessions_min": 4, "sessions_max": 5, "rest_days_min": 1, "rest_days_max": 2, "max_minutes": 60, "days": `+days+`, "quotas": {"demanding": 2, "recovery": 1, "activities": {"hack": 1}}}`), http.StatusOK)
	rh := obj(got["rhythm"])
	d := list(rh["days"])
	if len(d) != 7 || obj(d[0])["kind"] != "rest" || obj(d[5])["activity"] != "hack" || obj(d[3])["kind"] != "" {
		t.Fatalf("days = %v", rh["days"])
	}
	if q := obj(rh["quotas"]); q["demanding"] != float64(2) || obj(q["activities"])["hack"] != float64(1) {
		t.Fatalf("quotas = %v", rh["quotas"])
	}
	if again := e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK); len(list(obj(again["rhythm"])["days"])) != 7 {
		t.Fatalf("stored rhythm = %v", again["rhythm"])
	}

	for name, rhythm := range map[string]string{
		"unknown kind":          `{"days": [{"kind":"sleepy"},{},{},{},{},{},{}]}`,
		"six days":              `{"days": [{},{},{},{},{},{}]}`,
		"activity not allowed":  `{"days": [{"kind":"activity","activity":"jumping"},{},{},{},{},{},{}]}`,
		"activity without kind": `{"days": [{"kind":"rest","activity":"hall"},{},{},{},{},{},{}]}`,
		"too many rest days":    `{"rest_days_max": 1, "days": [{"kind":"rest"},{"kind":"rest"},{},{},{},{},{}]}`,
		"quota not allowed":     `{"quotas": {"activities": {"jumping": 1}}}`,
		"quotas overfull":       `{"rest_days_min": 2, "quotas": {"demanding": 3, "recovery": 2, "activities": {"hall": 1}}}`,
	} {
		if code := e.errCode(seed.UserJan, http.MethodPut, path, profileWithRhythm(rhythm), http.StatusBadRequest); code != "validation_failed" {
			t.Errorf("%s: code = %q", name, code)
		}
	}
}
