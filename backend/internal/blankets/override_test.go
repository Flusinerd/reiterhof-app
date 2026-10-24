package blankets_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

type overrideJSON struct {
	BlanketID *string      `json:"blanket_id"`
	Blanket   *blanketJSON `json:"blanket"`
	DayFrom   string       `json:"day_from"`
	DayUntil  *string      `json:"day_until"`
	Note      string       `json:"note"`
	SetBy     *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"set_by"`
}

type overridePlanJSON struct {
	Day                string        `json:"day"`
	Recommendation     recJSON       `json:"recommendation"`
	RuleRecommendation recJSON       `json:"rule_recommendation"`
	Override           *overrideJSON `json:"override"`
}

func TestBlanketOverride(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + seed.HorseLuna + "/blanket-override"
	plan := func() overridePlanJSON {
		t.Helper()
		var p overridePlanJSON
		e.do(seed.UserMia, "GET", "/api/v1/horses/"+seed.HorseLuna+"/blanket-plan", nil).status(t, 200).into(t, &p)
		return p
	}
	// 3 °C with rain: the rules say "Decke 100 g" (rule 4).
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 6, 0), 3, true)
	if p := plan(); p.Override != nil || p.Recommendation.Manual || p.Recommendation.Blanket.ID != seed.BlanketLuna100 ||
		!reflect.DeepEqual(p.Recommendation, p.RuleRecommendation) {
		t.Fatalf("plan without override = %+v", p)
	}

	// Riders and other members may not choose; validation.
	e.do(seed.UserMia, "PUT", path, m{"blanket_id": seed.BlanketLuna200}).errCode(t, 403, "forbidden")
	e.do(seed.UserJan, "PUT", path, m{"blanket_id": seed.BlanketBalu150}).errCode(t, 400, "validation_failed")
	e.do(seed.UserJan, "PUT", path, m{"blanket_id": "nope"}).errCode(t, 400, "validation_failed")
	e.do(seed.UserJan, "PUT", path, m{"blanket_id": seed.BlanketLuna200, "nights": 0}).errCode(t, 400, "validation_failed")
	e.do(seed.UserJan, "PUT", path, m{"blanket_id": seed.BlanketLuna200, "nights": 31}).errCode(t, 400, "validation_failed")
	e.do(seed.UserMia, "PUT", "/api/v1/horses/"+seed.HorseBalu+"/blanket-override", m{"blanket_id": nil}).errCode(t, 403, "forbidden")

	// The owner picks the 200 g blanket for two nights.
	var put struct {
		Override       overrideJSON `json:"override"`
		Recommendation recJSON      `json:"recommendation"`
	}
	e.do(seed.UserJan, "PUT", path, m{"blanket_id": seed.BlanketLuna200, "nights": 2, "note": "Schur"}).status(t, 200).into(t, &put)
	if o := put.Override; o.Blanket == nil || o.Blanket.ID != seed.BlanketLuna200 || o.DayFrom != "2026-09-30" ||
		o.DayUntil == nil || *o.DayUntil != "2026-10-01" || o.Note != "Schur" || o.SetBy == nil || o.SetBy.ID != seed.UserJan {
		t.Fatalf("override = %+v", o)
	}
	p := plan()
	r := p.Recommendation
	if r.Status != "blanket" || !r.Manual || r.RuleIndex != nil || r.Blanket == nil || r.Blanket.ID != seed.BlanketLuna200 || r.Note != "Schur" {
		t.Errorf("effective recommendation = %+v", r)
	}
	if rr := p.RuleRecommendation; rr.Manual || rr.Blanket == nil || rr.Blanket.ID != seed.BlanketLuna100 || rr.RuleIndex == nil || *rr.RuleIndex != 3 {
		t.Errorf("rule recommendation = %+v", rr)
	}
	for _, h := range e.today(seed.UserMia).Horses {
		if h.Horse.ID == seed.HorseLuna && (h.Recommendation.Blanket == nil || h.Recommendation.Blanket.ID != seed.BlanketLuna200) {
			t.Errorf("today recommendation = %+v", h.Recommendation)
		}
	}

	// "Eingedeckt" without a blanket stores the manual choice; any blanket of the horse may be named.
	var st struct{ State stateJSON }
	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blanket-state", m{"action": "covered"}).status(t, 200).into(t, &st)
	if st.State.CoveredWith == nil || *st.State.CoveredWith != seed.BlanketLuna200 {
		t.Errorf("covered with = %v", st.State.CoveredWith)
	}
	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blanket-state",
		m{"action": "covered", "covered_with": seed.BlanketLunaRain}).status(t, 200).into(t, &st)
	if st.State.CoveredWith == nil || *st.State.CoveredWith != seed.BlanketLunaRain {
		t.Errorf("covered with = %v", st.State.CoveredWith)
	}

	// The second night still has it, the third is back to the rules.
	e.at(berlinAt(2026, 10, 1, 18, 0))
	e.snapshot("2026-10-01", berlinAt(2026, 10, 1, 6, 0), 3, true)
	if p := plan(); p.Override == nil || p.Recommendation.Blanket.ID != seed.BlanketLuna200 {
		t.Errorf("second night = %+v", p)
	}
	e.at(berlinAt(2026, 10, 2, 18, 0))
	e.snapshot("2026-10-02", berlinAt(2026, 10, 2, 6, 0), 3, true)
	if p := plan(); p.Override != nil || p.Recommendation.Manual || p.Recommendation.Blanket.ID != seed.BlanketLuna100 {
		t.Errorf("third night = %+v", p)
	}

	// "No blanket" until reset replaces the earlier choice and lasts.
	e.do(seed.UserJan, "PUT", path, m{"blanket_id": nil}).status(t, 200)
	e.at(berlinAt(2026, 10, 20, 18, 0))
	e.snapshot("2026-10-20", berlinAt(2026, 10, 20, 6, 0), 3, true)
	p = plan()
	if p.Override == nil || p.Override.DayUntil != nil || p.Recommendation.Status != "none" || !p.Recommendation.Manual ||
		p.Recommendation.Blanket != nil || p.RuleRecommendation.Status != "blanket" {
		t.Errorf("until reset = %+v", p)
	}

	// Reset: back to the rules; deleting again is fine.
	e.do(seed.UserMia, "DELETE", path, nil).errCode(t, 403, "forbidden")
	e.do(seed.UserJan, "DELETE", path, nil).status(t, 204)
	e.do(seed.UserJan, "DELETE", path, nil).status(t, 204)
	if p := plan(); p.Override != nil || p.Recommendation.Manual {
		t.Errorf("after reset = %+v", p)
	}
}

func TestBlanketOverrideDroppedWithItsBlanket(t *testing.T) {
	e := newEnv(t)
	var b blanketJSON
	e.do(seed.UserJan, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blankets", m{"name": "Fleece"}).status(t, 201).into(t, &b)
	e.do(seed.UserJan, "PUT", "/api/v1/horses/"+seed.HorseLuna+"/blanket-override", m{"blanket_id": b.ID}).status(t, 200)
	e.do(seed.UserJan, "DELETE", "/api/v1/horses/"+seed.HorseLuna+"/blankets/"+b.ID, nil).status(t, 204)
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM blanket_overrides`).Scan(&n); err != nil || n != 0 {
		t.Errorf("overrides left = %d (%v)", n, err)
	}
}

