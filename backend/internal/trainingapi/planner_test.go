package trainingapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// The unit planner behind "Was heute?" and the week plan: focus rotation, unit, block, return to
// work and equipment as the API shows them. The pure rules are tested in training/recommend and
// training/weekplan; these tests check that the API feeds them with the right data.

// plannerProfile: a dressage horse of class L that rides in the hall, the arena and on the lunge,
// without a show, with a low minimum of sessions so that the blocks do not restart on their own.
func plannerProfile(extra string) string {
	body := `{
  "discipline": "dressage", "level": "L",
  "allowed_activities": [{"activity": "hall", "mode": "on"}, {"activity": "arena", "mode": "on"}, {"activity": "lunge", "mode": "on"}],
  "shows": [],
  "rhythm": {"sessions_min": 1, "sessions_max": 6, "rest_days_min": 0, "rest_days_max": 2, "max_minutes": 60},
  "status": "fit"`
	if extra != "" {
		body += ", " + extra
	}
	return body + "}"
}

const lunaProfilePath = "/api/v1/horses/" + luna + "/training-profile"
const todayPath = "/api/v1/horses/" + luna + "/today"

// recFor returns the recommendation of an activity.
func recFor(t *testing.T, got map[string]any, activity string) map[string]any {
	t.Helper()
	for _, r := range recs(got) {
		if r["activity"] == activity {
			return r
		}
	}
	t.Fatalf("no recommendation for %s: %v", activity, got["recommendations"])
	return nil
}

func partMinutes(u map[string]any) [3]float64 {
	return [3]float64{num(obj(u["warmup"])["minutes"]), num(obj(u["main"])["minutes"]), num(obj(u["cooldown"])["minutes"])}
}

func TestTodayHasFocusAndUnit(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	e.call(seed.UserJan, http.MethodPut, lunaProfilePath, plannerProfile(`"focuses": [{"key": "losgelassenheit"}, {"key": "takt"}]`), http.StatusOK)
	got := e.call(seed.UserJan, http.MethodGet, todayPath+"?minutes=45", "", http.StatusOK)
	hall := recFor(t, got, "hall")
	if f := obj(hall["focus"]); f["key"] != "losgelassenheit" || f["label"] != "Losgelassenheit" || f["why"] != "Schwerpunkt Losgelassenheit: noch nicht geübt." {
		t.Fatalf("focus = %v", hall["focus"])
	}
	u := obj(hall["unit"])
	if m := partMinutes(u); m != [3]float64{15, 22, 8} {
		t.Fatalf("minutes = %v", m)
	}
	if w := obj(u["warmup"]); obj(w["exercise"])["title"] != "Aufwärmen im Schritt" || num(w["exercise_minutes"]) != 10 {
		t.Errorf("warm-up = %v", w)
	}
	if m := obj(u["main"]); obj(m["exercise"])["title"] != "Schlangenlinien" || num(m["exercise_minutes"]) != 12 {
		t.Errorf("main = %v", m)
	}
	if c := obj(u["cooldown"]); obj(c["exercise"])["title"] != "Zügel aus der Hand" || c["exercise_minutes"] != nil {
		t.Errorf("cool-down = %v", c)
	}
	// "exercise" is the main exercise with its steps.
	if ex := obj(hall["exercise"]); ex["title"] != "Schlangenlinien" || len(list(ex["steps"])) == 0 {
		t.Errorf("exercise = %v", hall["exercise"])
	}
	// No set-up question for the stable: nothing to say about equipment.
	if hall["equipment_ok"] != nil || hall["equipment_note"] != nil {
		t.Errorf("equipment = %v / %v", hall["equipment_ok"], hall["equipment_note"])
	}
	// The horse's phase, no break, the first week of the blocks that the first focuses started, no
	// goal lessons in the library.
	if b := obj(got["block"]); got["phase"] != "schubkraft" || num(got["restart_week"]) != 0 || b["week"] != 1.0 || b["deload"] != false || got["progress"] != nil {
		t.Errorf("phase %v, restart_week %v, block %v, progress %v", got["phase"], got["restart_week"], got["block"], got["progress"])
	}
	// Every activity with a unit has the three parts; the lunge has its own minutes.
	if lunge := recFor(t, got, "lunge"); lunge["focus"] == nil || lunge["unit"] == nil {
		t.Errorf("lunge = %v", lunge)
	}
}

