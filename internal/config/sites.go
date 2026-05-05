package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Site represents a paragliding launch site.
type Site struct {
	Name      string
	Direction [2]string // [from, to] compass point names, e.g. ["SSW", "WSW"]
	Lat       float64
	Lon       float64
}

// ModelName is the human-readable name of the weather model in use.
const ModelName = "ECMWF IFS HRES 9km"

// MaxGusts is the maximum safe wind gust speed in km/h.
// Exposed so other packages can reference it if needed.
const MaxGusts = 25 // km/h

// validCompass is the set of valid 16-point compass names.
var validCompass = map[string]bool{
	"N": true, "NNE": true, "NE": true, "ENE": true,
	"E": true, "ESE": true, "SE": true, "SSE": true,
	"S": true, "SSW": true, "SW": true, "WSW": true,
	"W": true, "WNW": true, "NW": true, "NNW": true,
}

// LoadSites reads, parses, and validates a JSON sites file.
func LoadSites(path string) ([]Site, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading sites file %q: %w", path, err)
	}

	var sites []Site
	if err := json.Unmarshal(data, &sites); err != nil {
		return nil, fmt.Errorf("parsing sites file %q: %w", path, err)
	}

	if err := validateSites(sites); err != nil {
		return nil, fmt.Errorf("invalid sites in %q: %w", path, err)
	}

	return sites, nil
}

// validateSites checks all validation rules for a slice of sites.
func validateSites(sites []Site) error {
	if len(sites) == 0 {
		return fmt.Errorf("at least one site is required")
	}

	names := make(map[string]bool, len(sites))
	for i, s := range sites {
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("site[%d]: name must not be empty", i)
		}
		if names[s.Name] {
			return fmt.Errorf("duplicate site name %q", s.Name)
		}
		names[s.Name] = true

		if s.Lat < -90 || s.Lat > 90 {
			return fmt.Errorf("site %q: lat %.6f is out of range [-90, 90]", s.Name, s.Lat)
		}
		if s.Lon < -180 || s.Lon > 180 {
			return fmt.Errorf("site %q: lon %.6f is out of range [-180, 180]", s.Name, s.Lon)
		}

		if len(s.Direction) != 2 {
			return fmt.Errorf("site %q: direction must have exactly 2 compass points, got %d", s.Name, len(s.Direction))
		}
		for _, d := range s.Direction {
			if !validCompass[d] {
				return fmt.Errorf("site %q: invalid compass direction %q; valid names: N, NNE, NE, ENE, E, ESE, SE, SSE, S, SSW, SW, WSW, W, WNW, NW, NNW", s.Name, d)
			}
		}
	}

	return nil
}
