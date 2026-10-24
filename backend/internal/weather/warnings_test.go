package weather_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
	"github.com/Flusinerd/reiterhof-app/backend/internal/weather"
)

// The fixture has the shape of the GeoJSON answer of the DWD GeoServer layer
// dwd:Warnungen_Gemeinden: the storm warning twice (two municipality polygons), a frost
// warning without end and a test message.
const warningsFixture = "testdata/warnungen_gemeinden_sample.json"

func fixtureWarnings(t testing.TB) []byte {
	t.Helper()
	b, err := os.ReadFile(warningsFixture)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseWarnings(t *testing.T) {
	ws, err := weather.ParseWarnings(fixtureWarnings(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 2 {
		t.Fatalf("got %d warnings, want 2 (duplicate polygon and test message dropped): %+v", len(ws), ws)
	}
	storm, frost := ws[0], ws[1]
	if storm.Event != "STURMBÖEN" || storm.Severity != weather.SeverityModerate || storm.Headline != "Amtliche WARNUNG vor STURMBÖEN" ||
		!strings.HasPrefix(storm.Description, "Es treten Sturmböen") || !strings.HasPrefix(storm.Instruction, "ACHTUNG!") {
		t.Errorf("storm %+v", storm)
	}
	if !storm.Onset.Equal(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)) || storm.Expires == nil ||
		!storm.Expires.Equal(time.Date(2026, 10, 24, 20, 0, 0, 0, time.UTC)) {
		t.Errorf("storm times %v %v", storm.Onset, storm.Expires)
	}
	if frost.Event != "FROST" || frost.Severity != weather.SeverityMinor || frost.Expires != nil || frost.Instruction != "" {
		t.Errorf("frost %+v", frost)
	}

	if ws, err := weather.ParseWarnings([]byte(`{"type":"FeatureCollection","features":[]}`)); err != nil || len(ws) != 0 {
		t.Errorf("empty collection: %v %v", ws, err)
	}
	for name, body := range map[string]string{
		"xml exception": `<?xml version="1.0"?><ows:ExceptionReport/>`,
		"other json":    `{"error":"x"}`,
	} {
		if _, err := weather.ParseWarnings([]byte(body)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestWarningNotify(t *testing.T) {
	cases := []struct {
		event, severity string
		want            bool
	}{
		{"STURMBÖEN", weather.SeverityModerate, true},
		{"STRENGER FROST", weather.SeverityModerate, true},
		{"STARKE HITZE", weather.SeverityModerate, true},
		{"GEWITTER", weather.SeverityModerate, false},
		{"FROST", weather.SeverityMinor, false},
		{"UNWETTER mit HEFTIGEM STARKREGEN", weather.SeveritySevere, true},
		{"ORKAN", weather.SeverityExtreme, true},
	}
	for _, c := range cases {
		if got := (weather.Warning{Event: c.event, Severity: c.severity}).Notify(); got != c.want {
			t.Errorf("%s/%s: notify %v, want %v", c.event, c.severity, got, c.want)
		}
	}
}

func TestWarningSpan(t *testing.T) {
	loc := berlin(t)
	on := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	same := time.Date(2026, 10, 24, 20, 0, 0, 0, time.UTC)
	next := time.Date(2026, 10, 25, 7, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		exp  *time.Time
		want string
	}{
		{&same, "Sa 24.10., 14:00 bis 22:00 Uhr"},
		{&next, "Sa 24.10., 14:00 bis So 25.10., 08:00 Uhr"},
		{nil, "ab Sa 24.10., 14:00 Uhr"},
	} {
		if got := weather.WarningSpan(on, c.exp, loc); got != c.want {
			t.Errorf("span %q, want %q", got, c.want)
		}
	}
	title, body := weather.WarningPush(weather.StoredWarning{Severity: weather.SeveritySevere, Headline: "Amtliche UNWETTERWARNUNG vor ORKANBÖEN",
		Onset: on, Expires: &same}, loc)
	if title != "Unwetterwarnung" || body != "Amtliche UNWETTERWARNUNG vor ORKANBÖEN: Sa 24.10., 14:00 bis 22:00 Uhr." {
		t.Errorf("push %q %q", title, body)
	}
}

func TestDWDWarningsFetch(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/geoserver/dwd/ows" {
			http.NotFound(w, r)
			return
		}
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixtureWarnings(t))
	}))
	defer srv.Close()
	d := &weather.DWDWarnings{BaseURL: srv.URL}
	ws, err := d.FetchWarnings(context.Background(), 51.4, 6.97)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 2 {
		t.Errorf("got %d warnings", len(ws))
	}
	for _, part := range []string{"typeName=dwd%3AWarnungen_Gemeinden", "outputFormat=application%2Fjson",
		"bbox=51.39950%2C6.96950%2C51.40050%2C6.97050%2Curn%3Aogc%3Adef%3Acrs%3AEPSG%3A%3A4326"} {
		if !strings.Contains(query, part) {
			t.Errorf("query %q lacks %q", query, part)
		}
	}
	bad := &weather.DWDWarnings{BaseURL: srv.URL + "/missing"}
	if _, err := bad.FetchWarnings(context.Background(), 51.4, 6.97); err == nil {
		t.Error("expected an error for HTTP 404")
	}
}

