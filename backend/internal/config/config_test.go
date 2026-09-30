package config_test

import (
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
)

func TestWeatherConfig(t *testing.T) {
	t.Setenv("REITERHOF_WEATHER_ENABLED", "")
	t.Setenv("REITERHOF_WEATHER_STATION", "")
	if c := config.FromEnv(); !c.WeatherEnabled || c.WeatherStation != "" {
		t.Errorf("defaults: %+v", c)
	}
	t.Setenv("REITERHOF_WEATHER_ENABLED", "false")
	t.Setenv("REITERHOF_WEATHER_STATION", "10400")
	if c := config.FromEnv(); c.WeatherEnabled || c.WeatherStation != "10400" {
		t.Errorf("overrides: %+v", c)
	}
	t.Setenv("REITERHOF_WEATHER_ENABLED", "nonsense")
	if c := config.FromEnv(); !c.WeatherEnabled {
		t.Error("unparsable value should fall back to true")
	}
}
