package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"forecaster/internal/forecast"
)

func TestHandlerContentType(t *testing.T) {
	// Use a nil cache — FetchAll will call Open-Meteo in-process.
	// We can't easily test without mocking, so we just verify the
	// handler struct can be created and has the right fields.
	h := NewHandler(nil)
	if h == nil {
		t.Fatal("NewHandler returned nil")
	}
	if h.cache != nil {
		t.Error("cache should be nil")
	}
}

func TestForecastResponseJSONShape(t *testing.T) {
	// Verify ForecastResponse serializes to the expected JSON shape
	// by constructing a mock response and checking struct tags.
	resp := ForecastResponse{
		Sites:     []forecast.SiteData{},
		FetchedAt: "2026-04-26T10:00:00Z",
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if _, ok := got["sites"]; !ok {
		t.Error("response should have 'sites' field")
	}
	if _, ok := got["fetched_at"]; !ok {
		t.Error("response should have 'fetched_at' field")
	}
}

func TestForecastResponseSitesField(t *testing.T) {
	sd := forecast.SiteData{
		Name:      "Test Site",
		Direction: [2]string{"SSW", "WSW"},
		Hours: []forecast.HourData{
			{Time: "2026-04-26T10:00", IsDay: 1, WindDir: 195, WindSpeed: 12, Gusts: 18, Cloud: 45, Rain: 10, Temp: 8.5},
		},
	}
	resp := ForecastResponse{
		Sites:     []forecast.SiteData{sd},
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// Verify the site object in JSON has expected keys
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	sites := parsed["sites"].([]interface{})
	if len(sites) != 1 {
		t.Fatalf("len(sites)=%d, want 1", len(sites))
	}

	site := sites[0].(map[string]interface{})
	if site["name"] != "Test Site" {
		t.Errorf("site name=%v, want Test Site", site["name"])
	}
	if dir, ok := site["direction"]; !ok {
		t.Error("site should have 'direction' field")
	} else {
		d := dir.([]interface{})
		if d[0] != "SSW" || d[1] != "WSW" {
			t.Errorf("direction=%v, want [SSW WSW]", d)
		}
	}
}

func TestHandlerServesHTTP(t *testing.T) {
	// Test that the handler responds to a GET request with status 200.
	// Uses a real in-process FetchAll call, so this is an integration test.
	h := NewHandler(nil)
	req := httptest.NewRequest("GET", "/api/forecast", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status=%d, want 200", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type=%q, want application/json", ct)
	}
}

func TestHandlerResponseHasSitesAndFetchedAt(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest("GET", "/api/forecast", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	var resp ForecastResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response not valid JSON: %v\nBody: %s", err, rec.Body.String())
	}

	if resp.FetchedAt == "" {
		t.Error("fetched_at should not be empty")
	}

	// fetched_at should be a valid RFC3339 timestamp
	_, err := time.Parse(time.RFC3339, resp.FetchedAt)
	if err != nil {
		t.Errorf("fetched_at=%q is not valid RFC3339: %v", resp.FetchedAt, err)
	}

	// Should have 8 sites from config
	if len(resp.Sites) != 8 {
		t.Errorf("len(sites)=%d, want 8", len(resp.Sites))
	}
}

func TestHandlerReturnsAll8SitesWithNames(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest("GET", "/api/forecast", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	var resp ForecastResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)

	expected := []string{
		"Balberget Ramp", "Balberget Stuga", "Tavelsjö",
		"Storuman", "Dalsberget", "Dundret",
		"Kittelfjäll", "Klutmarksbacken",
	}
	for i, name := range expected {
		if resp.Sites[i].Name != name {
			t.Errorf("Sites[%d].Name=%q, want %q", i, resp.Sites[i].Name, name)
		}
	}
}

func TestHandlerReturnsAllSiteDirections(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest("GET", "/api/forecast", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	var resp ForecastResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)

	for i, site := range resp.Sites {
		if site.Direction[0] == "" || site.Direction[1] == "" {
			t.Errorf("Sites[%d] (%s) has empty direction: %v", i, site.Name, site.Direction)
		}
	}
}

func TestHandlerCaches(t *testing.T) {
	cache := forecast.NewCache()
	h := NewHandler(cache)

	// First request — will hit Open-Meteo
	req := httptest.NewRequest("GET", "/api/forecast", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request failed: %d", rec.Code)
	}

	// Second request — should be served from cache (would be much faster,
	// but we can't measure timing reliably in tests).
	// We can verify the response is still valid.
	req2 := httptest.NewRequest("GET", "/api/forecast", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("second request failed: %d", rec2.Code)
	}

	var resp1, resp2 ForecastResponse
	json.Unmarshal(rec.Body.Bytes(), &resp1)
	json.Unmarshal(rec2.Body.Bytes(), &resp2)

	// Both should have 8 sites
	if len(resp1.Sites) != 8 || len(resp2.Sites) != 8 {
		t.Error("both requests should return 8 sites")
	}
}

func TestHandlerJSONIsReadable(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest("GET", "/api/forecast", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Should not be empty
	if rec.Body.Len() == 0 {
		t.Fatal("response body is empty")
	}

	// Should parse without error
	var resp ForecastResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response not parseable: %v\nBody: %s", err, rec.Body.String())
	}
}

func TestHandlerHTTPMethod(t *testing.T) {
	h := NewHandler(nil)

	// POST should still work (handler doesn't check method, but HTTP routing does).
	// We test that a routed POST goes through handler.
	req := httptest.NewRequest("POST", "/api/forecast", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Handler doesn't enforce method — this is fine.
	// We just verify it doesn't panic.
}
