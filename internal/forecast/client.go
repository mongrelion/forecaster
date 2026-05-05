package forecast

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"forecaster/internal/config"
)

// OpenMeteoResponse mirrors the JSON shape returned by the Open-Meteo API.
type OpenMeteoResponse struct {
	Hourly HourlyBlock `json:"hourly"`
}

// HourlyBlock holds the arrays for each hourly field.
type HourlyBlock struct {
	Time              []string  `json:"time"`
	IsDay             []int     `json:"is_day"`
	PrecipitationProb []float64 `json:"precipitation_probability"`
	Temperature2m     []float64 `json:"temperature_2m"`
	CloudCover        []float64 `json:"cloud_cover"`
	WindSpeed10m      []float64 `json:"wind_speed_10m"`
	WindDirection10m  []float64 `json:"wind_direction_10m"`
	WindGusts10m      []float64 `json:"wind_gusts_10m"`
}

// SiteResult holds the result of fetching a single site.
type SiteResult struct {
	Site  config.Site
	Data  *OpenMeteoResponse
	Error error
}

// FetchSite fetches the forecast for a single site from Open-Meteo,
// using the provided cache if non-nil. Config values are used from cfg.
func FetchSite(site config.Site, cache *Cache, cfg config.ServerConfig) (*OpenMeteoResponse, error) {
	if cache != nil {
		if data, ok := cache.Get(site); ok {
			return data, nil
		}
	}

	data, err := fetchFromAPI(site, cfg)
	if err == nil && cache != nil {
		cache.Set(site, data)
	}
	return data, err
}

func fetchFromAPI(site config.Site, cfg config.ServerConfig) (*OpenMeteoResponse, error) {
	params := fmt.Sprintf(
		"latitude=%.6f&longitude=%.6f&models=ecmwf_ifs&hourly=is_day,precipitation_probability,temperature_2m,cloud_cover,wind_speed_10m,wind_direction_10m,wind_gusts_10m&timezone=%s&past_days=0&forecast_days=%d",
		site.Lat, site.Lon, cfg.Timezone, cfg.ForecastDays,
	)
	url := cfg.OpenMeteoURL + "?" + params

	client := &http.Client{Timeout: cfg.HTTPTimeout}
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

// FetchAll concurrently fetches forecasts for all sites using the provided cache.
// Errors are isolated per site; a failed site does not cancel others.
func FetchAll(sites []config.Site, cache *Cache, cfg config.ServerConfig) []SiteResult {
	results := make([]SiteResult, len(sites))
	var wg sync.WaitGroup

	for i, site := range sites {
		wg.Add(1)
		go func(i int, site config.Site) {
			defer wg.Done()
			data, err := FetchSite(site, cache, cfg)
			results[i] = SiteResult{Site: site, Data: data, Error: err}
		}(i, site)
	}

	wg.Wait()
	return results
}
