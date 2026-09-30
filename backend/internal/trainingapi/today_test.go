package trainingapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
	"github.com/Flusinerd/reiterhof-app/backend/internal/stables"
	"github.com/Flusinerd/reiterhof-app/backend/internal/weather"
)

func (e *env) rain(willRain bool, tempC float64) {
	e.t.Helper()
	err := (&weather.Store{Pool: e.pool}).Save(context.Background(), seed.StableB, "10410", e.now, weather.Summary{
		Day: "2026-03-25", NightMinC: tempC, RainProbability: 80, RainMM: 2, WillRain: willRain,
	})
	if err != nil {
		e.t.Fatal(err)
	}
}

func recs(out map[string]any) []map[string]any {
	var r []map[string]any
	for _, x := range list(out["recommendations"]) {
		r = append(r, obj(x))
	}
	return r
}

func TestTodayRainRecommendsHall(t *testing.T) {
	e := newEnv(t)
	e.rain(true, 6)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/training-profile", validProfile, http.StatusOK) // show on 28th, jumping off

	got := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/today?minutes=45", "", http.StatusOK)
	r := recs(got)
	if len(r) != 2 { // the profile allows only hall and (conditionally) hack
		t.Fatalf("recommendations = %d, want 2: %v", len(r), got["recommendations"])
	}
	if r[0]["activity"] != "hall" || num(r[0]["minutes"]) != 45 || !strings.Contains(str(r[0]["reason"]), "Regen") {
		t.Fatalf("first recommendation = %v, want hall/45 min with a rain reason", r[0])
	}
	if got["context"] != "Regen, 6 °C · Turnier in 3 Tagen" {
		t.Fatalf("context = %q", got["context"])
	}
	// profile: only hall and conditional hack are on, everything else is hidden with a reason
	hidden := map[string]string{}
	for _, h := range list(got["hidden"]) {
		hidden[str(obj(h)["activity"])] = str(obj(h)["reason"])
	}
	if hidden["jumping"] != "Springen ausgeblendet: laut Profil nicht für Luna" {
		t.Fatalf("hidden jumping = %q", hidden["jumping"])
	}
	if _, ok := hidden["hall"]; ok {
		t.Fatal("hall must not be hidden")
	}
	// the seven days end today, nothing trained yet
	dots := list(got["week"])
	if len(dots) != 7 || obj(dots[6])["is_today"] != true || obj(dots[6])["date"] != "2026-03-25" || obj(dots[0])["date"] != "2026-03-19" {
		t.Fatalf("dots = %v", dots)
	}
	// the hall exercise comes from the library (easiest not yet mastered dressage exercise)
	if ex := obj(r[0]["exercise"]); ex["title"] != "Übergänge" || len(list(ex["steps"])) == 0 {
		t.Fatalf("exercise = %v", r[0]["exercise"])
	}
	if got["can_log"] != true || got["has_profile"] != true {
		t.Fatalf("flags = %v", got)
	}

	// dry weather and wet ground push the arena down as well; no rain, no weather line
	e.exec(`DELETE FROM weather_snapshots`)
	if err := stables.SetGroundCondition(context.Background(), e.pool, seed.StableB, "muddy", e.now); err != nil {
		t.Fatal(err)
	}
	got = e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/today", "", http.StatusOK)
	if got["context"] != "Boden matschig · Turnier in 3 Tagen" {
		t.Fatalf("context = %q", got["context"])
	}
	if recs(got)[0]["activity"] == "arena" {
		t.Fatalf("muddy ground: arena recommended first: %v", got["recommendations"])
	}
}

func TestTodayRestDayAfterShow(t *testing.T) {
	e := newEnv(t)
	body := strings.Replace(validProfile, "2026-03-28", "2026-03-24", 1) // show yesterday
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/training-profile", body, http.StatusOK)
	got := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/today", "", http.StatusOK)
	r := recs(got)
	if len(r) != 1 || r[0]["activity"] != "rest" || !strings.Contains(str(r[0]["reason"]), "Ruhetag") {
		t.Fatalf("recommendations = %v", got["recommendations"])
	}
	dots := list(got["week"])
	if obj(dots[5])["kind"] != "rest" { // the show was on the 24th, the 25th is today: 24th shows nothing, 25th rest
		if obj(dots[6])["kind"] != "rest" {
			t.Fatalf("dots = %v", dots)
		}
	}
}

