package forecast

// SiteData is the processed output for a single site returned by the API.
type SiteData struct {
	Name      string          `json:"name"`
	Direction [2]string      `json:"direction"`
	Hours     []HourData     `json:"hours"`
	Error     *string        `json:"error"` // null on success
}

// HourData is a single hourly row exposed to the frontend.
type HourData struct {
	Time      string  `json:"time"`
	IsDay     int     `json:"is_day"`
	WindDir   float64 `json:"wind_dir"`
	WindSpeed float64 `json:"wind_speed"`
	Gusts     float64 `json:"gusts"`
	Cloud     float64 `json:"cloud"`
	Rain      float64 `json:"rain"`
	Temp      float64 `json:"temp"`
}

// ProcessSites converts raw Open-Meteo SiteResult entries into SiteData
// structs ready for the HTTP response. Errors are surfaced per-site.
func ProcessSites(results []SiteResult) []SiteData {
	out := make([]SiteData, len(results))
	for i, r := range results {
		if r.Error != nil {
			errStr := r.Error.Error()
			out[i] = SiteData{Name: r.Site.Name, Direction: r.Site.Direction, Error: &errStr}
			continue
		}
		h := r.Data.Hourly
		hours := make([]HourData, len(h.Time))
		for j := range h.Time {
			hours[j] = HourData{
				Time:      h.Time[j],
				IsDay:     h.IsDay[j],
				WindDir:   h.WindDirection10m[j],
				WindSpeed: h.WindSpeed10m[j],
				Gusts:     h.WindGusts10m[j],
				Cloud:     h.CloudCover[j],
				Rain:      h.PrecipitationProb[j],
				Temp:      h.Temperature2m[j],
			}
		}
		out[i] = SiteData{Name: r.Site.Name, Direction: r.Site.Direction, Hours: hours}
	}
	return out
}