package forecast

import (
	"errors"
	"reflect"
	"testing"

	"forecaster/internal/config"
)

// ── ProcessSites basic ─────────────────────────────────────────────────────────

func TestProcessSitesEmpty(t *testing.T) {
	result := ProcessSites([]SiteResult{})
	if len(result) != 0 {
		t.Errorf("empty input → empty output, got len=%d", len(result))
	}
}

func TestProcessSitesSuccess(t *testing.T) {
	results := []SiteResult{
		makeSiteResult("Site A", 3, nil),
	}
	out := ProcessSites(results)

	if len(out) != 1 {
		t.Fatalf("len=%d, want 1", len(out))
	}
	if out[0].Name != "Site A" {
		t.Errorf("Name=%q, want %q", out[0].Name, "Site A")
	}
	if out[0].Error != nil {
		t.Errorf("Error should be nil for success, got %v", *out[0].Error)
	}
	if len(out[0].Hours) != 3 {
		t.Errorf("Hours len=%d, want 3", len(out[0].Hours))
	}
}

func TestProcessSitesError(t *testing.T) {
	results := []SiteResult{
		{Site: makeTestSite("Bad Site"), Error: errors.New("network timeout")},
	}
	out := ProcessSites(results)

	if len(out) != 1 {
		t.Fatalf("len=%d, want 1", len(out))
	}
	if out[0].Error == nil {
		t.Fatal("Error should be set for failed fetch")
	}
	if *out[0].Error != "network timeout" {
		t.Errorf("Error=%q, want %q", *out[0].Error, "network timeout")
	}
	if len(out[0].Hours) != 0 {
		t.Errorf("Hours should be empty on error, got %d", len(out[0].Hours))
	}
}

func TestProcessSitesDirectionPreserved(t *testing.T) {
	results := []SiteResult{
		{
			Site: config.Site{Name: "X", Direction: [2]string{"ENE", "E"}, Lat: 1, Lon: 2},
			Data: &OpenMeteoResponse{
				Hourly: HourlyBlock{
					Time:              []string{"2026-04-26T10:00"},
					IsDay:             []int{1},
					PrecipitationProb: []float64{0},
					Temperature2m:     []float64{10},
					CloudCover:        []float64{0},
					WindSpeed10m:      []float64{0},
					WindDirection10m:  []float64{0},
					WindGusts10m:      []float64{0},
				},
			},
		},
	}
	out := ProcessSites(results)

	if out[0].Direction[0] != "ENE" || out[0].Direction[1] != "E" {
		t.Errorf("Direction=%v, want [ENE E]", out[0].Direction)
	}
}

// ── ProcessSites data mapping ──────────────────────────────────────────────────

func TestProcessSitesHourlyMapping(t *testing.T) {
	results := []SiteResult{
		{
			Site: makeTestSite("Test"),
			Data: &OpenMeteoResponse{
				Hourly: HourlyBlock{
					Time:              []string{"2026-04-26T10:00", "2026-04-26T11:00", "2026-04-26T12:00"},
					IsDay:             []int{1, 0, 1},
					PrecipitationProb: []float64{5, 20, 30},
					Temperature2m:     []float64{18.0, 12.5, 21.0},
					CloudCover:        []float64{20, 80, 60},
					WindSpeed10m:      []float64{10, 25, 8},
					WindDirection10m:  []float64{180, 270, 200},
					WindGusts10m:      []float64{12, 30, 10},
				},
			},
		},
	}
	out := ProcessSites(results)

	if len(out) != 1 || len(out[0].Hours) != 3 {
		t.Fatal("wrong output size")
	}

	h := out[0].Hours
	if h[0].Time != "2026-04-26T10:00" || h[0].IsDay != 1 {
		t.Errorf("hour 0: %+v", h[0])
	}
	if h[1].Time != "2026-04-26T11:00" || h[1].IsDay != 0 {
		t.Errorf("hour 1: %+v", h[1])
	}
	if h[2].Time != "2026-04-26T12:00" {
		t.Errorf("hour 2: %+v", h[2])
	}

	// Check field mapping
	if h[2].WindSpeed != 8 || h[2].Temp != 21.0 || h[2].Rain != 30 {
		t.Errorf("hour 2: wind_speed=%v temp=%v rain=%v, want 8 21.0 30",
			h[2].WindSpeed, h[2].Temp, h[2].Rain)
	}
}

func TestProcessSitesMixedSuccessAndError(t *testing.T) {
	results := []SiteResult{
		makeSiteResult("OK", 2, nil),
		{Site: makeTestSite("Fail"), Error: errors.New("oops")},
		makeSiteResult("Also OK", 1, nil),
	}
	out := ProcessSites(results)

	if len(out) != 3 {
		t.Fatalf("len=%d, want 3", len(out))
	}
	if out[0].Error != nil {
		t.Errorf("out[0] should be ok")
	}
	if out[1].Error == nil {
		t.Errorf("out[1] should have error")
	}
	if out[2].Error != nil {
		t.Errorf("out[2] should be ok")
	}
	if len(out[0].Hours) != 2 {
		t.Errorf("out[0] hours len=%d, want 2", len(out[0].Hours))
	}
	if len(out[1].Hours) != 0 {
		t.Errorf("out[1] hours len=%d, want 0", len(out[1].Hours))
	}
	if len(out[2].Hours) != 1 {
		t.Errorf("out[2] hours len=%d, want 1", len(out[2].Hours))
	}
}

