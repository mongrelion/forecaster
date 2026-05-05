package config

import (
	"os"
	"strconv"
	"time"
)

// ServerConfig holds all deploy-time configuration for the forecaster server.
// Each field is populated from an environment variable with a documented default.
type ServerConfig struct {
	// Host is the network interface to bind to (e.g. "0.0.0.0").
	// Env: HOST (default: "" — all interfaces).
	Host string

	// Port is the TCP port to listen on.
	// Env: PORT (default: "8080").
	Port string

	// PublicDir is the file path to the frontend static assets directory.
	// Env: PUBLIC_DIR (default: "public").
	PublicDir string

	// SitesPath is the file path to the flying sites JSON database.
	// Env: SITES_PATH (default: "sites.json").
	SitesPath string

	// OpenMeteoURL is the base URL of the Open-Meteo forecast API.
	// Env: OPEN_METEO_URL (default: "https://api.open-meteo.com/v1/forecast").
	OpenMeteoURL string

	// ForecastDays is the number of forecast days to request from the API.
	// Env: FORECAST_DAYS (default: 7).
	ForecastDays int

	// Timezone is the IANA timezone string used for forecast timestamps.
	// Env: TIMEZONE (default: "Europe/Stockholm").
	Timezone string

	// HTTPTimeout is the timeout for HTTP requests to the Open-Meteo API.
	// Env: HTTP_TIMEOUT (default: 15s — value is in seconds).
	HTTPTimeout time.Duration

	// MaxGusts is the maximum safe wind gust speed in km/h.
	// Hours with gusts above this threshold are considered not flyable.
	// Env: MAX_GUSTS (default: 25).
	MaxGusts float64
}

// envStr reads a string env var, falling back to the provided default.
func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// envInt reads an int env var, falling back to the provided default.
func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// envDuration reads a duration env var (in seconds), falling back to the
// provided default.
func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return def
}

// envFloat reads a float64 env var, falling back to the provided default.
func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// LoadServerConfig reads all environment variables and returns a populated
// ServerConfig with appropriate defaults.
//
// Environment variables:
//
//	HOST             — Host to bind (default: "" — all interfaces)
//	PORT             — Port to listen on (default: "8080")
//	PUBLIC_DIR       — Path to frontend static assets (default: "public")
//	SITES_PATH       — Path to sites JSON database (default: "sites.json")
//	OPEN_METEO_URL   — Base URL for Open-Meteo API (default: "https://api.open-meteo.com/v1/forecast")
//	FORECAST_DAYS    — Number of forecast days (default: 7)
//	TIMEZONE         — IANA timezone for timestamps (default: "Europe/Stockholm")
//	HTTP_TIMEOUT     — HTTP client timeout in seconds (default: 15)
//	MAX_GUSTS        — Maximum safe wind gusts in km/h (default: 25)
func LoadServerConfig() ServerConfig {
	return ServerConfig{
		Host:         envStr("HOST", ""),
		Port:         envStr("PORT", "8080"),
		PublicDir:    envStr("PUBLIC_DIR", "public"),
		SitesPath:    envStr("SITES_PATH", "sites.json"),
		OpenMeteoURL: envStr("OPEN_METEO_URL", "https://api.open-meteo.com/v1/forecast"),
		ForecastDays: envInt("FORECAST_DAYS", 7),
		Timezone:     envStr("TIMEZONE", "Europe/Stockholm"),
		HTTPTimeout:  envDuration("HTTP_TIMEOUT", 15*time.Second),
		MaxGusts:     envFloat("MAX_GUSTS", 25),
	}
}