func TestTodayRiderRules(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/today?minutes=45"

	got := e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK)
	for _, r := range recs(got) {
		if r["activity"] == "hack" || r["activity"] == "jumping" {
			t.Fatalf("rider got %v", r)
		}
	}
	hidden := map[string]string{}
	for _, h := range list(got["hidden"]) {
		hidden[str(obj(h)["activity"])] = str(obj(h)["reason"])
	}
	if !strings.Contains(hidden["hack"], "Mia") || hidden["jumping"] == "" {
		t.Fatalf("hidden = %v", hidden)
	}
	if got["can_edit"] != false {
		t.Fatalf("can_edit = %v", got["can_edit"])
	}

	// a rider without rules in the profile sees nothing (fail closed)
	e.exec(`UPDATE training_profiles SET rb_rules = '{}' WHERE horse_id = $1`, luna)
	got = e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK)
	r := recs(got)
	if len(r) != 1 || r[0]["activity"] != "rest" || len(list(got["hidden"])) != 7 {
		t.Fatalf("rider without rules: %v hidden=%d", got["recommendations"], len(list(got["hidden"])))
	}
	// the owner is not affected
	if recs(e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK))[0]["activity"] == "rest" {
		t.Fatal("owner must still get a training recommendation")
	}

	// hack_alone in the rider's rule list lifts the "no hacking alone" restriction
	e.exec(`UPDATE training_profiles SET rb_rules = jsonb_build_object($2::text, '{"allowed_activities":["hack"],"max_intensity":"any"}'::jsonb) WHERE horse_id = $1`, luna, seed.UserMia)
	got = e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK)
	if recs(got)[0]["activity"] == "hack" {
		t.Fatalf("hack without hack_alone: %v", got["recommendations"])
	}
	e.exec(`UPDATE horse_riders SET rules = '["ride","hack_alone"]' WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia)
	got = e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK)
	if recs(got)[0]["activity"] != "hack" {
		t.Fatalf("hack with hack_alone: %v", got["recommendations"])
	}

	// members without a role and invalid input
	e.errCode(seed.UserSarah, http.MethodGet, path, "", http.StatusForbidden)
	e.errCode(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/today?minutes=abc", "", http.StatusBadRequest)
}

func TestTodayRehaPhase(t *testing.T) {
	e := newEnv(t)
	// The seed plan starts 10 days before the real day of seeding; move it relative to the fixed clock.
	e.exec(`UPDATE reha_plans SET start_date = '2026-03-15' WHERE horse_id = $1`, fanta)

	got := e.call(seed.UserAnna, http.MethodGet, "/api/v1/horses/"+fanta+"/today?minutes=45", "", http.StatusOK)
	r := recs(got)
	if len(r) != 1 || r[0]["activity"] != "walker" || !strings.Contains(str(r[0]["reason"]), "Schrittführen") || num(r[0]["minutes"]) != 19 {
		t.Fatalf("reha recommendation = %v", got["recommendations"])
	}
	reha := obj(got["reha"])
	if reha["phase"] != "Schrittführen" || num(reha["phase_index"]) != 1 || num(reha["phases"]) != 2 {
		t.Fatalf("reha = %v", reha)
	}
	if !strings.Contains(str(got["context"]), "Reha: Schrittführen") {
		t.Fatalf("context = %q", got["context"])
	}

	// day 15 of the plan: second phase
	e.exec(`UPDATE reha_plans SET start_date = '2026-03-10' WHERE horse_id = $1`, fanta)
	got = e.call(seed.UserAnna, http.MethodGet, "/api/v1/horses/"+fanta+"/today", "", http.StatusOK)
	if recs(got)[0]["activity"] != "lunge" || obj(got["reha"])["phase"] != "Longieren im Schritt" {
		t.Fatalf("phase 2 = %v", got["recommendations"])
	}

	// after the last phase (and without an active plan) the reha status is plain light-only
	e.exec(`UPDATE reha_plans SET start_date = '2026-02-01' WHERE horse_id = $1`, fanta)
	got = e.call(seed.UserAnna, http.MethodGet, "/api/v1/horses/"+fanta+"/today", "", http.StatusOK)
	if got["reha"] != nil {
		t.Fatalf("finished plan must give no phase: %v", got["reha"])
	}
	for _, x := range recs(got) {
		if x["intensity"] != "light" && x["activity"] != "rest" {
			t.Fatalf("status reha without phase must be light only: %v", x)
		}
	}
	// the rider Lea only sees what her rules allow
	for _, x := range recs(e.call(seed.UserLea, http.MethodGet, "/api/v1/horses/"+fanta+"/today", "", http.StatusOK)) {
		if x["activity"] == "lunge" {
			t.Fatalf("Lea may not lunge: %v", x)
		}
	}
}