func TestTodayRotatesTheFocusesWithTheHistory(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	e.call(seed.UserJan, http.MethodPut, lunaProfilePath, plannerProfile(`"focuses": [{"key": "losgelassenheit"}, {"key": "takt"}]`), http.StatusOK)
	focusOf := func() string {
		return str(obj(recFor(t, e.call(seed.UserJan, http.MethodGet, todayPath+"?minutes=45", "", http.StatusOK), "hall")["focus"])["key"])
	}
	if got := focusOf(); got != "losgelassenheit" {
		t.Fatalf("first focus = %s", got)
	}
	// A session of yesterday with a focus: the other one is next.
	e.call(seed.UserJan, http.MethodPost, "/api/v1/horses/"+luna+"/sessions",
		`{"activity":"arena","minutes":40,"started_at":"2026-03-24T17:00:00+01:00","focus_key":"losgelassenheit"}`, http.StatusCreated)
	if got := focusOf(); got != "takt" {
		t.Errorf("after losgelassenheit: %s", got)
	}
	// The focus of a session without one comes from the day's slot (the unit that was planned).
	e.exec(`DELETE FROM sessions`)
	e.exec(`DELETE FROM week_slots`)
	e.insertSession(luna, seed.UserJan, "arena", time.Date(2026, 3, 24, 17, 0, 0, 0, berlin), 40, 32, true)
	if got := focusOf(); got != "losgelassenheit" {
		t.Errorf("a session without a focus: %s", got)
	}
	e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, activity, status, focus_key) VALUES ($1, $2, '2026-03-24', $3, 'arena', 'done', 'losgelassenheit')`,
		seed.StableB, luna, seed.UserJan)
	if got := focusOf(); got != "takt" {
		t.Errorf("a session whose slot has the focus: %s", got)
	}
}

func TestTodayRiderFocus(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	e.call(seed.UserJan, http.MethodPut, lunaProfilePath, plannerProfile(`"focuses": [{"key": "losgelassenheit"}]`), http.StatusOK)
	e.call(seed.UserMia, http.MethodPut, "/api/v1/horses/"+luna+"/my-focus", `{"keys":["takt"]}`, http.StatusOK)
	// Mia's own focus comes first for her, the owner gets the horse's.
	mia := recFor(t, e.call(seed.UserMia, http.MethodGet, todayPath+"?minutes=45", "", http.StatusOK), "hall")
	if f := obj(mia["focus"]); f["key"] != "takt" || f["why"] != "Mias Schwerpunkt Takt." {
		t.Errorf("Mia's focus = %v", mia["focus"])
	}
	jan := recFor(t, e.call(seed.UserJan, http.MethodGet, todayPath+"?minutes=45", "", http.StatusOK), "hall")
	if f := obj(jan["focus"]); f["key"] != "losgelassenheit" {
		t.Errorf("Jan's focus = %v", jan["focus"])
	}
}

func TestTodayEntlastungswocheAndReturnToWork(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	e.call(seed.UserJan, http.MethodPut, lunaProfilePath, plannerProfile(""), http.StatusOK)
	today := func() map[string]any {
		return e.call(seed.UserJan, http.MethodGet, todayPath, "", http.StatusOK)
	}
	// Three weeks with a hall session of load 40 each, a session in each so that no block restarts.
	for _, d := range []int{3, 10, 17} {
		e.insertSession(luna, seed.UserJan, "hall", time.Date(2026, 3, d, 17, 0, 0, 0, berlin), 50, 40, true)
	}
	// An ordinary week: the default 45 minutes of hall work.
	got := today()
	if got["block"] != nil || num(recFor(t, got, "hall")["minutes"]) != 45 {
		t.Fatalf("without blocks: %v", got["recommendations"])
	}
	// The fourth week of a block is the Entlastungswoche: 0.8 of the 40 load a week is 32, a hall
	// unit of 40 minutes (the most the load allows).
	e.exec(`UPDATE training_profiles SET cycle_start = '2026-03-02' WHERE horse_id = $1`, luna)
	got = today()
	if b := obj(got["block"]); b["week"] != 4.0 || b["deload"] != true {
		t.Fatalf("block = %v", got["block"])
	}
	if m := num(recFor(t, got, "hall")["minutes"]); m != 40 {
		t.Errorf("Entlastungswoche: hall = %v minutes, want 40", m)
	}
	// With little history the cap is lower than any unit: a rest day with the reason.
	e.exec(`UPDATE sessions SET duration_min = 10, load_score = 8`)
	got = today()
	if r := recs(got); len(r) != 1 || r[0]["activity"] != "rest" || !strings.Contains(str(r[0]["reason"]), "Entlastungswoche: die Wochenlast bleibt unter 80 %") {
		t.Errorf("Entlastungswoche without load left: %v", got["recommendations"])
	}

	// A break of more than two weeks: the first week back has light units only.
	e.exec(`UPDATE training_profiles SET cycle_start = NULL WHERE horse_id = $1`, luna)
	e.exec(`DELETE FROM sessions`)
	e.insertSession(luna, seed.UserJan, "hall", time.Date(2026, 3, 5, 17, 0, 0, 0, berlin), 40, 32, true)
	got = today()
	if num(got["restart_week"]) != 1 || got["block"] != nil {
		t.Fatalf("restart_week = %v, block = %v", got["restart_week"], got["block"])
	}
	if m := num(recFor(t, got, "hall")["minutes"]); m != 37 {
		t.Errorf("first week back: hall = %v minutes, light only (37)", m)
	}
	// After a session today the horse is back: the week counts from that day on, still week 1.
	e.insertSession(luna, seed.UserJan, "hall", time.Date(2026, 3, 25, 8, 0, 0, 0, berlin), 30, 24, true)
	if got := today(); num(got["restart_week"]) != 1 {
		t.Errorf("restart_week after the first session = %v", got["restart_week"])
	}
	// A recent history is no break.
	e.exec(`DELETE FROM sessions`)
	e.insertSession(luna, seed.UserJan, "hall", time.Date(2026, 3, 20, 17, 0, 0, 0, berlin), 40, 32, true)
	if got := today(); num(got["restart_week"]) != 0 {
		t.Errorf("restart_week = %v without a break", got["restart_week"])
	}
}

func TestTodayYoungHorseUnits(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	e.call(seed.UserJan, http.MethodPut, lunaProfilePath, plannerProfile(""), http.StatusOK)
	e.exec(`UPDATE horses SET birth_year = 2022 WHERE id = $1`, luna) // 4 years old
	got := e.call(seed.UserJan, http.MethodGet, todayPath+"?minutes=60", "", http.StatusOK)
	if got["phase"] != "gewoehnung" {
		t.Errorf("phase = %v", got["phase"])
	}
	for _, r := range recs(got) {
		if num(r["minutes"]) > 30 {
			t.Errorf("young horse: %v", r)
		}
	}
}

func TestTodayEquipment(t *testing.T) {
	e := newEnv(t)
	e.clearLibrary()
	e.addExercise(exSpec{id: exID(1), library: "cavaletti", level: "beginner", title: "Trabstangen", tags: []string{"losgelassenheit", "biegung", "takt"}, equipment: []string{"poles"}})
	e.addExercise(exSpec{id: exID(2), library: "dressage", level: "beginner", title: "Schlangenlinien", tags: []string{"losgelassenheit"}})
	e.call(seed.UserJan, http.MethodPut, lunaProfilePath, plannerProfile(`"focuses": [{"key": "losgelassenheit"}]`), http.StatusOK)
	hall := func() map[string]any {
		return recFor(t, e.call(seed.UserJan, http.MethodGet, todayPath+"?minutes=45", "", http.StatusOK), "hall")
	}
	e.exec(`UPDATE stables SET training_equipment = '{}'`)
	// The default hall has poles: the best exercise.
	if ex := obj(hall()["exercise"]); ex["title"] != "Trabstangen" {
		t.Fatalf("with poles: %v", ex)
	}
	// A hall without poles: the other exercise, and the unit says why.
	e.call(seed.UserJan, http.MethodPut, equipmentPath, `{"locations":{"hall":{"equipment":[],"shared":false}}}`, http.StatusOK)
	r := hall()
	main := obj(obj(r["unit"])["main"])
	if obj(r["exercise"])["title"] != "Schlangenlinien" || obj(main["instead_of"])["title"] != "Trabstangen" ||
		main["why"] != "Ohne Stangen in der Halle: Schlangenlinien statt Trabstangen." || r["equipment_note"] != main["why"] {
		t.Fatalf("without poles: %v", r)
	}
	// A shared hall is planned without set-up; the day can say that it can be set up.
	e.call(seed.UserJan, http.MethodPut, equipmentPath, `{"locations":{"hall":{"equipment":["poles"],"shared":true}}}`, http.StatusOK)
	r = hall()
	if obj(r["exercise"])["title"] != "Schlangenlinien" || !strings.HasPrefix(str(r["equipment_note"]), "Halle geteilt") {
		t.Fatalf("shared hall: %v", r)
	}
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/week/2026-03-25", `{"status":"planned","user_id":"","equipment_ok":true}`, http.StatusOK)
	r = hall()
	if obj(r["exercise"])["title"] != "Trabstangen" || r["equipment_ok"] != true || r["equipment_note"] != nil {
		t.Fatalf("shared hall with set-up: %v", r)
	}
	// And a day without set-up in a hall with poles.
	e.call(seed.UserJan, http.MethodPut, equipmentPath, `{"locations":{"hall":{"equipment":["poles"],"shared":false}}}`, http.StatusOK)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/week/2026-03-25", `{"status":"planned","user_id":"","equipment_ok":false}`, http.StatusOK)
	r = hall()
	if obj(r["exercise"])["title"] != "Schlangenlinien" || r["equipment_ok"] != false || r["equipment_note"] != "Heute ohne Aufbau geplant." {
		t.Fatalf("a day without set-up: %v", r)
	}
}

func TestPlanWeekUnitFields(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	e.call(seed.UserJan, http.MethodPut, lunaProfilePath, plannerProfile(`"focuses": [{"key": "losgelassenheit"}, {"key": "takt"}]`), http.StatusOK)
	e.call(seed.UserMia, http.MethodPut, "/api/v1/horses/"+luna+"/my-focus", `{"keys":["takt"]}`, http.StatusOK)
	// Mia claims today without an activity: an open day with her focus, which wins the tie of the
	// first unit. Thursday has the equipment override.
	e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, status) VALUES ($1, $2, '2026-03-25', $3, 'planned')`, seed.StableB, luna, seed.UserMia)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/week/2026-03-26", `{"status":"planned","user_id":"","equipment_ok":false}`, http.StatusOK)

	got := e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK)
	days := planDays(got)
	if got["block"] == nil {
		t.Fatalf("the first focuses start the blocks: %v", got["block"])
	}
	if b := obj(got["block"]); b["week"] != 1.0 || b["deload"] != false {
		t.Errorf("block = %v", b)
	}
	units := 0
	for date, d := range days {
		if d["activity"] == "rest" || d["activity"] == "walker" {
			if d["focus_key"] != nil || d["unit"] != nil || d["focus_why"] != nil {
				t.Errorf("%s: %v has no unit", date, d["activity"])
			}
			continue
		}
		units++
		u := obj(d["unit"])
		if d["focus_key"] == nil || d["focus_label"] == nil || str(d["focus_why"]) == "" || u["warmup"] == nil || u["main"] == nil || u["cooldown"] == nil {
			t.Errorf("%s: %v", date, d)
		}
		if m := partMinutes(u); m[0]+m[1]+m[2] != num(d["minutes"]) {
			t.Errorf("%s: the parts %v take %v minutes", date, m, d["minutes"])
		}
		// The main exercise of the unit is the day's exercise.
		if obj(obj(u["main"])["exercise"])["id"] != obj(d["exercise"])["id"] {
			t.Errorf("%s: exercise %v, main %v", date, d["exercise"], obj(u["main"])["exercise"])
		}
	}
	if units < 3 {
		t.Fatalf("days = %v", got["days"])
	}
	// The claimed day has Mia's focus, the day with the override says it has no set-up.
	if wed := days["2026-03-25"]; wed["activity"] == "rest" || wed["focus_key"] != "takt" || wed["focus_why"] != "Mias Schwerpunkt Takt." {
		t.Errorf("Wednesday = %v", wed)
	}
	if thu := days["2026-03-26"]; thu != nil && thu["activity"] != "rest" {
		if thu["equipment_ok"] != false || thu["equipment_note"] != "Heute ohne Aufbau geplant." {
			t.Errorf("Thursday = %v", thu)
		}
	}
}

