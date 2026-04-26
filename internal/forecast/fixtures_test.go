package forecast

import (
	"forecaster/internal/config"
)

// makeTestSite creates a Site with default coords for use in tests.
func makeTestSite(name string) config.Site {
	return config.Site{Name: name, Direction: [2]string{"SSW", "WSW"}, Lat: 63.0, Lon: 19.0}
}

// makeSiteResult creates a SiteResult with a synthetic OpenMeteoResponse
// containing nHours identical hours.
func makeSiteResult(name string, nHours int, err error) SiteResult {
	site := makeTestSite(name)
	if err != nil {
		return SiteResult{Site: site, Error: err}
	}
	return SiteResult{
		Site: site,
		Data: &OpenMeteoResponse{
			Hourly: HourlyBlock{
				Time:              repeatedStrings("2026-04-26T12:00", nHours),
				IsDay:             repeatedInts(1, nHours),
				PrecipitationProb: repeatedFloats(10, nHours),
				Temperature2m:     repeatedFloats(20.5, nHours),
				CloudCover:        repeatedFloats(40, nHours),
				WindSpeed10m:      repeatedFloats(15, nHours),
				WindDirection10m:  repeatedFloats(195, nHours),
				WindGusts10m:      repeatedFloats(18, nHours),
			},
		},
	}
}
