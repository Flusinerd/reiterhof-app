package weekplan

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/recommend"
)

// today is Wednesday 2026-09-30; the week runs from Monday 28th to Sunday 4th.
var today = time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

func day(offset int) time.Time { return today.AddDate(0, 0, offset) }

func allOn() []training.AllowedActivity {
	var l []training.AllowedActivity
	for _, a := range training.AllActivities {
		l = append(l, training.AllowedActivity{Activity: a, Mode: training.ModeOn})
	}
	return l
}

// week: Mon/Tue done (sessions), Wed (today) to Sun open.
func week() Input {
	in := Input{
		Today: today,
		Profile: training.Profile{
			HorseName: "Luna", Discipline: "dressage", Level: "L bei Anna Muster",
			Allowed: allOn(), Rhythm: training.DefaultRhythm(), Status: training.StatusFit,
			Shows: []training.Show{{Date: day(10), Name: "Turnier bei Familie Muster"}},
		},
		Recent: []training.Session{
			{Day: day(-2), Activity: training.ActivityHall, Minutes: 45, Load: 36, Feel: "tired"},
			{Day: day(-1), Activity: training.ActivityHack, Minutes: 60, Load: 60, CanterShare: 0.25},
		},
	}
	for i := -2; i <= 4; i++ {
		in.Days = append(in.Days, Day{Date: day(i), Open: i >= 0})
	}
	return in
}

func acts(es []Entry) []training.Activity {
	var out []training.Activity
	for _, e := range es {
		out = append(out, e.Recommendation.Activity)
	}
	return out
}

func TestRulesPlansOpenDaysOnly(t *testing.T) {
	es := Rules(week())
	if len(es) != 5 {
		t.Fatalf("entries = %d, want 5 (Wed to Sun)", len(es))
	}
	sessions := 2
	for i, e := range es {
		if !e.Date.Equal(day(i)) || e.Source != SourceRules || e.Replaced != "" || e.Recommendation.Reason == "" {
			t.Errorf("entry %d = %+v", i, e)
		}
		if e.Recommendation.Activity != training.ActivityRest {
			sessions++
		}
	}
	// The rhythm allows at most 5 sessions a week (2 are done).
	if sessions > 5 {
		t.Errorf("planned %d sessions this week: %v", sessions, acts(es))
	}
	// Variety: the simulated history keeps the same activity from filling the week.
	seen := map[training.Activity]int{}
	for _, a := range acts(es) {
		seen[a]++
	}
	for a, n := range seen {
		if a != training.ActivityRest && n > 2 {
			t.Errorf("%s planned %d times: %v", a, n, acts(es))
		}
	}
}

func TestRulesRespectClosedDaysAndReha(t *testing.T) {
	in := week()
	in.Profile.Status = training.StatusReha
	in.Days[3].Open, in.Days[3].Planned = false, training.ActivityWalker // Thursday planned by someone
	in.Days[4].Reha = &recommend.RehaPhase{Name: "Phase 2", Activity: training.ActivityLunge, MinMinutes: 10, MaxMinutes: 15}
	es := Rules(in)
	if len(es) != 4 {
		t.Fatalf("entries = %d", len(es))
	}
	for _, e := range es {
		if e.Date.Equal(day(1)) {
			t.Error("closed Thursday was planned")
		}
		if e.Date.Equal(day(2)) && (e.Recommendation.Activity != training.ActivityLunge || e.Recommendation.Minutes != 15) {
			t.Errorf("reha Friday = %+v", e.Recommendation)
		}
		if a := e.Recommendation.Activity; a != training.ActivityRest && training.DefaultIntensity(a) != training.IntensityLight {
			t.Errorf("reha without phase allows only light work, got %s", a)
		}
	}
}

