package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Severity values of a warning (the CAP severity of the DWD, lower case).
const (
	SeverityMinor    = "minor"    // Wetterwarnung (level 1)
	SeverityModerate = "moderate" // markante Wetterwarnung (level 2)
	SeveritySevere   = "severe"   // Unwetterwarnung (level 3)
	SeverityExtreme  = "extreme"  // extremes Unwetter (level 4)
)

// SeverityRank orders the severities: minor 1 .. extreme 4, unknown 0.
func SeverityRank(s string) int {
	switch s {
	case SeverityMinor:
		return 1
	case SeverityModerate:
		return 2
	case SeveritySevere:
		return 3
	case SeverityExtreme:
		return 4
	}
	return 0
}

// Warning is an active official weather warning of the DWD for a location.
type Warning struct {
	// DWDID is the CAP identifier of the message. An update of a warning gets a new one.
	DWDID       string
	Event       string // e.g. "STURMBÖEN", "FROST", "GLATTEIS"
	Severity    string // Severity* constant
	Headline    string // e.g. "Amtliche WARNUNG vor STURMBÖEN"
	Description string
	Instruction string
	Onset       time.Time
	Expires     *time.Time // nil: until further notice
}

// horseEvents are the parts of DWD event names that matter for horses at moderate
// severity (storm on the paddock, frost and ice, heat). Thunderstorms, rain and fog of
// that level are frequent and only shown, not pushed.
var horseEvents = []string{"STURM", "ORKAN", "FROST", "GLÄTTE", "GLATTEIS", "HITZE"}

// Notify reports whether members get a push for the warning: every severe and extreme
// warning, and moderate ones about storm, frost, ice and heat.
func (w Warning) Notify() bool {
	switch w.Severity {
	case SeveritySevere, SeverityExtreme:
		return true
	case SeverityModerate:
		ev := strings.ToUpper(w.Event)
		for _, e := range horseEvents {
			if strings.Contains(ev, e) {
				return true
			}
		}
	}
	return false
}

// WarningFetcher loads the active warnings for a point.
type WarningFetcher interface {
	FetchWarnings(ctx context.Context, lat, lng float64) ([]Warning, error)
}

// DefaultWarningsBaseURL is the GeoServer of the DWD (open data, no key).
const DefaultWarningsBaseURL = "https://maps.dwd.de"

// DWDWarnings fetches the warnings of the municipality ("Gemeinde") that contains the
// stable from the WFS layer dwd:Warnungen_Gemeinden of the DWD GeoServer. A tiny
// bounding box around the point selects the municipality, so no warn cell id is needed.
type DWDWarnings struct {
	// Client defaults to an http.Client with a 30 s timeout.
	Client *http.Client
	// BaseURL defaults to DefaultWarningsBaseURL; tests point it at httptest.
	BaseURL string
}

// bboxPad is half the side of the bounding box around the stable (about 50 m).
const bboxPad = 0.0005

// URL is the WFS GetFeature request for the point. With the urn CRS the axis order is
// latitude, longitude.
func (d *DWDWarnings) URL(lat, lng float64) string {
	base := strings.TrimRight(d.BaseURL, "/")
	if base == "" {
		base = DefaultWarningsBaseURL
	}
	q := url.Values{}
	q.Set("service", "WFS")
	q.Set("version", "2.0.0")
	q.Set("request", "GetFeature")
	q.Set("typeName", "dwd:Warnungen_Gemeinden")
	q.Set("outputFormat", "application/json")
	q.Set("count", "100")
	q.Set("bbox", fmt.Sprintf("%.5f,%.5f,%.5f,%.5f,urn:ogc:def:crs:EPSG::4326",
		lat-bboxPad, lng-bboxPad, lat+bboxPad, lng+bboxPad))
	return base + "/geoserver/dwd/ows?" + q.Encode()
}

// FetchWarnings implements WarningFetcher.
func (d *DWDWarnings) FetchWarnings(ctx context.Context, lat, lng float64) ([]Warning, error) {
	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.URL(lat, lng), nil)
	if err != nil {
		return nil, fmt.Errorf("weather warnings: build request: %w", err)
	}
	req.Header.Set("User-Agent", "reiterhof-app/1.0 (weather warnings job)")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weather warnings: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather warnings: fetch: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload))
	if err != nil {
		return nil, fmt.Errorf("weather warnings: read: %w", err)
	}
	return ParseWarnings(body)
}

type warningCollection struct {
	Type     string `json:"type"`
	Features []struct {
		Properties struct {
			Identifier  string  `json:"IDENTIFIER"`
			Status      string  `json:"STATUS"`
			MsgType     string  `json:"MSGTYPE"`
			Event       string  `json:"EVENT"`
			Severity    string  `json:"SEVERITY"`
			Headline    string  `json:"HEADLINE"`
			Description string  `json:"DESCRIPTION"`
			Instruction string  `json:"INSTRUCTION"`
			Sent        string  `json:"SENT"`
			Onset       string  `json:"ONSET"`
			Expires     *string `json:"EXPIRES"`
		} `json:"properties"`
	} `json:"features"`
}

// ParseWarnings reads the GeoJSON answer of the WFS layer. Every municipality polygon is
// one feature, so a warning can come several times; it is returned once. Test and
// exercise messages and cancellations are skipped. A body that is no feature collection
// (the GeoServer answers errors as XML with status 200) is an error.
func ParseWarnings(body []byte) ([]Warning, error) {
	var fc warningCollection
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, fmt.Errorf("weather warnings: parse: %w", err)
	}
	if fc.Type != "FeatureCollection" {
		return nil, fmt.Errorf("weather warnings: parse: not a feature collection")
	}
	out := []Warning{}
	seen := map[string]bool{}
	for _, f := range fc.Features {
		p := f.Properties
		if p.Identifier == "" || seen[p.Identifier] {
			continue
		}
		if !strings.EqualFold(p.Status, "Actual") || strings.EqualFold(p.MsgType, "Cancel") {
			continue
		}
		sev := strings.ToLower(p.Severity)
		if SeverityRank(sev) == 0 {
			continue
		}
		onset, err := parseCAPTime(p.Onset)
		if err != nil {
			if onset, err = parseCAPTime(p.Sent); err != nil {
				return nil, fmt.Errorf("weather warnings: warning %s without onset", p.Identifier)
			}
		}
		w := Warning{
			DWDID: p.Identifier, Event: strings.TrimSpace(p.Event), Severity: sev,
			Headline: strings.TrimSpace(p.Headline), Description: strings.TrimSpace(p.Description),
			Instruction: strings.TrimSpace(p.Instruction), Onset: onset,
		}
		if w.Headline == "" {
			w.Headline = "Amtliche WARNUNG vor " + w.Event
		}
		if p.Expires != nil && *p.Expires != "" {
			if exp, err := parseCAPTime(*p.Expires); err == nil {
				w.Expires = &exp
			}
		}
		seen[p.Identifier] = true
		out = append(out, w)
	}
	return out, nil
}

func parseCAPTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.TrimSpace(s))
}
