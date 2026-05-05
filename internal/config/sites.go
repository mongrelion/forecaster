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

// Sites is the list of all configured launch sites.
var Sites = []Site{
	{Name: "Balberget Ramp", Direction: [2]string{"SSW", "WSW"}, Lat: 63.94344038093285, Lon: 19.046277812311036},
	{Name: "Balberget Stuga", Direction: [2]string{"ESE", "SSE"}, Lat: 63.94013281904288, Lon: 19.122045235179684},
	{Name: "Tavelsjö", Direction: [2]string{"ENE", "E"}, Lat: 64.01453496664449, Lon: 20.0159563},
	{Name: "Storuman", Direction: [2]string{"SE", "NW"}, Lat: 64.96104228812447, Lon: 17.69696781869336},
	{Name: "Dalsberget", Direction: [2]string{"ENE", "ESE"}, Lat: 62.91695106970932, Lon: 18.466744719924737},
	{Name: "Dundret", Direction: [2]string{"NW", "ENE"}, Lat: 67.11411862249734, Lon: 20.588067722234612},
	{Name: "Kittelfjäll", Direction: [2]string{"ESE", "SSE"}, Lat: 65.25436262582429, Lon: 15.487933185539914},
	{Name: "Klutmarksbacken", Direction: [2]string{"SSW", "WSW"}, Lat: 64.72117147014961, Lon: 20.782167371833356},
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
