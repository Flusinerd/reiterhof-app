// Package weather fetches the DWD MOSMIX point forecast, reduces it to a night
// summary per stable and stores it in weather_snapshots.
//
// Data source: DWD open data, product MOSMIX_L (single station KMZ files),
// see docs/architecture.md for the station choice.
package weather

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Hour is one forecast time step. Missing values are nil (DWD writes "-").
// For the accumulated values (RainMM, RainProb) the step covers the hour that
// ends at Time.
type Hour struct {
	Time     time.Time `json:"time"`
	TempC    *float64  `json:"temp_c,omitempty"`    // TTT, air temperature 2 m, converted from Kelvin
	RainProb *float64  `json:"rain_prob,omitempty"` // R101 (fallback wwP), probability of precipitation > 0.1 mm in the hour, percent
	RainMM   *float64  `json:"rain_mm,omitempty"`   // RR1c, precipitation in the hour, mm (kg/m2)
	WindKmh  *float64  `json:"wind_kmh,omitempty"`  // FF, mean wind speed, converted from m/s
}

// MOSMIX element names used by the parser.
const (
	elemTemp      = "TTT"
	elemRainProb  = "R101"
	elemRainProb2 = "wwP"
	elemRainMM    = "RR1c"
	elemWind      = "FF"
)

const kelvinOffset = 273.15

type kmlDoc struct {
	Steps      []string    `xml:"Document>ExtendedData>ProductDefinition>ForecastTimeSteps>TimeStep"`
	Placemarks []placemark `xml:"Document>Placemark"`
}

type placemark struct {
	Name      string     `xml:"name"`
	Forecasts []forecast `xml:"ExtendedData>Forecast"`
}

type forecast struct {
	Element string `xml:"elementName,attr"`
	Value   string `xml:"value"`
}

// charsetReader lets the decoder read the ISO-8859-1 declared by the DWD files.
// Latin-1 maps every byte to the code point of the same value, so no table is needed.
func charsetReader(label string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(label) {
	case "iso-8859-1", "iso8859-1", "latin1", "latin-1":
		b, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}
		var sb strings.Builder
		sb.Grow(len(b))
		for _, c := range b {
			sb.WriteRune(rune(c))
		}
		return strings.NewReader(sb.String()), nil
	}
	return nil, fmt.Errorf("unsupported charset %q", label)
}

// ParseMOSMIX parses a MOSMIX_L single-station KMZ (zip with one .kml) or a
// plain KML document and returns the hourly values in time order.
func ParseMOSMIX(data []byte) ([]Hour, error) {
	if bytes.HasPrefix(data, []byte("PK")) {
		kml, err := unzipKML(data)
		if err != nil {
			return nil, err
		}
		data = kml
	}
	var doc kmlDoc
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = charsetReader
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("weather: parse kml: %w", err)
	}
	if len(doc.Steps) == 0 {
		return nil, errors.New("weather: kml has no forecast time steps")
	}
	if len(doc.Placemarks) == 0 {
		return nil, errors.New("weather: kml has no placemark")
	}
	// Single-station files carry exactly one placemark.
	pm := doc.Placemarks[0]

	times := make([]time.Time, len(doc.Steps))
	for i, s := range doc.Steps {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
		if err != nil {
			return nil, fmt.Errorf("weather: time step %q: %w", s, err)
		}
		times[i] = t
	}

	series := map[string][]*float64{}
	for _, f := range pm.Forecasts {
		switch f.Element {
		case elemTemp, elemRainProb, elemRainProb2, elemRainMM, elemWind:
			vals, err := parseValues(f.Value)
			if err != nil {
				return nil, fmt.Errorf("weather: element %s: %w", f.Element, err)
			}
			if len(vals) != len(times) {
				return nil, fmt.Errorf("weather: element %s has %d values for %d time steps", f.Element, len(vals), len(times))
			}
			series[f.Element] = vals
		}
	}
	if series[elemTemp] == nil {
		return nil, errors.New("weather: kml has no TTT (temperature) series")
	}
	prob := series[elemRainProb]
	if prob == nil {
		prob = series[elemRainProb2]
	}

	hours := make([]Hour, len(times))
	for i, t := range times {
		h := Hour{Time: t}
		if v := series[elemTemp][i]; v != nil {
			c := *v - kelvinOffset
			h.TempC = &c
		}
		if prob != nil {
			h.RainProb = prob[i]
		}
		if s := series[elemRainMM]; s != nil {
			h.RainMM = s[i]
		}
		if s := series[elemWind]; s != nil && s[i] != nil {
			k := *s[i] * 3.6
			h.WindKmh = &k
		}
		hours[i] = h
	}
	return hours, nil
}

func parseValues(s string) ([]*float64, error) {
	fields := strings.Fields(s)
	out := make([]*float64, len(fields))
	for i, f := range fields {
		if f == "-" {
			continue
		}
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return nil, fmt.Errorf("value %q: %w", f, err)
		}
		out[i] = &v
	}
	return out, nil
}

// maxKML bounds the decompressed KML size (a station file is ~200 KB).
const maxKML = 32 << 20

func unzipKML(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("weather: open kmz: %w", err)
	}
	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".kml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("weather: open %s: %w", f.Name, err)
		}
		defer rc.Close()
		b, err := io.ReadAll(io.LimitReader(rc, maxKML+1))
		if err != nil {
			return nil, fmt.Errorf("weather: read %s: %w", f.Name, err)
		}
		if len(b) > maxKML {
			return nil, errors.New("weather: kml too large")
		}
		return b, nil
	}
	return nil, errors.New("weather: kmz contains no .kml file")
}