type fakeWarnings struct {
	ws  []weather.Warning
	err error
}

func (f *fakeWarnings) FetchWarnings(context.Context, float64, float64) ([]weather.Warning, error) {
	return f.ws, f.err
}

type sent struct {
	to          []string
	kind, title string
	body        string
}

type recNotifier struct {
	sent []sent
	err  error
}

func (n *recNotifier) NotifyUsers(_ context.Context, _ string, userIDs []string, kind, title, body string, _ map[string]any) error {
	if n.err != nil {
		return n.err
	}
	n.sent = append(n.sent, sent{to: userIDs, kind: kind, title: title, body: body})
	return nil
}

func TestWarningServiceStoresAndPushesOnce(t *testing.T) {
	berlin(t)
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()
	var members int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE stable_id = $1 AND deleted_at IS NULL`, seed.StableB).Scan(&members); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	exp := now.Add(8 * time.Hour)
	past := now.Add(-time.Hour)
	storm := weather.Warning{DWDID: "storm-1", Event: "STURMBÖEN", Severity: weather.SeverityModerate,
		Headline: "Amtliche WARNUNG vor STURMBÖEN", Onset: now.Add(2 * time.Hour), Expires: &exp}
	rain := weather.Warning{DWDID: "rain-1", Event: "DAUERREGEN", Severity: weather.SeverityMinor,
		Headline: "Amtliche WARNUNG vor DAUERREGEN", Onset: now}
	gone := weather.Warning{DWDID: "old-1", Event: "NEBEL", Severity: weather.SeverityMinor,
		Headline: "Amtliche WARNUNG vor NEBEL", Onset: now.Add(-5 * time.Hour), Expires: &past}
	fetch := &fakeWarnings{ws: []weather.Warning{storm, rain, gone}}
	notifier := &recNotifier{}
	svc := &weather.WarningService{Pool: pool, Fetcher: fetch, Notify: notifier, Now: func() time.Time { return now }}

	if err := svc.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if len(notifier.sent) != 1 || notifier.sent[0].kind != push.KindWeatherWarning || len(notifier.sent[0].to) != members ||
		notifier.sent[0].title != "Wetterwarnung" || !strings.Contains(notifier.sent[0].body, "14:00 bis 20:00 Uhr") {
		t.Fatalf("pushes %+v (members %d)", notifier.sent, members)
	}
	store := &weather.Store{Pool: pool}
	active, err := store.ActiveWarnings(ctx, seed.StableB, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 2 || active[0].Event != "STURMBÖEN" || active[1].Event != "DAUERREGEN" {
		t.Fatalf("active %+v (expired dropped, most severe first)", active)
	}

	// Same answer again: nothing new.
	if err := svc.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("second run pushed again: %+v", notifier.sent)
	}

	// The DWD updates the storm warning (new identifier, same level): no new push.
	// Then it escalates to severe: push.
	updated := storm
	updated.DWDID = "storm-2"
	fetch.ws = []weather.Warning{updated}
	if err := svc.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("update pushed again: %+v", notifier.sent)
	}
	if active, _ := store.ActiveWarnings(ctx, seed.StableB, now); len(active) != 1 || active[0].Headline != storm.Headline {
		t.Fatalf("after update %+v", active)
	}
	severe := storm
	severe.DWDID, severe.Severity, severe.Headline = "storm-3", weather.SeveritySevere, "Amtliche UNWETTERWARNUNG vor ORKANARTIGEN BÖEN"
	fetch.ws = []weather.Warning{severe}
	if err := svc.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if len(notifier.sent) != 2 || notifier.sent[1].title != "Unwetterwarnung" {
		t.Fatalf("escalation %+v", notifier.sent)
	}

	// A failed push releases the claims and is retried on the next run.
	extreme := severe
	extreme.DWDID, extreme.Event, extreme.Severity = "heat-1", "EXTREME HITZE", weather.SeverityExtreme
	fetch.ws = []weather.Warning{severe, extreme}
	notifier.err = errors.New("push down")
	if err := svc.Refresh(ctx); err == nil {
		t.Fatal("expected the push error")
	}
	var claims int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM reminders r JOIN weather_warnings w ON w.id = r.source_id
		WHERE r.source_table = 'weather_warnings' AND w.dwd_id = 'heat-1'`).Scan(&claims); err != nil || claims != 0 {
		t.Fatalf("claims after failure %d %v", claims, err)
	}
	notifier.err = nil
	if err := svc.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if len(notifier.sent) != 3 || len(notifier.sent[2].to) != members {
		t.Fatalf("retry %+v", notifier.sent)
	}

	// Fetch errors are reported; no warnings means the rows are cleared.
	fetch.err = errors.New("dwd down")
	if err := svc.Refresh(ctx); err == nil {
		t.Error("expected the fetch error")
	}
	fetch.err, fetch.ws = nil, nil
	if err := svc.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if active, _ := store.ActiveWarnings(ctx, seed.StableB, now); len(active) != 0 {
		t.Errorf("warnings not cleared: %+v", active)
	}
}

