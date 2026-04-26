package config

import (
	"encoding/json"
	"testing"
)

func TestSitesNotEmpty(t *testing.T) {
	if len(Sites) == 0 {
		t.Fatal("Sites should not be empty")
	}
}

func TestSiteCount(t *testing.T) {
	if len(Sites) != 8 {
		t.Errorf("want 8 sites, got %d", len(Sites))
	}
}

func TestAllSitesHaveName(t *testing.T) {
	for i, s := range Sites {
		if s.Name == "" {
			t.Errorf("Site[%d] has empty Name", i)
		}
	}
}

func TestAllSitesHaveTwoDirections(t *testing.T) {
	for i, s := range Sites {
		if len(s.Direction) != 2 {
			t.Errorf("Site[%d] (%s) Direction len=%d, want 2", i, s.Name, len(s.Direction))
		}
	}
}

func TestAllSitesHaveValidCoords(t *testing.T) {
	for i, s := range Sites {
		if s.Lat < -90 || s.Lat > 90 {
			t.Errorf("Site[%d] (%s) has invalid Lat: %f", i, s.Name, s.Lat)
		}
		if s.Lon < -180 || s.Lon > 180 {
			t.Errorf("Site[%d] (%s) has invalid Lon: %f", i, s.Name, s.Lon)
		}
	}
}

func TestSiteNamesUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, s := range Sites {
		if seen[s.Name] {
			t.Errorf("Duplicate site name: %q", s.Name)
		}
		seen[s.Name] = true
	}
}

func TestMaxGustsConstant(t *testing.T) {
	if MaxGusts != 25 {
		t.Errorf("MaxGusts = %d, want 25", MaxGusts)
	}
}

func TestSiteJSON(t *testing.T) {
	site := Site{Name: "Test", Direction: [2]string{"N", "NE"}, Lat: 64.0, Lon: 20.0}
	data, err := json.Marshal(site)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var got Site
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if got.Name != site.Name || got.Lat != site.Lat || got.Lon != site.Lon {
		t.Errorf("round-trip: got %+v, want %+v", got, site)
	}
}

func TestSitesExpectedSiteNames(t *testing.T) {
	expected := []string{
		"Balberget Ramp", "Balberget Stuga", "Tavelsjö",
		"Storuman", "Dalsberget", "Dundret",
		"Kittelfjäll", "Klutmarksbacken",
	}
	if len(Sites) != len(expected) {
		t.Fatalf("Site count mismatch: got %d", len(Sites))
	}
	for i, name := range expected {
		if Sites[i].Name != name {
			t.Errorf("Sites[%d].Name = %q, want %q", i, Sites[i].Name, name)
		}
	}
}
