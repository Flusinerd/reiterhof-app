// Package config reads the service configuration from the environment.
package config

import (
	"os"
	"strconv"
)

// Config holds all settings of the API service.
type Config struct {
	// Addr is the address the HTTP server listens on.
	Addr string
	// DatabaseURL is the Postgres connection string (REITERHOF_DATABASE_URL).
	DatabaseURL string
	// WeatherEnabled switches the hourly DWD weather job (REITERHOF_WEATHER_ENABLED, default true).
	WeatherEnabled bool
	// WeatherStation forces one MOSMIX station id for all stables
	// (REITERHOF_WEATHER_STATION); empty picks the nearest station per stable.
	WeatherStation string
	// Auth holds the authentication settings.
	Auth Auth
}

// FromEnv reads the configuration from environment variables and applies defaults.
func FromEnv() Config {
	return Config{
		Addr:           getenv("REITERHOF_ADDR", ":8080"),
		DatabaseURL:    getenv("REITERHOF_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/reiterhof?sslmode=disable"),
		WeatherEnabled: getbool("REITERHOF_WEATHER_ENABLED", true),
		WeatherStation: os.Getenv("REITERHOF_WEATHER_STATION"),
		Auth:           authFromEnv(),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getbool parses a boolean variable; unset or unparsable values give the fallback.
func getbool(key string, fallback bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(key)); err == nil {
		return v
	}
	return fallback
}
