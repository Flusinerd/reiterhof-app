// Package config liest die Konfiguration des Dienstes aus der Umgebung.
package config

import "os"

// Config bündelt alle Einstellungen des API-Dienstes.
type Config struct {
	// Addr ist die Adresse, auf der der HTTP-Server lauscht.
	Addr string
}

// FromEnv liest die Konfiguration aus Umgebungsvariablen und setzt Standardwerte.
func FromEnv() Config {
	return Config{
		Addr: getenv("REITERHOF_ADDR", ":8080"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