func TestPlanWeekPromptHasPlanningDataAndNoNames(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	e.grantAI(seed.UserJan)
	e.addExercise(exSpec{id: exID(9), library: "dressage", level: "beginner", title: "Annas Geheimübung", tags: []string{"losgelassenheit"}, own: true})
	e.call(seed.UserJan, http.MethodPut, lunaProfilePath, plannerProfile(`"focuses": [{"key": "losgelassenheit"}, {"key": "versammlung", "until": "2026-04-30"}]`), http.StatusOK)
	e.call(seed.UserMia, http.MethodPut, "/api/v1/horses/"+luna+"/my-focus", `{"keys":["takt"]}`, http.StatusOK)
	e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, status) VALUES ($1, $2, '2026-03-27', $3, 'planned')`, seed.StableB, luna, seed.UserMia)
	chat := &fakeChat{answer: `{"days":[]}`}
	e.withChat(chat)
	e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK)

	var msg struct {
		Phase        string `json:"phase"`
		GoalLevel    string `json:"goal_level"`
		Block        map[string]any
		FocusCatalog []map[string]any `json:"focus_catalog"`
		HorseFocuses []map[string]any `json:"horse_focuses"`
		Exercises    []map[string]any
		Days         []map[string]any
		Activities   []map[string]any
	}
	if err := json.Unmarshal([]byte(chat.user), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Phase != "schubkraft" || msg.GoalLevel != "advanced" || msg.Block["week"] != 1.0 || len(msg.FocusCatalog) != 15 {
		t.Errorf("prompt = %s", chat.user)
	}
	// Versammlung is a phase 3 focus for a horse in phase 2.
	if len(msg.HorseFocuses) != 2 || msg.HorseFocuses[0]["key"] != "losgelassenheit" || msg.HorseFocuses[0]["above_phase"] != nil ||
		msg.HorseFocuses[1]["key"] != "versammlung" || msg.HorseFocuses[1]["above_phase"] != true {
		t.Errorf("horse_focuses = %v", msg.HorseFocuses)
	}
	if len(msg.Exercises) != 4 || msg.Exercises[0]["phase"] == nil || msg.Exercises[1]["key"] != "E2" {
		t.Errorf("exercises = %v", msg.Exercises)
	}
	var fri map[string]any
	for _, d := range msg.Days {
		if d["day"] == "fr" {
			fri = d
		}
	}
	if rf := list(fri["rider_focuses"]); len(rf) != 1 || rf[0] != "takt" {
		t.Errorf("Friday = %v", fri)
	}
	// Equipment per activity: the stable's defaults.
	for _, a := range msg.Activities {
		if a["activity"] == "hall" && len(list(a["available_equipment"])) != 1 {
			t.Errorf("hall = %v", a)
		}
	}
	for _, leak := range []string{"Luna", "Mia", "Anna", "Geheimübung", exID(9), "2026-"} {
		if strings.Contains(chat.user, leak) {
			t.Errorf("prompt contains %q", leak)
		}
	}
}

func TestPlanWeekModelProposalIsPlannedAsAUnit(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	// Two exercises for the focus takt: the rules take the first ("Taktwechsel"), the model the other.
	e.addExercise(exSpec{id: exID(5), library: "dressage", level: "beginner", title: "Taktwechsel", tags: []string{"takt"}})
	e.addExercise(exSpec{id: exID(6), library: "dressage", level: "beginner", title: "Trabtakt", tags: []string{"takt"}})
	e.grantAI(seed.UserJan)
	e.call(seed.UserJan, http.MethodPut, lunaProfilePath, plannerProfile(`"focuses": [{"key": "losgelassenheit"}, {"key": "takt"}]`), http.StatusOK)
	chat := &fakeChat{answer: `{"days":[]}`}
	e.withChat(chat)
	e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK)
	var msg struct{ Exercises []struct{ Key, Title string } }
	if err := json.Unmarshal([]byte(chat.user), &msg); err != nil {
		t.Fatal(err)
	}
	keyOf := func(title string) string {
		for _, x := range msg.Exercises {
			if x.Title == title {
				return x.Key
			}
		}
		t.Fatalf("no exercise %s", title)
		return ""
	}
	chat.answer = `{"days":[
		{"day":"mi","activity":"hall","minutes":45,"focus_key":"takt","focus":"Takt im Trab","warmup":"` + keyOf("Aufwärmen im Schritt") + `","main":"` + keyOf("Trabtakt") + `","cooldown":"` + keyOf("Zügel aus der Hand") + `","reason":"Takt üben."},
		{"day":"do","activity":"arena","minutes":45,"focus_key":"versammlung","main":"` + keyOf("Aufwärmen im Schritt") + `","reason":"Falsche Teile."}]}`
	got := e.call(seed.UserJan, http.MethodPost, planPath, "", http.StatusOK)
	days := planDays(got)
	wed := days["2026-03-25"]
	if wed["source"] != "ai" || wed["focus_key"] != "takt" || wed["focus"] != "Takt im Trab" || wed["replaced"] != nil {
		t.Fatalf("wednesday = %v", wed)
	}
	u := obj(wed["unit"])
	if obj(obj(u["warmup"])["exercise"])["title"] != "Aufwärmen im Schritt" || obj(obj(u["main"])["exercise"])["title"] != "Trabtakt" ||
		obj(obj(u["cooldown"])["exercise"])["title"] != "Zügel aus der Hand" || obj(wed["exercise"])["title"] != "Trabtakt" {
		t.Errorf("unit = %v", u)
	}
	// A warm-up exercise as the main exercise and a focus that is none of the horse's: the rules pick.
	thu := days["2026-03-26"]
	if thu["source"] != "ai" || thu["focus_key"] == "versammlung" || !strings.Contains(str(thu["replaced"]), "Übung „Aufwärmen im Schritt“ ersetzt: gehört nicht in diesen Teil der Einheit.") {
		t.Errorf("thursday = %v", thu)
	}
}

func TestProfileQuotaLimits(t *testing.T) {
	e := newEnv(t)
	withJumping := `{
  "discipline": "jumping", "level": "L",
  "allowed_activities": [{"activity": "hall", "mode": "on"}, {"activity": "jumping", "mode": "on"}, {"activity": "hack", "mode": "on"}],
  "shows": [],
  "rhythm": {"sessions_min": 3, "sessions_max": 6, "rest_days_min": 1, "rest_days_max": 2, "max_minutes": 60, "quotas": %s},
  "status": "fit"
}`
	put := func(quotas string, want int) map[string]any {
		return e.call(seed.UserJan, http.MethodPut, lunaProfilePath, strings.Replace(withJumping, "%s", quotas, 1), want)
	}
	put(`{"demanding": 2, "activities": {"jumping": 2}}`, http.StatusOK)
	put(`{"demanding": 0}`, http.StatusOK)
	if got := put(`{"demanding": 3}`, http.StatusBadRequest); !strings.Contains(str(obj(got["error"])["message"]), "quotas.demanding must be between 0 and 2") {
		t.Errorf("demanding 3: %v", got)
	}
	if got := put(`{"activities": {"jumping": 3}}`, http.StatusBadRequest); !strings.Contains(str(obj(got["error"])["message"]), "jumping") {
		t.Errorf("jumping 3: %v", got)
	}
	// Other activities may have more.
	put(`{"activities": {"hack": 3}}`, http.StatusOK)
}

func TestTodayLevelGoalAndProgress(t *testing.T) {
	e := newEnv(t)
	e.clearLibrary()
	// Class L (intermediate): the goal is the class above. Of its lessons one is mastered.
	e.addExercise(exSpec{id: exID(1), library: "dressage", level: "intermediate", title: "Volte", tags: []string{"losgelassenheit"}})
	e.addExercise(exSpec{id: exID(2), library: "dressage", level: "advanced", title: "Piaffe", tags: []string{"losgelassenheit"}})
	e.addExercise(exSpec{id: exID(3), library: "dressage", level: "advanced", title: "Passage", tags: []string{"versammlung"}, requires: []string{exID(2)}})
	e.addExercise(exSpec{id: exID(4), library: "dressage", level: "advanced", title: "Pirouette", tags: []string{"versammlung"}})
	e.call(seed.UserJan, http.MethodPut, lunaProfilePath, plannerProfile(`"focuses": [{"key": "losgelassenheit"}]`), http.StatusOK)
	e.rate(luna, exID(4), 3, "2026-03-20", "hall") // a recent session: no break, no return to work

	got := e.call(seed.UserJan, http.MethodGet, todayPath+"?minutes=45", "", http.StatusOK)
	p := obj(got["progress"])
	if p["text"] != "Auf dem Weg zu M: 1 von 3 Lektionen gefestigt." {
		t.Errorf("progress = %v", got["progress"])
	}
	// The next lessons are the ones whose prerequisites sit: Passage needs Piaffe first.
	next := list(p["next"])
	if len(next) != 1 || obj(next[0])["title"] != "Piaffe" || obj(next[0])["id"] != exID(2) {
		t.Errorf("next = %v", p["next"])
	}
	// The main exercise of the hall unit is the lesson of the goal level, with the reason.
	main := obj(obj(recFor(t, got, "hall")["unit"])["main"])
	if obj(main["exercise"])["title"] != "Piaffe" || main["why"] != "Richtung M: Piaffe ist die nächste Lektion." {
		t.Errorf("main = %v", main)
	}
	// A young horse keeps to its level.
	e.exec(`UPDATE horses SET birth_year = 2022 WHERE id = $1`, luna)
	got = e.call(seed.UserJan, http.MethodGet, todayPath+"?minutes=25", "", http.StatusOK)
	main = obj(obj(recFor(t, got, "hall")["unit"])["main"])
	if obj(main["exercise"])["title"] == "Piaffe" || main["why"] != nil {
		t.Errorf("a young horse: %v", main)
	}
}
