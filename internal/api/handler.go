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
	cache *forecast.Cache
}

// NewHandler creates a Handler with the given cache.
func NewHandler(cache *forecast.Cache) *Handler {
	return &Handler{cache: cache}
}

// ForecastResponse is the JSON shape returned by GET /api/forecast.
type ForecastResponse struct {
	Sites     []forecast.SiteData `json:"sites"`
	FetchedAt string              `json:"fetched_at"`
}

// ServeHTTP handles GET /api/forecast.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	results := forecast.FetchAll(config.Sites, h.cache)
	sites   := forecast.ProcessSites(results)

	resp := ForecastResponse{
		Sites:     sites,
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}