package weather_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
	"github.com/Flusinerd/reiterhof-app/backend/internal/weather"
)

const fixture = "testdata/MOSMIX_L_sample_10410.kml"

func berlin(t testing.TB) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	return loc
}

func fixtureKML(t testing.TB) []byte {
	t.Helper()
	b, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func fixtureKMZ(t testing.TB) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("MOSMIX_L_2026102415_10410.kml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(fixtureKML(t)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestParseMOSMIX(t *testing.T) {
	for name, data := range map[string][]byte{"kml": fixtureKML(t), "kmz": fixtureKMZ(t)} {
		t.Run(name, func(t *testing.T) {
			hours, err := weather.ParseMOSMIX(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(hours) != 72 {
				t.Fatalf("got %d hours, want 72", len(hours))
			}
			if want := time.Date(2026, 10, 24, 15, 0, 0, 0, time.UTC); !hours[0].Time.Equal(want) {
				t.Errorf("first step %v, want %v", hours[0].Time, want)
			}
			h := hours[0]
			if h.TempC == nil || h.WindKmh == nil || h.RainProb == nil || h.RainMM == nil {
				t.Fatalf("first hour incomplete: %+v", h)
			}
			// Fixture: TTT 284.65 K -> 11.5 C, FF 3.0 m/s -> 10.8 km/h, R101 5 %.
			if math.Abs(*h.TempC-(284.65-273.15)) > 0.1 || math.Abs(*h.WindKmh-10.8) > 0.1 || *h.RainProb != 5 {
				t.Errorf("first hour values: temp %v wind %v prob %v", *h.TempC, *h.WindKmh, *h.RainProb)
			}
			if hours[10].RainProb != nil {
				t.Errorf("hour 10 R101 should be missing, got %v", *hours[10].RainProb)
			}
			if hours[7].RainMM == nil || !near(*hours[7].RainMM, 1.2) || *hours[7].RainProb != 70 {
				t.Errorf("hour 7 should be rainy: %+v", hours[7])
			}
		})
	}
}

// The real DWD files declare ISO-8859-1 and carry Latin-1 station names.
func TestParseMOSMIXLatin1(t *testing.T) {
	data := bytes.Replace(fixtureKML(t), []byte(`encoding="UTF-8"`), []byte(`encoding="ISO-8859-1"`), 1)
	data = bytes.Replace(data, []byte("<kml:name>10410</kml:name>"),
		[]byte("<kml:name>10410</kml:name>\n   <kml:description>M\xdcNSTER</kml:description>"), 1)
	hours, err := weather.ParseMOSMIX(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(hours) != 72 {
		t.Fatalf("got %d hours, want 72", len(hours))
	}

	other := bytes.Replace(fixtureKML(t), []byte(`encoding="UTF-8"`), []byte(`encoding="KOI8-R"`), 1)
	if _, err := weather.ParseMOSMIX(other); err == nil {
		t.Error("unknown charset: expected error")
	}
}

func TestParseMOSMIXErrors(t *testing.T) {
	for name, data := range map[string]string{
		"garbage":      "not xml",
		"no steps":     `<kml:kml xmlns:kml="http://www.opengis.net/kml/2.2"><kml:Document/></kml:kml>`,
		"bad zip":      "PKnotazip",
		"length clash": `<kml xmlns:dwd="x"><Document><ExtendedData><dwd:ProductDefinition><dwd:ForecastTimeSteps><dwd:TimeStep>2026-10-24T15:00:00.000Z</dwd:TimeStep></dwd:ForecastTimeSteps></dwd:ProductDefinition></ExtendedData><Placemark><name>1</name><ExtendedData><dwd:Forecast dwd:elementName="TTT"><dwd:value>1 2</dwd:value></dwd:Forecast></ExtendedData></Placemark></Document></kml>`,
	} {
		if _, err := weather.ParseMOSMIX([]byte(data)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestSummarizeFixture(t *testing.T) {
	loc := berlin(t)
	hours, err := weather.ParseMOSMIX(fixtureKML(t))
	if err != nil {
		t.Fatal(err)
	}
	// Sat 24 Oct 18:00 CEST = 16:00Z until Sun 25 Oct 12:30 CET = 11:30Z.
	sum, err := weather.Summarize(hours, loc, time.Date(2026, 10, 24, 9, 0, 0, 0, loc), weather.DefaultWindow)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Day != "2026-10-24" {
		t.Errorf("day %s", sum.Day)
	}
	if !sum.WindowStart.Equal(time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)) ||
		!sum.WindowEnd.Equal(time.Date(2026, 10, 25, 11, 30, 0, 0, time.UTC)) {
		t.Errorf("window %v - %v", sum.WindowStart, sum.WindowEnd)
	}
	// Recompute the expectations by brute force from the parsed hours.
	minC := math.Inf(1)
	var n int
	for _, h := range hours {
		if !h.Time.Before(sum.WindowStart) && !h.Time.After(sum.WindowEnd) && h.TempC != nil {
			minC = math.Min(minC, *h.TempC)
			n++
		}
	}
	if n != 20 || sum.Hours != 20 {
		t.Errorf("hours in window: brute force %d, summary %d, want 20", n, sum.Hours)
	}
	if math.Abs(sum.NightMinC-minC) > 0.05 {
		t.Errorf("night min %v, want %v", sum.NightMinC, minC)
	}
	// Fixture: four hours with 70 % and 1.2 mm in the window.
	if sum.RainProbability != 70 || !near(sum.RainMM, 4.8) || !sum.WillRain {
		t.Errorf("rain: prob %d mm %v will %v", sum.RainProbability, sum.RainMM, sum.WillRain)
	}
	if sum.WindKmh <= 0 {
		t.Errorf("wind %v", sum.WindKmh)
	}

	// Beyond the forecast horizon there is no data.
	if _, err := weather.Summarize(hours, loc, time.Date(2026, 11, 20, 0, 0, 0, 0, loc), weather.DefaultWindow); !errors.Is(err, weather.ErrNoData) {
		t.Errorf("far future: err %v", err)
	}
}

func f(v float64) *float64 { return &v }

// hoursFrom builds an hourly series from start to end (inclusive) with a
// constant temperature and no rain.
func hoursFrom(start, end time.Time, temp float64) []weather.Hour {
	var out []weather.Hour
	for t := start; !t.After(end); t = t.Add(time.Hour) {
		out = append(out, weather.Hour{Time: t, TempC: f(temp), RainProb: f(0), RainMM: f(0), WindKmh: f(5)})
	}
	return out
}

func TestSummarizeDST(t *testing.T) {
	loc := berlin(t)
	tests := []struct {
		name      string
		day       time.Time
		wantHours time.Duration
	}{
		{"autumn changeover night (25 h day)", time.Date(2026, 10, 24, 12, 0, 0, 0, loc), 19*time.Hour + 30*time.Minute},
		{"night after changeover", time.Date(2026, 10, 25, 12, 0, 0, 0, loc), 18*time.Hour + 30*time.Minute},
		{"spring changeover night (23 h day)", time.Date(2027, 3, 27, 12, 0, 0, 0, loc), 17*time.Hour + 30*time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hs := hoursFrom(tt.day.Add(-48*time.Hour), tt.day.Add(72*time.Hour), 4)
			sum, err := weather.Summarize(hs, loc, tt.day, weather.DefaultWindow)
			if err != nil {
				t.Fatal(err)
			}
			if got := sum.WindowEnd.Sub(sum.WindowStart); got != tt.wantHours {
				t.Errorf("window length %v, want %v", got, tt.wantHours)
			}
			if sum.Hours != int(tt.wantHours/time.Hour)+1 {
				t.Errorf("steps %d, want %d", sum.Hours, int(tt.wantHours/time.Hour)+1)
			}
			if h, m, _ := sum.WindowStart.In(loc).Clock(); h != 18 || m != 0 {
				t.Errorf("start %v", sum.WindowStart.In(loc))
			}
			if h, m, _ := sum.WindowEnd.In(loc).Clock(); h != 12 || m != 30 {
				t.Errorf("end %v", sum.WindowEnd.In(loc))
			}
		})
	}
}

func TestSummarizeWindowEdges(t *testing.T) {
	loc := berlin(t)
	day := time.Date(2026, 10, 24, 0, 0, 0, 0, loc)
	start := time.Date(2026, 10, 24, 18, 0, 0, 0, loc)
	end := time.Date(2026, 10, 25, 12, 30, 0, 0, loc)
	last := time.Date(2026, 10, 25, 12, 0, 0, 0, loc) // last whole-hour step inside the window
	hs := hoursFrom(start.Add(-3*time.Hour), end.Add(3*time.Hour), 6)
	for i := range hs {
		switch {
		case hs[i].Time.Equal(start.Add(-time.Hour)), hs[i].Time.Equal(last.Add(time.Hour)):
			hs[i].TempC = f(-20) // outside the window
		case hs[i].Time.Equal(last):
			hs[i].TempC = f(-2) // last step is inside
		case hs[i].Time.Equal(start):
			hs[i].RainProb, hs[i].RainMM = f(90), f(3) // precipitation of the hour before the window
		}
	}
	sum, err := weather.Summarize(hs, loc, day, weather.DefaultWindow)
	if err != nil {
		t.Fatal(err)
	}
	if sum.NightMinC != -2 {
		t.Errorf("night min %v, want -2", sum.NightMinC)
	}
	if sum.WillRain || sum.RainProbability != 0 || sum.RainMM != 0 {
		t.Errorf("rain before the window leaked in: %+v", sum)
	}
}

func TestSummarizeRainDetails(t *testing.T) {
	loc := berlin(t)
	day := time.Date(2026, 10, 24, 0, 0, 0, 0, loc)
	at := func(d, h, m int) time.Time { return time.Date(2026, 10, d, h, m, 0, 0, loc) }
	hs := hoursFrom(at(24, 15, 0), at(25, 15, 0), 5)
	for i := range hs {
		switch {
		case hs[i].Time.Equal(at(24, 22, 0)):
			hs[i].RainMM, hs[i].RainProb = f(2), f(80)
		case hs[i].Time.Equal(at(24, 23, 0)):
			hs[i].RainMM, hs[i].RainProb = f(1), f(60)
		case hs[i].Time.Equal(at(25, 13, 0)):
			hs[i].RainMM, hs[i].RainProb = f(1), f(40) // 12:00-13:00, half of it counts
		case hs[i].Time.Equal(at(25, 14, 0)):
			hs[i].RainMM, hs[i].RainProb = f(9), f(90) // after the window
		case hs[i].Time.Equal(at(25, 10, 0)):
			hs[i].TempC = f(11)
		}
	}
	sum, err := weather.Summarize(hs, loc, day, weather.DefaultWindow)
	if err != nil {
		t.Fatal(err)
	}
	if !near(sum.RainMM, 3.5) || sum.RainHours != 3 || sum.RainPeakMM != 2 || sum.RainProbability != 80 {
		t.Errorf("rain: mm %v hours %d peak %v prob %d", sum.RainMM, sum.RainHours, sum.RainPeakMM, sum.RainProbability)
	}
	if !sum.RainFrom.Equal(at(24, 21, 0)) || !sum.RainUntil.Equal(at(25, 12, 30)) {
		t.Errorf("rain from %v until %v", sum.RainFrom, sum.RainUntil)
	}
	if sum.NightMinC != 5 || sum.TempMaxC != 11 {
		t.Errorf("temp %v..%v", sum.NightMinC, sum.TempMaxC)
	}
	// 18:00 .. 12:00 hourly steps; the first one carries no rain.
	if len(sum.Timeline) != 20 || !sum.Timeline[0].Time.Equal(at(24, 18, 0)) || !sum.Timeline[19].Time.Equal(at(25, 12, 0)) {
		t.Fatalf("timeline has %d points", len(sum.Timeline))
	}
	if p := sum.Timeline[4]; p.RainMM != 2 || p.RainProb != 80 {
		t.Errorf("point 22:00: %+v", p)
	}

	dry, err := weather.Summarize(hoursFrom(at(24, 15, 0), at(25, 15, 0), 5), loc, day, weather.DefaultWindow)
	if err != nil {
		t.Fatal(err)
	}
	if dry.RainHours != 0 || !dry.RainFrom.IsZero() || !dry.RainUntil.IsZero() || dry.WillRain {
		t.Errorf("dry: %+v", dry)
	}
}

func TestWillRainThresholds(t *testing.T) {
	loc := berlin(t)
	day := time.Date(2026, 10, 24, 0, 0, 0, 0, loc)
	at := time.Date(2026, 10, 24, 22, 0, 0, 0, loc)
	tests := []struct {
		name string
		prob float64
		mm   float64
		want bool
	}{
		{"dry", 10, 0, false},
		{"prob just below", 49, 0.4, false},
		{"prob at threshold", weather.RainProbabilityThreshold, 0, true},
		{"mm at threshold", 10, weather.RainSumThresholdMM, true},
	}
	for _, tt := range tests {
		hs := hoursFrom(at.Add(-4*time.Hour), at.Add(4*time.Hour), 5)
		for i := range hs {
			if hs[i].Time.Equal(at) {
				hs[i].RainProb, hs[i].RainMM = f(tt.prob), f(tt.mm)
			}
		}
		sum, err := weather.Summarize(hs, loc, day, weather.DefaultWindow)
		if err != nil {
			t.Fatal(err)
		}
		if sum.WillRain != tt.want {
			t.Errorf("%s: will_rain %v, want %v", tt.name, sum.WillRain, tt.want)
		}
	}

	// Several light hours add up to the sum threshold.
	hs := hoursFrom(at.Add(-4*time.Hour), at.Add(4*time.Hour), 5)
	for i := range hs {
		if !hs[i].Time.Before(at) && hs[i].Time.Before(at.Add(3*time.Hour)) {
			hs[i].RainMM = f(0.2)
		}
	}
	sum, _ := weather.Summarize(hs, loc, day, weather.DefaultWindow)
	if !sum.WillRain {
		t.Errorf("0.6 mm summed should count as rain: %+v", sum)
	}
}

func TestNearestStation(t *testing.T) {
	if got := weather.NearestStation(51.66, 6.96); got.ID != "10410" {
		t.Errorf("Dorsten -> %s, want 10410", got.ID)
	}
	if got := weather.NearestStation(52.0, 7.6); got.ID != "10315" {
		t.Errorf("Muenster -> %s, want 10315", got.ID)
	}
	if got := weather.StationByID("99999"); got.ID != "99999" {
		t.Errorf("unknown id not passed through: %+v", got)
	}
}

func TestDWDFetch(t *testing.T) {
	kmz := fixtureKMZ(t)
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Path == "/weather/local_forecasts/mos/MOSMIX_L/single_stations/10410/kml/MOSMIX_L_LATEST_10410.kmz" {
			_, _ = w.Write(kmz)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	d := &weather.DWD{Client: srv.Client(), BaseURL: srv.URL + "/"}
	hours, err := d.Fetch(context.Background(), weather.StationByID("10410"))
	if err != nil {
		t.Fatal(err)
	}
	if len(hours) != 72 {
		t.Errorf("got %d hours", len(hours))
	}
	if _, err := d.Fetch(context.Background(), weather.StationByID("00000")); err == nil {
		t.Errorf("expected error for 404 (last path %s)", gotPath)
	}
	if got := (&weather.DWD{}).URL(weather.StationByID("10410")); got !=
		"https://opendata.dwd.de/weather/local_forecasts/mos/MOSMIX_L/single_stations/10410/kml/MOSMIX_L_LATEST_10410.kmz" {
		t.Errorf("default URL %s", got)
	}
}

type fakeFetcher struct {
	hours []weather.Hour
	err   error
	calls []string
}

func (f *fakeFetcher) Fetch(_ context.Context, st weather.Station) ([]weather.Hour, error) {
	f.calls = append(f.calls, st.ID)
	return f.hours, f.err
}

func TestStoreAndService(t *testing.T) {
	loc := berlin(t)
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()
	hours, err := weather.ParseMOSMIX(fixtureKML(t))
	if err != nil {
		t.Fatal(err)
	}
	store := &weather.Store{Pool: pool}

	if _, err := store.Latest(ctx, seed.StableB, time.Date(2026, 10, 24, 0, 0, 0, 0, loc)); !errors.Is(err, weather.ErrNotFound) {
		t.Fatalf("empty store: err %v", err)
	}

	now := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	ff := &fakeFetcher{hours: hours}
	svc := &weather.Service{Pool: pool, Fetcher: ff, Now: func() time.Time { return now }}
	if err := svc.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if len(ff.calls) != 1 || ff.calls[0] != "10410" {
		t.Errorf("fetch calls %v, want one for 10410", ff.calls)
	}

	day := time.Date(2026, 10, 24, 0, 0, 0, 0, loc)
	snap, err := store.Latest(ctx, seed.StableB, day)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := weather.Summarize(hours, loc, day, weather.DefaultWindow)
	if snap.ValidFor != "2026-10-24" || snap.NightMinC != want.NightMinC || snap.RainProb != 70 ||
		!near(snap.RainMM, want.RainMM) || snap.WindKmh != want.WindKmh || !snap.WillRain ||
		!snap.FetchedAt.Equal(now) || snap.StationID != "10410" || snap.RawSummary.Hours != 16 {
		t.Errorf("snapshot %+v, want summary %+v", snap, want)
	}
	if _, err := store.Latest(ctx, seed.StableB, day.AddDate(0, 0, 1)); err != nil {
		t.Errorf("tomorrow's snapshot missing: %v", err)
	}
	if _, err := store.Latest(ctx, seed.StableB, day.AddDate(0, 0, 5)); !errors.Is(err, weather.ErrNotFound) {
		t.Errorf("day 5: err %v", err)
	}

	// A later run appends; Latest returns the newest.
	later := now.Add(time.Hour)
	svc.Now = func() time.Time { return later }
	svc.StationID = "10400"
	if err := svc.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	snap, _ = store.Latest(ctx, seed.StableB, day)
	if !snap.FetchedAt.Equal(later) || snap.StationID != "10400" {
		t.Errorf("latest not the newest: %+v", snap)
	}

	// Fetch errors are reported.
	svc.Fetcher = &fakeFetcher{err: errors.New("dwd down")}
	if err := svc.Refresh(ctx); err == nil {
		t.Error("expected error when the fetch fails")
	}
}

// A stable can set its own covered window; the summary follows it.
func TestSummarizeCustomWindow(t *testing.T) {
	loc := berlin(t)
	day := time.Date(2026, 10, 24, 0, 0, 0, 0, loc)
	at := func(d, h, m int) time.Time { return time.Date(2026, 10, d, h, m, 0, 0, loc) }
	hs := hoursFrom(at(24, 12, 0), at(25, 16, 0), 5)
	for i := range hs {
		switch {
		case hs[i].Time.Equal(at(24, 16, 0)):
			hs[i].TempC = f(-1) // inside the custom window only
		case hs[i].Time.Equal(at(25, 12, 0)):
			hs[i].TempC = f(-3) // after the custom end, inside the default one
		case hs[i].Time.Equal(at(25, 8, 0)):
			hs[i].RainMM, hs[i].RainProb = f(1), f(70)
		}
	}
	win := weather.Window{StartMinutes: 16 * 60, EndMinutes: 8*60 + 30}
	sum, err := weather.Summarize(hs, loc, day, win)
	if err != nil {
		t.Fatal(err)
	}
	if !sum.WindowStart.Equal(at(24, 16, 0)) || !sum.WindowEnd.Equal(at(25, 8, 30)) {
		t.Errorf("window %v to %v", sum.WindowStart, sum.WindowEnd)
	}
	if sum.NightMinC != -1 {
		t.Errorf("night min %v, want -1 (the step at 12:00 lies outside)", sum.NightMinC)
	}
	// The hour 07:00-08:00 is fully inside, so the rain counts.
	if !sum.WillRain || !near(sum.RainMM, 1) {
		t.Errorf("rain %+v", sum)
	}
	if def, _ := weather.Summarize(hs, loc, day, weather.DefaultWindow); def.NightMinC != -3 {
		t.Errorf("default window min %v, want -3", def.NightMinC)
	}
}
