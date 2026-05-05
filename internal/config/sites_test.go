package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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

// ---------------------------------------------------------------------------
// LoadSites tests
// ---------------------------------------------------------------------------

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sites.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeTempFile: %v", err)
	}
	return path
}

func TestLoadSites_FileNotFound(t *testing.T) {
	_, err := LoadSites("/nonexistent/path/sites.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadSites_InvalidJSON(t *testing.T) {
	path := writeTempFile(t, "this is not json")
	_, err := LoadSites(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoadSites_EmptyArray(t *testing.T) {
	path := writeTempFile(t, "[]")
	_, err := LoadSites(path)
	if err == nil {
		t.Fatal("expected error for empty array, got nil")
	}
}

func TestLoadSites_EmptyName(t *testing.T) {
	data := `[{"name": "", "direction": ["N", "NE"], "lat": 60.0, "lon": 15.0}]`
	path := writeTempFile(t, data)
	_, err := LoadSites(path)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestLoadSites_InvalidDirection(t *testing.T) {
	data := `[{"name": "Test", "direction": ["N", "INVALID"], "lat": 60.0, "lon": 15.0}]`
	path := writeTempFile(t, data)
	_, err := LoadSites(path)
	if err == nil {
		t.Fatal("expected error for invalid direction, got nil")
	}
}

func TestLoadSites_InvalidCoords(t *testing.T) {
	data := `[{"name": "Test", "direction": ["N", "NE"], "lat": 200.0, "lon": 15.0}]`
	path := writeTempFile(t, data)
	_, err := LoadSites(path)
	if err == nil {
		t.Fatal("expected error for invalid lat, got nil")
	}
}

func TestLoadSites_DuplicateName(t *testing.T) {
	data := `[
		{"name": "Same", "direction": ["N", "NE"], "lat": 60.0, "lon": 15.0},
		{"name": "Same", "direction": ["S", "SW"], "lat": 61.0, "lon": 16.0}
	]`
	path := writeTempFile(t, data)
	_, err := LoadSites(path)
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestLoadSites_ValidFile(t *testing.T) {
	data := `[
		{"name": "Site A", "direction": ["N", "NE"], "lat": 60.0, "lon": 15.0},
		{"name": "Site B", "direction": ["S", "SW"], "lat": 61.0, "lon": 16.0}
	]`
	path := writeTempFile(t, data)
	sites, err := LoadSites(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sites) != 2 {
		t.Fatalf("want 2 sites, got %d", len(sites))
	}
	if sites[0].Name != "Site A" || sites[1].Name != "Site B" {
		t.Errorf("unexpected site names: %q, %q", sites[0].Name, sites[1].Name)
	}
	if sites[0].Direction != [2]string{"N", "NE"} {
		t.Errorf("Site A direction = %v, want [N NE]", sites[0].Direction)
	}
}

func TestLoadSites_Success(t *testing.T) {
	// Load the real sites.json from the project root (two levels up from internal/config/).
	path := filepath.Join("..", "..", "sites.json")
	sites, err := LoadSites(path)
	if err != nil {
		t.Fatalf("loading %s: %v", path, err)
	}
	if len(sites) != 8 {
		t.Fatalf("want 8 sites from sites.json, got %d", len(sites))
	}
	expected := []struct {
		name      string
		direction [2]string
		lat       float64
		lon       float64
	}{
		{"Balberget Ramp", [2]string{"SSW", "WSW"}, 63.94344038093285, 19.046277812311036},
		{"Balberget Stuga", [2]string{"ESE", "SSE"}, 63.94013281904288, 19.122045235179684},
		{"Tavelsjö", [2]string{"ENE", "E"}, 64.01453496664449, 20.0159563},
		{"Storuman", [2]string{"SE", "NW"}, 64.96104228812447, 17.69696781869336},
		{"Dalsberget", [2]string{"ENE", "ESE"}, 62.91695106970932, 18.466744719924737},
		{"Dundret", [2]string{"NW", "ENE"}, 67.11411862249734, 20.588067722234612},
		{"Kittelfjäll", [2]string{"ESE", "SSE"}, 65.25436262582429, 15.487933185539914},
		{"Klutmarksbacken", [2]string{"SSW", "WSW"}, 64.72117147014961, 20.782167371833356},
	}
	for i, want := range expected {
		got := sites[i]
		if got.Name != want.name {
			t.Errorf("sites[%d].Name = %q, want %q", i, got.Name, want.name)
		}
		if got.Direction != want.direction {
			t.Errorf("sites[%d].Direction = %v, want %v", i, got.Direction, want.direction)
		}
		if got.Lat != want.lat {
			t.Errorf("sites[%d].Lat = %v, want %v", i, got.Lat, want.lat)
		}
		if got.Lon != want.lon {
			t.Errorf("sites[%d].Lon = %v, want %v", i, got.Lon, want.lon)
		}
	}
}
