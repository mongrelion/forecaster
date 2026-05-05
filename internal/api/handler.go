package api

import (
	"encoding/json"
	"net/http"
	"time"

	"forecaster/internal/config"
	"forecaster/internal/forecast"
)

// Handler serves forecast data to the frontend.
type Handler struct {
	sites []config.Site
	cache *forecast.Cache
	cfg   config.ServerConfig
}

// NewHandler creates a Handler with the given sites, cache, and config.
func NewHandler(sites []config.Site, cache *forecast.Cache, cfg config.ServerConfig) *Handler {
	return &Handler{sites: sites, cache: cache, cfg: cfg}
}

// ForecastResponse is the JSON shape returned by GET /api/forecast.
type ForecastResponse struct {
	Sites     []forecast.SiteData `json:"sites"`
	Model     string              `json:"model"`
	MaxGusts  float64             `json:"max_gusts"`
	FetchedAt string              `json:"fetched_at"`
}

// ServeHTTP handles GET /api/forecast.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	results := forecast.FetchAll(h.sites, h.cache, h.cfg)
	sites := forecast.ProcessSites(results)

	resp := ForecastResponse{
		Sites:     sites,
		Model:     config.ModelName,
		MaxGusts:  h.cfg.MaxGusts,
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Healthz returns 200 OK — used by Docker HEALTHCHECK.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
