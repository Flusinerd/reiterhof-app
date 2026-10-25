package privacy_test

import (
	"encoding/json"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// The own focuses a person picked for riding a horse (rider_focuses) are personal data: the export
// lists the own ones only, and deleting the account removes them.
func TestRiderFocusesAreExportedAndDeleted(t *testing.T) {
	e := setup(t)
	// The demo data has a focus of Mia already: the statement replaces it.
	e.exec(`INSERT INTO rider_focuses (stable_id, horse_id, user_id, keys) VALUES ($1, $2, $3, '{gelassenheit}'), ($1, $2, $4, '{takt,anlehnung}')
	        ON CONFLICT (horse_id, user_id) DO UPDATE SET keys = EXCLUDED.keys`,
		seed.StableB, seed.HorseLuna, seed.UserMia, seed.UserJan)

	rec := e.do(seed.UserMia, "GET", "/api/v1/me/export", nil)
	if rec.Code != 200 {
		t.Fatalf("export = %d %s", rec.Code, rec.Body)
	}
	doc := decode[map[string]json.RawMessage](t, rec)
	var focuses []struct {
		HorseName string   `json:"horse_name"`
		Keys      []string `json:"keys"`
	}
	if err := json.Unmarshal(doc["rider_focuses"], &focuses); err != nil {
		t.Fatalf("rider_focuses = %s (%v)", doc["rider_focuses"], err)
	}
	if len(focuses) != 1 || focuses[0].HorseName != "Luna" || len(focuses[0].Keys) != 1 || focuses[0].Keys[0] != "gelassenheit" {
		t.Fatalf("exported focuses = %+v, want only Mia's", focuses)
	}

	if rec := e.do(seed.UserMia, "POST", "/api/v1/me/delete", map[string]any{"confirm": true}); rec.Code != 204 {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body)
	}
	if n := e.count(`SELECT count(*) FROM rider_focuses WHERE user_id = $1`, seed.UserMia); n != 0 {
		t.Errorf("the deleted user keeps %d focuses", n)
	}
	if n := e.count(`SELECT count(*) FROM rider_focuses WHERE user_id = $1`, seed.UserJan); n != 1 {
		t.Errorf("the focuses of another user: %d, want 1", n)
	}
}
