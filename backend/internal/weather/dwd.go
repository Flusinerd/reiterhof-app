package weather

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// Station is a MOSMIX forecast station.
type Station struct {
	ID       string // MOSMIX id, e.g. "10410"
	Name     string
	Lat, Lng float64
}

// Stations is a small embedded list of MOSMIX stations around the Ruhr area.
// Coordinates are rounded and were entered from memory of the DWD station
// catalogue (mosmix_stationskatalog.cfg); extend the list or set
// REITERHOF_WEATHER_STATION when a stable is elsewhere. Only the ids matter for
// the fetch; the coordinates only pick the nearest station.
var Stations = []Station{
	{ID: "10410", Name: "Essen-Bredeney", Lat: 51.40, Lng: 6.97},
	{ID: "10400", Name: "Duesseldorf", Lat: 51.30, Lng: 6.77},
	{ID: "10315", Name: "Muenster/Osnabrueck", Lat: 52.13, Lng: 7.70},
	{ID: "10513", Name: "Koeln/Bonn", Lat: 50.87, Lng: 7.16},
}

// StationByID looks a station up in Stations; unknown ids are accepted as
// long as they are non-empty (the id is all the DWD URL needs).
func StationByID(id string) Station {
	for _, s := range Stations {
		if s.ID == id {
			return s
		}
	}
	return Station{ID: id, Name: id}
}

// NearestStation returns the station of Stations closest to the point.
func NearestStation(lat, lng float64) Station {
	best, bestD := Stations[0], math.Inf(1)
	for _, s := range Stations {
		if d := distanceKm(lat, lng, s.Lat, s.Lng); d < bestD {
			best, bestD = s, d
		}
	}
	return best
}

func distanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371.0
	rad := math.Pi / 180
	dLat, dLng := (lat2-lat1)*rad, (lng2-lng1)*rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * r * math.Asin(math.Sqrt(a))
}

// Fetcher loads the hourly forecast of a station.
type Fetcher interface {
	Fetch(ctx context.Context, station Station) ([]Hour, error)
}

// DefaultBaseURL is the DWD open data server.
const DefaultBaseURL = "https://opendata.dwd.de"

// maxDownload bounds the KMZ download (a station file is ~30 KB compressed).
const maxDownload = 16 << 20

// DWD fetches MOSMIX_L single station files from DWD open data.
type DWD struct {
	// Client defaults to an http.Client with a 30 s timeout.
	Client *http.Client
	// BaseURL defaults to DefaultBaseURL; tests point it at httptest.
	BaseURL string
}

// URL is the download URL of the latest MOSMIX_L file of the station.
func (d *DWD) URL(station Station) string {
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = DefaultBaseURL
	}
	return fmt.Sprintf("%s/weather/local_forecasts/mos/MOSMIX_L/single_stations/%[2]s/kml/MOSMIX_L_LATEST_%[2]s.kmz", base, station.ID)
}

// Fetch implements Fetcher.
func (d *DWD) Fetch(ctx context.Context, station Station) ([]Hour, error) {
	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.URL(station), nil)
	if err != nil {
		return nil, fmt.Errorf("weather: build request: %w", err)
	}
	req.Header.Set("User-Agent", "reiterhof-app/1.0 (weather job)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weather: fetch station %s: %w", station.ID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather: fetch station %s: HTTP %d", station.ID, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload))
	if err != nil {
		return nil, fmt.Errorf("weather: read station %s: %w", station.ID, err)
	}
	return ParseMOSMIX(body)
}