// ── ProcessSites output types ───────────────────────────────────────────────────

func TestSiteDataFields(t *testing.T) {
	results := []SiteResult{makeSiteResult("Test", 1, nil)}
	out := ProcessSites(results)

	sd := out[0]
	if sd.Name != "Test" {
		t.Errorf("Name=%q", sd.Name)
	}
	if sd.Direction[0] != "SSW" {
		t.Errorf("Direction[0]=%q", sd.Direction[0])
	}
	if sd.Hours == nil {
		t.Error("Hours should not be nil")
	}
	if sd.Error != nil {
		t.Error("Error should be nil")
	}
}

func TestHourDataFields(t *testing.T) {
	results := []SiteResult{
		{
			Site: makeTestSite("Test"),
			Data: &OpenMeteoResponse{
				Hourly: HourlyBlock{
					Time:              []string{"2026-04-26T00:00"},
					IsDay:             []int{1},
					PrecipitationProb: []float64{99.5},
					Temperature2m:     []float64{-5.5},
					CloudCover:        []float64{100},
					WindSpeed10m:      []float64{50},
					WindDirection10m:  []float64{359},
					WindGusts10m:      []float64{60},
				},
			},
		},
	}
	out := ProcessSites(results)

	h := out[0].Hours[0]
	if h.Rain != 99.5 {
		t.Errorf("Rain=%v, want 99.5", h.Rain)
	}
	if h.Temp != -5.5 {
		t.Errorf("Temp=%v, want -5.5", h.Temp)
	}
	if h.Cloud != 100 {
		t.Errorf("Cloud=%v, want 100", h.Cloud)
	}
	if h.WindSpeed != 50 {
		t.Errorf("WindSpeed=%v, want 50", h.WindSpeed)
	}
	if h.WindDir != 359 {
		t.Errorf("WindDir=%v, want 359", h.WindDir)
	}
	if h.Gusts != 60 {
		t.Errorf("Gusts=%v, want 60", h.Gusts)
	}
}

// ── ProcessSites preserves order ─────────────────────────────────────────────

func TestProcessSitesPreservesOrder(t *testing.T) {
	results := []SiteResult{
		makeSiteResult("First", 1, nil),
		makeSiteResult("Second", 1, nil),
		makeSiteResult("Third", 1, nil),
	}
	out := ProcessSites(results)

	if out[0].Name != "First" || out[1].Name != "Second" || out[2].Name != "Third" {
		t.Errorf("Order not preserved: %v", []string{out[0].Name, out[1].Name, out[2].Name})
	}
}

// ── ProcessSites all 8 sites ────────────────────────────────────────────────────

func TestProcessSitesAll8Sites(t *testing.T) {
	results := make([]SiteResult, 8)
	for i := 0; i < 8; i++ {
		results[i] = makeSiteResult("Site", 24, nil)
	}
	out := ProcessSites(results)
	if len(out) != 8 {
		t.Errorf("len=%d, want 8", len(out))
	}
}

// ── Struct integrity ────────────────────────────────────────────────────────────

func TestSiteResultStruct(t *testing.T) {
	site := makeTestSite("t")
	result := SiteResult{Site: site, Data: nil, Error: nil}
	if result.Site.Name != "t" {
		t.Error("Site field not set")
	}
}

func TestOpenMeteoResponseStruct(t *testing.T) {
	resp := OpenMeteoResponse{
		Hourly: HourlyBlock{
			Time:              []string{"t"},
			IsDay:             []int{1},
			PrecipitationProb: []float64{0},
			Temperature2m:     []float64{0},
			CloudCover:        []float64{0},
			WindSpeed10m:      []float64{0},
			WindDirection10m:  []float64{0},
			WindGusts10m:      []float64{0},
		},
	}
	if len(resp.Hourly.Time) != 1 {
		t.Error("Hourly block not initialized correctly")
	}
}

func TestProcessSitesNoAliasing(t *testing.T) {
	results := []SiteResult{makeSiteResult("Orig", 1, nil)}
	out := ProcessSites(results)

	out[0].Name = "Mutated"
	if results[0].Site.Name == "Mutated" {
		t.Error("ProcessSites should not alias Site back to SiteResult")
	}
}

func TestSiteDataSize(t *testing.T) {
	out := ProcessSites([]SiteResult{makeSiteResult("t", 1, nil)})
	sd := &out[0]
	val := reflect.ValueOf(*sd)
	for i := 0; i < val.NumField(); i++ {
		_ = val.Field(i).Interface()
	}
}