func TestWarningsEndpoint(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	exp := now.Add(time.Hour)
	if err := (&weather.Store{Pool: pool}).ReplaceWarnings(ctx, seed.StableB, []weather.Warning{
		{DWDID: "a", Event: "FROST", Severity: weather.SeverityMinor, Headline: "Amtliche WARNUNG vor FROST", Onset: now},
		{DWDID: "b", Event: "ORKANBÖEN", Severity: weather.SeverityExtreme, Headline: "Amtliche WARNUNG vor ORKANBÖEN",
			Description: "Orkan.", Onset: now, Expires: &exp},
	}, now); err != nil {
		t.Fatal(err)
	}
	h := httpapi.NewHandler(httpapi.Deps{Pool: pool, Now: func() time.Time { return now }})
	call := func(user string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/weather/warnings", nil)
		if user != "" {
			req.Header.Set("Authorization", "Bearer "+authtest.TokenAt(t, pool, user, now))
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if rec := call(""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}
	rec := call(seed.UserLea)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var resp struct {
		Warnings []struct {
			ID, Event, Severity, Headline, Description string
			Expires                                    *time.Time
		} `json:"warnings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Warnings) != 2 || resp.Warnings[0].Severity != "extreme" || resp.Warnings[0].Description != "Orkan." ||
		resp.Warnings[1].Event != "FROST" || resp.Warnings[1].Expires != nil || resp.Warnings[0].ID == "" {
		t.Errorf("warnings %+v", resp.Warnings)
	}
}
