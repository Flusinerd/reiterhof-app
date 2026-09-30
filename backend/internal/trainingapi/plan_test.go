package trainingapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/mistral"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// fakeChat stands in for the language model.
type fakeChat struct {
	answer string
	err    error
	calls  int
	system string
	user   string
}

func (f *fakeChat) CompleteJSON(_ context.Context, system, user string) (string, error) {
	f.calls++
	f.system, f.user = system, user
	return f.answer, f.err
}

func (e *env) withChat(c *fakeChat) {
	e.h = httpapi.NewHandler(httpapi.Deps{Pool: e.pool, Now: func() time.Time { return e.now }, Chat: c})
}

func (e *env) grantAI(user string) {
	e.exec(`INSERT INTO consents (user_id, stable_id, kind, version, granted_at) VALUES ($1, $2, 'ai_training', 'test', now())`, user, seed.StableB)
}

func planDays(out map[string]any) map[string]map[string]any {
	days := map[string]map[string]any{}
	for _, x := range list(out["days"]) {
		days[str(obj(x)["date"])] = obj(x)
	}
	return days
}

// planWeekFixture: Luna's profile allows hall and hack (conditional), a show on Saturday 28th;
// Monday and Tuesday are done, Mia planned a hack on Thursday. Today is Wednesday 25th.
func planWeekFixture(t *testing.T) *env {
	e := newEnv(t)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/training-profile", validProfile, http.StatusOK)
	e.insertSession(luna, seed.UserMia, "hall", time.Date(2026, 3, 23, 17, 0, 0, 0, berlin), 45, 36, true)
	e.insertSession(luna, seed.UserMia, "hack", time.Date(2026, 3, 24, 17, 0, 0, 0, berlin), 60, 60, true)
	e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, activity, status) VALUES ($1, $2, '2026-03-26', $3, 'hack', 'planned')`, seed.StableB, luna, seed.UserMia)
	return e
}

const planPath = "/api/v1/horses/" + luna + "/week/plan"

func TestPlanWeekWithRules(t *testing.T) {
	e := planWeekFixture(t)
	got := e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK)
	if got["ai_status"] != "not_configured" || got["source"] != "rules" || got["owner_is_me"] != true || got["start"] != "2026-03-23" {
		t.Fatalf("plan = %v", got)
	}
	days := planDays(got)
	// Wednesday (today), Friday and Saturday are open; Thursday is Mia's, Sunday the rest day after the show.
	if len(days) != 3 || days["2026-03-25"] == nil || days["2026-03-27"] == nil || days["2026-03-28"] == nil {
		t.Fatalf("days = %v", got["days"])
	}
	for date, d := range days {
		if d["source"] != "rules" || str(d["reason"]) == "" || str(d["label"]) == "" {
			t.Errorf("%s = %v", date, d)
		}
	}
	// The day before the show only allows light work, and only hall and hack are allowed: rest.
	if fri := days["2026-03-27"]; fri["activity"] != "rest" || fri["weekday"] != float64(4) {
		t.Errorf("friday = %v", fri)
	}

	// Only owners and admins plan; riders and other members may not.
	e.errCode(seed.UserMia, http.MethodPost, planPath, "", http.StatusForbidden)
	e.errCode(seed.UserSarah, http.MethodPost, planPath, "", http.StatusForbidden)
	e.errCode(seed.UserJan, http.MethodPost, planPath+"?start=morgen", "", http.StatusBadRequest)

	// A past week has nothing to plan.
	if got := e.call(seed.UserJan, http.MethodPost, planPath+"?start=2026-03-10", "", http.StatusOK); got["ai_status"] != "nothing_to_plan" || len(list(got["days"])) != 0 {
		t.Errorf("past week = %v", got)
	}
}

func TestPlanWeekWithModel(t *testing.T) {
	e := planWeekFixture(t)
	chat := &fakeChat{answer: `{"days":[
		{"day":"mi","activity":"hall","minutes":45,"reason":"Nach dem Ausritt gestern passt die Halle."},
		{"day":"fr","activity":"hack","minutes":60,"reason":"Ausritt vor dem Turnier."},
		{"day":"do","activity":"hall","minutes":30,"reason":"closed day"}]}`}
	e.withChat(chat)

	// Without the owner's consent nothing goes to the model.
	if got := e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK); got["ai_status"] != "no_consent" || chat.calls != 0 {
		t.Fatalf("no consent: %v, calls %d", got, chat.calls)
	}

	e.grantAI(seed.UserJan)
	got := e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK)
	if got["ai_status"] != "used" || got["source"] != "ai" || chat.calls != 1 {
		t.Fatalf("plan = %v, calls %d", got, chat.calls)
	}
	days := planDays(got)
	// After two working days (Monday 36, Tuesday 60) Wednesday may only be light: 45 minutes
	// in the hall are cut to 37 (JAN-93).
	if wed := days["2026-03-25"]; wed["source"] != "ai" || wed["activity"] != "hall" || wed["minutes"] != float64(37) ||
		wed["level"] != "light" || wed["level_label"] != "leicht" ||
		wed["reason"] != "Nach dem Ausritt gestern passt die Halle." || wed["note"] != nil {
		t.Errorf("wednesday = %v", wed)
	}
	if fri := days["2026-03-27"]; fri["source"] != "rules" || fri["activity"] != "rest" || !strings.Contains(str(fri["replaced"]), "Turnier ist morgen") {
		t.Errorf("friday = %v", fri)
	}
	if sat := days["2026-03-28"]; sat["source"] != "rules" || sat["replaced"] != nil {
		t.Errorf("saturday = %v", sat)
	}
	if days["2026-03-26"] != nil {
		t.Error("the model planned Mia's Thursday")
	}
	// The prompt carries no names, notes or dates.
	for _, leak := range []string{"Luna", "Jan", "Mia", "Anna", "Turnier", "Begleitung", "2026", seed.UserJan, luna} {
		if strings.Contains(chat.user, leak) {
			t.Errorf("prompt contains %q: %s", leak, chat.user)
		}
	}
	if !strings.Contains(chat.user, `"planned":"hack"`) || !strings.Contains(chat.user, `"conditional":true`) {
		t.Errorf("prompt = %s", chat.user)
	}

	// The model runs on the owner's consent only: Jan (admin) planning Anna's Fanta does not use it.
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+fanta+"/training-profile", validProfile, http.StatusOK)
	got = e.call(seed.UserJan, http.MethodPost, "/api/v1/horses/"+fanta+"/week/plan", "", http.StatusOK)
	if got["ai_status"] != "no_consent" || got["owner_is_me"] != false || chat.calls != 1 {
		t.Errorf("fanta = %v, calls %d", got, chat.calls)
	}

	// An owner under 16 (parental consent instead of the own statement) gets no model.
	e.exec(`UPDATE users SET age_confirmed_at = NULL, parental_consent_at = now() WHERE id = $1`, seed.UserJan)
	if got := e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK); got["ai_status"] != "owner_under_16" || got["source"] != "rules" || chat.calls != 1 {
		t.Errorf("owner under 16: %v, calls %d", got, chat.calls)
	}
	e.exec(`UPDATE users SET age_confirmed_at = now(), parental_consent_at = NULL WHERE id = $1`, seed.UserJan)

	// A revoked consent stops it again.
	e.exec(`UPDATE consents SET revoked_at = now() WHERE user_id = $1 AND kind = 'ai_training'`, seed.UserJan)
	if got := e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK); got["ai_status"] != "no_consent" || chat.calls != 1 {
		t.Errorf("revoked: %v", got)
	}
}

func TestPlanWeekModelFailureFallsBackToRules(t *testing.T) {
	e := planWeekFixture(t)
	e.grantAI(seed.UserJan)
	for _, tc := range []struct {
		chat *fakeChat
		want string
	}{
		{&fakeChat{err: mistral.ErrLimit}, "limit"},
		{&fakeChat{err: errors.New("mistral: HTTP 500")}, "failed"},
		{&fakeChat{answer: "Gern, hier ist dein Plan!"}, "failed"},
	} {
		e.withChat(tc.chat)
		got := e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK)
		if got["ai_status"] != tc.want || got["source"] != "rules" || len(list(got["days"])) != 3 {
			t.Errorf("%s: plan = %v", tc.want, got)
		}
	}
}

// exerciseKeyOf finds the key of a library exercise in the prompt sent to the model.
func exerciseKeyOf(t *testing.T, prompt, title string) string {
	t.Helper()
	var msg struct {
		Exercises []struct{ Key, Title, Library string }
	}
	if err := json.Unmarshal([]byte(prompt), &msg); err != nil {
		t.Fatal(err)
	}
	for _, e := range msg.Exercises {
		if e.Title == title {
			return e.Key
		}
	}
	t.Fatalf("exercise %q not in prompt", title)
	return ""
}

func TestPlanWeekFocusExerciseAndPromptData(t *testing.T) {
	e := planWeekFixture(t)
	e.grantAI(seed.UserJan)
	e.exec(`UPDATE horses SET birth_year = 2014 WHERE id = $1`, luna)
	e.exec(`INSERT INTO exercises (id, stable_id, discipline, level, title) VALUES (gen_random_uuid(), $1, 'dressage', 'beginner', 'Anna ihre Spezialübung')`, seed.StableB)
	chat := &fakeChat{answer: `{"days":[]}`}
	e.withChat(chat)
	e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK)

	var msg struct {
		Age       int    `json:"horse_age_years"`
		Level     string `json:"level"`
		Exercises []struct{ Library string }
	}
	if err := json.Unmarshal([]byte(chat.user), &msg); err != nil {
		t.Fatal(err)
	}
	// Level "L" of the fixture profile goes out as a library level, the age in years;
	// only global exercises of the libraries that fit the allowed activities (hall: dressage).
	if msg.Age != 12 || msg.Level != "intermediate" || len(msg.Exercises) == 0 {
		t.Fatalf("prompt = %s", chat.user)
	}
	for _, ex := range msg.Exercises {
		if ex.Library != "dressage" {
			t.Errorf("exercise of library %q sent", ex.Library)
		}
	}
	if strings.Contains(chat.user, "Spezialübung") || strings.Contains(chat.user, "L bei") {
		t.Errorf("own-stable exercise or free text in the prompt: %s", chat.user)
	}

	key := exerciseKeyOf(t, chat.user, "Übergänge")
	chat.answer = `{"days":[{"day":"mi","activity":"hall","minutes":45,"focus":"Übergänge Schritt-Trab","exercise":"` + key + `","reason":"Nach dem Ausritt gestern ruhige Arbeit in der Halle."}]}`
	got := e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK)
	wed := planDays(got)["2026-03-25"]
	if wed["source"] != "ai" || wed["focus"] != "Übergänge Schritt-Trab" || obj(wed["exercise"])["title"] != "Übergänge" {
		t.Fatalf("wednesday = %v", wed)
	}
	// Days from the rules carry the next exercise of their library, but no focus.
	if sat := planDays(got)["2026-03-28"]; sat["source"] != "rules" || sat["focus"] != nil {
		t.Errorf("saturday = %v", sat)
	}
}

func TestPlanSingleDay(t *testing.T) {
	e := planWeekFixture(t)
	wed, thu, fri, sat := "2026-03-25", "2026-03-26", "2026-03-27", "2026-03-28"
	at := func(day string) string { return planPath + "?start=2026-03-23&day=" + day }

	// Rules: only the asked day is planned, the rejected activities are not picked.
	got := e.call(seed.UserJan, http.MethodPost, at(wed), `{"exclude":["hall"]}`, http.StatusOK)
	days := planDays(got)
	if len(list(got["days"])) != 1 || days[wed] == nil || got["ai_status"] != "not_configured" || got["source"] != "rules" {
		t.Fatalf("plan = %v", got)
	}
	if days[wed]["activity"] == "hall" {
		t.Errorf("wednesday = %v", days[wed])
	}
	got = e.call(seed.UserJan, http.MethodPost, at(wed), `{"exclude":["hall","hack"]}`, http.StatusOK)
	if d := planDays(got)[wed]; d["activity"] != "rest" || d["reason"] != "Keine andere Aktivität passt heute." || d["minutes"] != float64(0) {
		t.Errorf("everything rejected: %v", d)
	}
	// An empty body, {} and a draft or exclude without day keep planning the whole week.
	for _, body := range []string{"", `{}`, `{"draft":[{"date":"2026-03-27","activity":"hall","minutes":60}],"exclude":["hall"]}`} {
		if got := e.call(seed.UserJan, http.MethodPost, planPath, body, http.StatusOK); len(list(got["days"])) != 3 {
			t.Errorf("body %q: days = %v", body, got["days"])
		}
		if got := e.call(seed.UserJan, http.MethodPost, at(wed), body, http.StatusOK); len(list(got["days"])) != 1 {
			t.Errorf("body %q with day: days = %v", body, got["days"])
		}
	}

	// Model: the draft of the other open days and the rejected activity reach the prompt;
	// the day Mia planned keeps its activity without minutes.
	e.grantAI(seed.UserJan)
	chat := &fakeChat{answer: `{"days":[
		{"day":"mi","activity":"hall","minutes":30,"reason":"Nach dem Ausritt gestern passt die Halle."},
		{"day":"fr","activity":"hall","minutes":30,"reason":"Nicht gefragt."}]}`}
	e.withChat(chat)
	got = e.call(seed.UserJan, http.MethodPost, at(wed),
		`{"draft":[{"date":"`+fri+`","activity":"hall","minutes":60},{"date":"`+sat+`","activity":"rest","minutes":0},{"date":"`+thu+`","activity":"hall","minutes":30}],"exclude":["hack"]}`,
		http.StatusOK)
	if got["ai_status"] != "used" || len(list(got["days"])) != 1 || chat.calls != 1 {
		t.Fatalf("plan = %v, calls %d", got, chat.calls)
	}
	if d := planDays(got)[wed]; d["source"] != "ai" || d["activity"] != "hall" {
		t.Errorf("wednesday = %v", d)
	}
	if n := strings.Count(chat.user, `"open":true`); n != 1 {
		t.Errorf("open days in the prompt = %d: %s", n, chat.user)
	}
	for _, want := range []string{`"planned":"hall","planned_minutes":60`, `"rest":true`, `"avoid":["hack"]`, `"planned":"hack"`} {
		if !strings.Contains(chat.user, want) {
			t.Errorf("prompt lacks %s: %s", want, chat.user)
		}
	}
	// Thursday is Mia's: the draft cannot change it.
	if strings.Contains(chat.user, `"planned_minutes":30`) {
		t.Errorf("the draft overrode a closed day: %s", chat.user)
	}
	// The model's proposal for a rejected activity is replaced by the rules.
	chat.answer = `{"days":[{"day":"mi","activity":"hall","minutes":30,"reason":"Halle"}]}`
	got = e.call(seed.UserJan, http.MethodPost, at(wed), `{"exclude":["hall"]}`, http.StatusOK)
	if d := planDays(got)[wed]; d["source"] != "rules" || d["replaced"] != "Vom Besitzer abgelehnt." || d["activity"] == "hall" {
		t.Errorf("rejected proposal = %v", d)
	}

	// Mia's planned day may be planned again; she stays entered.
	got = e.call(seed.UserJan, http.MethodPost, at(thu), `{}`, http.StatusOK)
	if d := planDays(got)[thu]; len(list(got["days"])) != 1 || d == nil || obj(d["user"])["name"] != "Mia" {
		t.Errorf("thursday = %v", got["days"])
	}

	// Days that cannot be planned: over, done, the rest day after the show, outside the week.
	for _, day := range []string{"2026-03-24", "2026-03-23", "2026-03-29", "2026-04-02", "2026-03-16"} {
		out := e.call(seed.UserJan, http.MethodPost, at(day), `{}`, http.StatusBadRequest)
		if er := obj(out["error"]); er["code"] != "validation_failed" || er["message"] != "day cannot be planned" {
			t.Errorf("%s: error = %v", day, out)
		}
	}
	e.errCode(seed.UserJan, http.MethodPost, at("morgen"), `{}`, http.StatusBadRequest)

	// Invalid drafts and exclusions.
	var eight []string
	for i := 0; i < 8; i++ {
		eight = append(eight, `{"date":"`+fri+`","activity":"hall","minutes":30}`)
	}
	for _, body := range []string{
		`{"draft":[{"date":"` + fri + `","activity":"hall","minutes":999}]}`,
		`{"draft":[{"date":"` + fri + `","activity":"hall","minutes":-1}]}`,
		`{"draft":[{"date":"2026-04-03","activity":"hall","minutes":30}]}`,
		`{"draft":[{"date":"morgen","activity":"hall","minutes":30}]}`,
		`{"draft":[{"date":"` + fri + `","activity":"swimming","minutes":30}]}`,
		`{"draft":[` + strings.Join(eight, ",") + `]}`,
		`{"exclude":["swimming"]}`,
	} {
		if code := e.errCode(seed.UserJan, http.MethodPost, at(wed), body, http.StatusBadRequest); code != "validation_failed" {
			t.Errorf("%s: code = %q", body, code)
		}
	}
	e.errCode(seed.UserJan, http.MethodPost, at(wed), `{"unknown":1}`, http.StatusBadRequest)

	// Only owners and admins plan.
	e.errCode(seed.UserMia, http.MethodPost, at(wed), `{}`, http.StatusForbidden)
}

func TestWeekSlotFocusAndExerciseSurviveAClaim(t *testing.T) {
	e := newEnv(t)
	day := "/api/v1/horses/" + luna + "/week/2026-03-26"
	var exID string
	if err := e.pool.QueryRow(context.Background(), `SELECT id::text FROM exercises WHERE stable_id IS NULL AND title = 'Übergänge'`).Scan(&exID); err != nil {
		t.Fatal(err)
	}
	e.errCode(seed.UserJan, http.MethodPut, day, `{"status":"planned","user_id":"","activity":"hall","exercise_id":"00000000-0000-4000-8000-00000000ffff"}`, http.StatusBadRequest)
	e.errCode(seed.UserJan, http.MethodPut, day, `{"status":"planned","user_id":"","focus":"`+strings.Repeat("x", 81)+`"}`, http.StatusBadRequest)

	got := e.call(seed.UserJan, http.MethodPut, day,
		`{"status":"planned","user_id":"","activity":"hall","focus":"Übergänge Schritt-Trab","exercise_id":"`+exID+`","note":"KI-Vorschlag: 45 Min."}`, http.StatusOK)
	thu := weekDays(got)[3]
	if thu["focus"] != "Übergänge Schritt-Trab" || obj(thu["exercise"])["id"] != exID || thu["activity"] != "hall" || thu["user"] != nil {
		t.Fatalf("planned thursday = %v", thu)
	}
	// Mia takes the day with "Ich": activity, note, focus and exercise stay.
	got = e.call(seed.UserMia, http.MethodPut, day, `{}`, http.StatusOK)
	thu = weekDays(got)[3]
	if obj(thu["user"])["name"] != "Mia" || thu["activity"] != "hall" || thu["focus"] != "Übergänge Schritt-Trab" ||
		obj(thu["exercise"])["title"] != "Übergänge" || str(thu["note"]) != "KI-Vorschlag: 45 Min." {
		t.Fatalf("claimed thursday = %v", thu)
	}
	// Missing keys and null keep focus and exercise, "" clears each of them on its own.
	got = e.call(seed.UserJan, http.MethodPut, day, `{"status":"planned","user_id":"","activity":"arena","focus":null,"exercise_id":null}`, http.StatusOK)
	if thu = weekDays(got)[3]; thu["user"] != nil || thu["activity"] != "arena" || thu["focus"] != "Übergänge Schritt-Trab" || obj(thu["exercise"])["id"] != exID {
		t.Fatalf("null keeps: %v", thu)
	}
	got = e.call(seed.UserJan, http.MethodPut, day, `{"status":"planned","user_id":"","focus":""}`, http.StatusOK)
	if thu = weekDays(got)[3]; thu["focus"] != nil || obj(thu["exercise"])["id"] != exID || thu["activity"] != "arena" {
		t.Fatalf("focus cleared: %v", thu)
	}
	got = e.call(seed.UserJan, http.MethodPut, day, `{"status":"planned","user_id":"","focus":"Dehnungshaltung"}`, http.StatusOK)
	if thu = weekDays(got)[3]; thu["focus"] != "Dehnungshaltung" || obj(thu["exercise"])["id"] != exID {
		t.Fatalf("focus set: %v", thu)
	}
	got = e.call(seed.UserJan, http.MethodPut, day, `{"status":"planned","user_id":"","exercise_id":""}`, http.StatusOK)
	if thu = weekDays(got)[3]; thu["exercise"] != nil || thu["focus"] != "Dehnungshaltung" || thu["activity"] != "arena" {
		t.Fatalf("exercise cleared: %v", thu)
	}
	got = e.call(seed.UserJan, http.MethodPut, day, `{"status":"planned","user_id":"","focus":"","exercise_id":"`+exID+`"}`, http.StatusOK)
	if thu = weekDays(got)[3]; thu["focus"] != nil || obj(thu["exercise"])["id"] != exID {
		t.Fatalf("focus cleared and exercise set in one request: %v", thu)
	}
	// A rest day has no activity, focus or exercise.
	got = e.call(seed.UserJan, http.MethodPut, day, `{"status":"rest"}`, http.StatusOK)
	if thu = weekDays(got)[3]; thu["status"] != "rest" || thu["focus"] != nil || thu["exercise"] != nil {
		t.Fatalf("rest thursday = %v", thu)
	}
	var focus, ex *string
	if err := e.pool.QueryRow(context.Background(), `SELECT focus, exercise_id::text FROM week_slots WHERE horse_id = $1 AND day = '2026-03-26'`, luna).Scan(&focus, &ex); err != nil || focus != nil || ex != nil {
		t.Fatalf("stored rest day: focus %v, exercise %v, err %v", focus, ex, err)
	}
}