func TestWeatherChangeWithOverride(t *testing.T) {
	e := newEnv(t)
	check := func() {
		t.Helper()
		if err := e.svc.CheckWeatherChange(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	// 8 °C dry: the rules say no blanket; Jan picks the 200 g blanket anyway and Mia covers.
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 18, 0), 8, false)
	e.do(seed.UserJan, "PUT", "/api/v1/horses/"+seed.HorseLuna+"/blanket-override", m{"blanket_id": seed.BlanketLuna200}).status(t, 200)
	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blanket-state", m{"action": "covered"}).status(t, 200)

	// -1 °C: the rules now agree with the manual choice, nothing to say.
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 19, 30), -1, false)
	e.at(berlinAt(2026, 9, 30, 19, 35))
	check()
	if len(e.fake.Sent()) != 0 {
		t.Fatalf("pushed although the rules agree: %+v", e.fake.Sent())
	}
	// 3 °C rain: the rules want 100 g, the manual 200 g stays; the push says so.
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 20, 0), 3, true)
	e.at(berlinAt(2026, 9, 30, 20, 5))
	check()
	msgs := e.fake.Sent()
	if len(msgs) != 2 || !strings.Contains(msgs[0].Body, "Regel jetzt Decke 100 g statt keine Decke") ||
		!strings.Contains(msgs[0].Body, "Von Hand gewählt bleibt Decke 200 g") {
		t.Errorf("pushes = %+v", msgs)
	}
}
