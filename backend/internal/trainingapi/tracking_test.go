package trainingapi_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const windowJSON = `{"t":0,"f":[1.5,0.2,0.1,0.05,0.1,0.8],"p":"walk","a":"walk","v":5.1,"c":"trot"}`

func TestTrackedSessionStoresTrackAndWindows(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/sessions"
	body := `{
	  "activity": "hack", "minutes": 62, "started_at": "2026-03-25T08:30:00+01:00", "distance_m": 7400,
	  "gait_shares": {"halt": 0.05, "walk": 0.6, "trot": 0.3, "canter": 0.05},
	  "track": [
	    {"lat": 52.5, "lon": 9.7, "t": 0, "alt": 51.5, "g": "walk"},
	    {"lat": 52.5005, "lon": 9.7008, "t": 60, "g": "trot"}
	  ],
	  "gait_windows": [` + windowJSON + `, ` + windowJSON + `]
	}`
	got := e.call(seed.UserJan, http.MethodPost, path, body, http.StatusCreated)
	s := obj(got["session"])
	if s["source"] != "tracked" || num(s["distance_m"]) != 7400 {
		t.Fatalf("session = %v", s)
	}
	id := str(s["id"])
	var points, windows int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT jsonb_array_length(track) FROM sessions WHERE id = $1`, id).Scan(&points); err != nil || points != 2 {
		t.Fatalf("stored track points = %d, err %v", points, err)
	}
	var count int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT window_count, jsonb_array_length(windows) FROM gait_windows WHERE session_id = $1`, id).Scan(&count, &windows); err != nil || count != 2 || windows != 2 {
		t.Fatalf("stored windows = %d/%d, err %v", count, windows, err)
	}

	// A session without track or windows stores neither.
	got = e.call(seed.UserJan, http.MethodPost, path, `{"activity":"hall","minutes":30,"gait_shares":{"walk":1}}`, http.StatusCreated)
	id = str(obj(got["session"])["id"])
	var nullTrack bool
	if err := e.pool.QueryRow(context.Background(), `SELECT track IS NULL FROM sessions WHERE id = $1`, id).Scan(&nullTrack); err != nil || !nullTrack {
		t.Fatalf("track should be NULL: %v %v", nullTrack, err)
	}
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM gait_windows WHERE session_id = $1`, id).Scan(&count); err != nil || count != 0 {
		t.Fatalf("windows stored without input: %d %v", count, err)
	}

	// Only a track marks the session as tracked.
	got = e.call(seed.UserJan, http.MethodPost, path, `{"activity":"hack","minutes":30,"track":[{"lat":1,"lon":2,"t":0}]}`, http.StatusCreated)
	if obj(got["session"])["source"] != "tracked" {
		t.Fatalf("source = %v", obj(got["session"])["source"])
	}
}

func TestTrackAndWindowValidation(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/sessions"
	point := func(n int) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = fmt.Sprintf(`{"lat":52,"lon":9,"t":%d}`, i)
		}
		return `[` + strings.Join(parts, ",") + `]`
	}
	windows := func(n int) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = windowJSON
		}
		return `[` + strings.Join(parts, ",") + `]`
	}
	base := `{"activity":"hack","minutes":30,`
	for name, body := range map[string]string{
		"lat out of range":   base + `"track":[{"lat":91,"lon":9,"t":0}]}`,
		"lon out of range":   base + `"track":[{"lat":52,"lon":181,"t":0}]}`,
		"negative time":      base + `"track":[{"lat":52,"lon":9,"t":-1}]}`,
		"descending time":    base + `"track":[{"lat":52,"lon":9,"t":10},{"lat":52,"lon":9,"t":5}]}`,
		"altitude":           base + `"track":[{"lat":52,"lon":9,"t":0,"alt":20000}]}`,
		"track gait":         base + `"track":[{"lat":52,"lon":9,"t":0,"g":"gallop"}]}`,
		"track unknown key":  base + `"track":[{"lat":52,"lon":9,"t":0,"x":1}]}`,
		"too many points":    base + `"track":` + point(5001) + `}`,
		"window features":    base + `"gait_windows":[{"t":0,"f":[1,2],"p":"walk","a":"walk"}]}`,
		"window gait":        base + `"gait_windows":[{"t":0,"f":[1,2,3,4,5,6],"p":"run","a":"walk"}]}`,
		"window label":       base + `"gait_windows":[{"t":0,"f":[1,2,3,4,5,6],"p":"walk","a":"walk","c":"x"}]}`,
		"window speed":       base + `"gait_windows":[{"t":0,"f":[1,2,3,4,5,6],"p":"walk","a":"walk","v":-1}]}`,
		"too many windows":   base + `"gait_windows":` + windows(3001) + `}`,
		"distance too large": base + `"distance_m":600000}`,
	} {
		if code := e.errCode(seed.UserJan, http.MethodPost, path, body, http.StatusBadRequest); code != "validation_failed" && code != "invalid_json" {
			t.Errorf("%s: code = %q", name, code)
		}
	}
	// The limits themselves are accepted.
	e.call(seed.UserJan, http.MethodPost, path, base+`"track":`+point(5000)+`}`, http.StatusCreated)
	e.call(seed.UserJan, http.MethodPost, path, base+`"gait_windows":`+windows(3000)+`}`, http.StatusCreated)

	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM sessions`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("stored %d sessions, want only the two valid ones (err %v)", n, err)
	}
}
