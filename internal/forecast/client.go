package forecast

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"forecaster/internal/config"
)

const (
	baseURL      = "https://api.open-meteo.com/v1/forecast"
	forecastDays = 7
	timezone     = "Europe/Stockholm"
	timeout      = 15 * time.Second
)

// OpenMeteoResponse mirrors the JSON shape returned by the Open-Meteo API.
type OpenMeteoResponse struct {
	Hourly HourlyBlock `json:"hourly"`
}

// HourlyBlock holds the arrays for each hourly field.
type HourlyBlock struct {
	Time               []string  `json:"time"`
	IsDay             []int     `json:"is_day"`
	PrecipitationProb  []float64 `json:"precipitation_probability"`
	Temperature2m      []float64 `json:"temperature_2m"`
	CloudCover         []float64 `json:"cloud_cover"`
	WindSpeed10m       []float64 `json:"wind_speed_10m"`
	WindDirection10m   []float64 `json:"wind_direction_10m"`
	WindGusts10m       []float64 `json:"wind_gusts_10m"`
}

// SiteResult holds the result of fetching a single site.
type SiteResult struct {
	Site  config.Site
	Data  *OpenMeteoResponse
	Error error
}

// FetchSite fetches the 7-day forecast for a single site from Open-Meteo.
func FetchSite(site config.Site) (*OpenMeteoResponse, error) {
	params := fmt.Sprintf(
		"latitude=%.6f&longitude=%.6f&hourly=is_day,precipitation_probability,temperature_2m,cloud_cover,wind_speed_10m,wind_direction_10m,wind_gusts_10m&timezone=%s&past_days=0&forecast_days=%d",
		site.Lat, site.Lon, timezone, forecastDays,
	)
	url := baseURL + "?" + params

	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", site.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, site.Name)
	}

	var data OpenMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding response for %s: %w", site.Name, err)
	}

	return &data, nil
}

// FetchAll concurrently fetches forecasts for all sites.
// Errors are isolated per site; a failed site does not cancel others.
func FetchAll(sites []config.Site) []SiteResult {
	results := make([]SiteResult, len(sites))
	var wg sync.WaitGroup

	for i, site := range sites {
		wg.Add(1)
		go func(i int, site config.Site) {
			defer wg.Done()
			data, err := FetchSite(site)
			results[i] = SiteResult{Site: site, Data: data, Error: err}
		}(i, site)
	}

	wg.Wait()
	return results
}