func TestMergeAcceptsAndReplaces(t *testing.T) {
	in := week()
	in.Profile.Allowed = []training.AllowedActivity{
		{Activity: training.ActivityHall, Mode: training.ModeOn},
		{Activity: training.ActivityLunge, Mode: training.ModeOn},
		{Activity: training.ActivityHack, Mode: training.ModeOn},
	}
	props := map[string]Proposal{
		"2026-09-30": {Activity: training.ActivityLunge, Minutes: 30, Reason: "Nach dem Ausritt gestern etwas Leichtes."},
		"2026-10-01": {Activity: training.ActivityJumping, Minutes: 40, Reason: "Springen"},
		"2026-10-02": {Activity: training.ActivityHall, Minutes: 200},
		"2026-10-03": {Activity: training.ActivityRest},
	}
	es := Merge(in, props)
	if len(es) != 5 {
		t.Fatalf("entries = %d", len(es))
	}
	wed, thu, fri, sat, sun := es[0], es[1], es[2], es[3], es[4]
	if wed.Source != SourceAI || wed.Recommendation.Activity != training.ActivityLunge || wed.Recommendation.Minutes != 30 ||
		wed.Recommendation.Reason != "Nach dem Ausritt gestern etwas Leichtes." {
		t.Errorf("wed = %+v", wed)
	}
	if thu.Source != SourceRules || thu.Recommendation.Activity == training.ActivityJumping || !strings.Contains(thu.Replaced, "Springen ausgeblendet") {
		t.Errorf("thu = %+v", thu)
	}
	if fri.Source != SourceAI || fri.Recommendation.Minutes != 60 || fri.Recommendation.Reason != "Vorschlag der KI." {
		t.Errorf("fri: minutes must be capped to the rhythm maximum: %+v", fri)
	}
	if sat.Source != SourceAI || sat.Recommendation.Activity != training.ActivityRest {
		t.Errorf("sat = %+v", sat)
	}
	if sun.Source != SourceRules || sun.Replaced != "" {
		t.Errorf("sun without proposal = %+v", sun)
	}
}

func TestMergeEnforcesWeekMaximum(t *testing.T) {
	in := week()
	in.Profile.Rhythm.SessionsMax = 3 // 2 done, so one more
	props := map[string]Proposal{}
	for i := 0; i <= 4; i++ {
		props[day(i).Format(dateLayout)] = Proposal{Activity: training.ActivityLunge, Minutes: 20}
	}
	es := Merge(in, props)
	if es[0].Recommendation.Activity != training.ActivityLunge || es[0].Source != SourceAI {
		t.Fatalf("wed = %+v", es[0])
	}
	for _, e := range es[1:] {
		if e.Recommendation.Activity != training.ActivityRest || !strings.Contains(e.Replaced, "3 Einheiten") {
			t.Errorf("%s = %+v", e.Date.Format(dateLayout), e)
		}
	}
}

