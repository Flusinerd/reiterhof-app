// Package config reads the service configuration from the environment.
package config

import "os"

// Config holds all settings of the API service.
type Config struct {
	// Addr is the address the HTTP server listens on.
	Addr string
	// DatabaseURL is the Postgres connection string (REITERHOF_DATABASE_URL).
	DatabaseURL string
	// Auth holds the authentication settings.
	Auth Auth
}

// FromEnv reads the configuration from environment variables and applies defaults.
func FromEnv() Config {
	return Config{
		Addr:        getenv("REITERHOF_ADDR", ":8080"),
		DatabaseURL: getenv("REITERHOF_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/reiterhof?sslmode=disable"),
		Auth:        authFromEnv(),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