func TestPromptHasNoIdentifyingData(t *testing.T) {
	in := week()
	in.Profile.Allowed[0] = training.AllowedActivity{Activity: training.ActivityHall, Mode: training.ModeConditional, Note: "nur mit Anna"}
	in.Weather = &recommend.Weather{Rain: true, TempC: 5.6}
	in.Ground = recommend.GroundWet
	in.Days[5].Reha = &recommend.RehaPhase{Name: "Sehne links, Dr. Muster", Activity: training.ActivityWalker, MinMinutes: 10, MaxMinutes: 20, Conditions: "nur ebener Boden"}
	p := Prompt(in)
	for _, leak := range []string{"Luna", "Anna", "Muster", "Sehne", "ebener", "2026", "L bei"} {
		if strings.Contains(p, leak) {
			t.Errorf("prompt contains %q: %s", leak, p)
		}
	}
	var msg struct {
		Discipline  string
		Activities  []map[string]any
		History     []map[string]any
		ShowsInDays []int `json:"shows_in_days"`
		Today       map[string]any
		Days        []map[string]any
	}
	if err := json.Unmarshal([]byte(p), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Discipline != "dressage" || len(msg.Activities) != 7 || msg.Activities[0]["conditional"] != true {
		t.Errorf("profile = %+v", msg)
	}
	if len(msg.History) != 2 || msg.History[0]["days_ago"] != float64(2) || msg.History[0]["feel"] != "tired" {
		t.Errorf("history = %v", msg.History)
	}
	if len(msg.ShowsInDays) != 1 || msg.ShowsInDays[0] != 10 {
		t.Errorf("shows = %v", msg.ShowsInDays)
	}
	if msg.Today["rain"] != true || msg.Today["night_min_c"] != float64(6) || msg.Today["ground"] != "wet" {
		t.Errorf("today = %v", msg.Today)
	}
	if len(msg.Days) != 7 || msg.Days[0]["day"] != "mo" || msg.Days[0]["done"] != true || msg.Days[2]["open"] != true || msg.Days[2]["in_days"] != float64(0) {
		t.Errorf("days = %v", msg.Days)
	}
	if r := msg.Days[5]["reha"].(map[string]any); r["activity"] != "walker" || r["max_minutes"] != float64(20) {
		t.Errorf("reha = %v", r)
	}
}

func TestParse(t *testing.T) {
	in := week()
	answer := "```json\n" + `{"days":[
		{"day":"mi","activity":"Hall","minutes":45.4,"reason":"Abwechslung\nnach dem Ausritt."},
		{"day":"mo","activity":"hack","minutes":30,"reason":"closed day"},
		{"day":"do","activity":"swimming","minutes":30},
		{"day":"fr","activity":"rest","minutes":0,"reason":"` + strings.Repeat("x", 300) + `"},
		{"day":"xx","activity":"hall"}
	]}` + "\n```"
	got, err := Parse(in, answer)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("proposals = %v", got)
	}
	if p := got["2026-09-30"]; p.Activity != training.ActivityHall || p.Minutes != 45 || p.Reason != "Abwechslung nach dem Ausritt." {
		t.Errorf("wed = %+v", p)
	}
	if p := got["2026-10-02"]; p.Activity != training.ActivityRest || len([]rune(p.Reason)) != MaxReasonRunes {
		t.Errorf("fri = %+v (%d runes)", p, len([]rune(p.Reason)))
	}
	for _, bad := range []string{"Gern! Hier ist der Plan.", `{"plan":[]}`, ""} {
		if _, err := Parse(in, bad); err == nil {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}
}

func TestSystemPromptNamesGermanWordsAndRanges(t *testing.T) {
	for _, want := range []string{
		"lunge = Longe", "15-30 minutes", "walker = Führanlage", "hack = Ausritt", "30-120 minutes",
		`no "du"`, `{"days":[`, "Skala der Ausbildung", "Losgelassenheit", `"exercise":"<key or null>"`,
	} {
		if !strings.Contains(SystemPrompt, want) {
			t.Errorf("system prompt lacks %q", want)
		}
	}
}

func library() []Exercise {
	return []Exercise{
		{ID: "id-ueb", Library: "dressage", Level: training.LevelBeginner, Title: "Übergänge", Tags: []string{"durchlaessigkeit"}, Mastered: true, NextID: "id-mt"},
		{ID: "id-kauen", Library: "dressage", Level: training.LevelBeginner, Title: "Zügel-aus-der-Hand-kauen-lassen", Tags: []string{"losgelassenheit"}, NextID: "id-mt"},
		{ID: "id-mt", Library: "dressage", Level: training.LevelIntermediate, Title: "Mitteltrab", Tags: []string{"schwung"}},
		{ID: "id-longe", Library: "lunge", Level: training.LevelBeginner, Title: "Übergänge an der Longe"},
	}
}

func TestPromptCarriesAgeLevelTomorrowAndExercises(t *testing.T) {
	in := week()
	in.AgeYears, in.Level = 12, training.LevelIntermediate
	in.Tomorrow = &recommend.Weather{Rain: true, TempC: 3.4}
	in.Exercises = library()
	in.Recent[0].ExerciseID, in.Recent[0].FocusRating = "id-kauen", 1
	p := Prompt(in)
	var msg struct {
		Level     string           `json:"level"`
		Age       int              `json:"horse_age_years"`
		Tomorrow  map[string]any   `json:"tomorrow"`
		Exercises []map[string]any `json:"exercises"`
		History   []map[string]any `json:"history"`
	}
	if err := json.Unmarshal([]byte(p), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Level != "intermediate" || msg.Age != 12 || msg.Tomorrow["rain"] != true || msg.Tomorrow["night_min_c"] != float64(3) {
		t.Errorf("msg = %+v", msg)
	}
	if len(msg.Exercises) != 4 || msg.Exercises[0]["key"] != "E1" || msg.Exercises[0]["next"] != "E3" || msg.Exercises[0]["mastered"] != true || msg.Exercises[1]["title"] != "Zügel-aus-der-Hand-kauen-lassen" {
		t.Errorf("exercises = %v", msg.Exercises)
	}
	if h := msg.History[0]; h["exercise"] != "E2" || h["rating"] != "hard" {
		t.Errorf("history = %v", h)
	}
	for _, leak := range []string{"id-", "Luna"} {
		if strings.Contains(p, leak) {
			t.Errorf("prompt contains %q", leak)
		}
	}
}

func TestParseAndMergeFocusAndExercise(t *testing.T) {
	in := week()
	in.Exercises = library()
	got, err := Parse(in, `{"days":[
		{"day":"mi","activity":"hall","minutes":45,"focus":"Dehnungshaltung\n im Trab","exercise":"e2","reason":"Lockern."},
		{"day":"do","activity":"hack","minutes":60,"focus":"Kondition","exercise":"E2","reason":"Gelände."},
		{"day":"fr","activity":"lunge","minutes":20,"focus":"Takt","exercise":"E99","reason":"Longe."},
		{"day":"sa","activity":"rest","minutes":0,"focus":"Pause","exercise":null,"reason":"Ruhe."}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if p := got["2026-09-30"]; p.Focus != "Dehnungshaltung im Trab" || p.Exercise == nil || p.Exercise.ID != "id-kauen" {
		t.Fatalf("wed = %+v", p)
	}
	if p := got["2026-10-02"]; p.Exercise != nil {
		t.Errorf("unknown key must give no exercise: %+v", p)
	}
	es := Merge(in, got)
	byDate := map[string]Entry{}
	for _, e := range es {
		byDate[e.Date.Format(dateLayout)] = e
	}
	if e := byDate["2026-09-30"]; e.Focus != "Dehnungshaltung im Trab" || e.Exercise == nil || e.Exercise.ID != "id-kauen" {
		t.Errorf("wed = %+v", e)
	}
	// A dressage exercise does not fit a hack: dropped, the focus stays.
	if e := byDate["2026-10-01"]; e.Exercise != nil || e.Focus != "Kondition" {
		t.Errorf("thu = %+v", e)
	}
	if e := byDate["2026-10-03"]; e.Focus != "" || e.Exercise != nil || e.Recommendation.Activity != training.ActivityRest {
		t.Errorf("sat = %+v", e)
	}
	// Days from the rules get the first exercise of their library the horse has not mastered.
	if e := byDate["2026-10-04"]; e.Source != SourceRules {
		t.Fatalf("sun = %+v", e)
	} else if lib := training.ExerciseLibrary(e.Recommendation.Activity, "dressage"); lib == "dressage" && (e.Exercise == nil || e.Exercise.ID != "id-kauen") {
		t.Errorf("sun rules exercise = %+v", e.Exercise)
	}
}

func TestTomorrowWeatherReachesTheRules(t *testing.T) {
	in := week()
	in.Days[3].Open = true // Thursday
	dry := Rules(in)
	in.Tomorrow = &recommend.Weather{Rain: true, TempC: 8}
	wet := Rules(in)
	if wet[1].Recommendation.Activity != training.ActivityHall {
		t.Errorf("rain tomorrow: thursday = %s (dry: %s)", wet[1].Recommendation.Activity, dry[1].Recommendation.Activity)
	}
}
