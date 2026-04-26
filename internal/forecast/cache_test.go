package forecast

import (
	"sync"
	"testing"
	"time"

	"forecaster/internal/config"
)

func makeTestResponse(n int) *OpenMeteoResponse {
	return &OpenMeteoResponse{
		Hourly: HourlyBlock{
			Time:              repeatedStrings("2026-04-26T00:00", n),
			IsDay:             repeatedInts(1, n),
			PrecipitationProb: repeatedFloats(10, n),
			Temperature2m:     repeatedFloats(15.0, n),
			CloudCover:        repeatedFloats(30, n),
			WindSpeed10m:      repeatedFloats(12, n),
			WindDirection10m:  repeatedFloats(200, n),
			WindGusts10m:      repeatedFloats(15, n),
		},
	}
}

func repeatedStrings(s string, n int) []string {
	r := make([]string, n)
	for i := range r {
		r[i] = s
	}
	return r
}

func repeatedInts(v int, n int) []int {
	r := make([]int, n)
	for i := range r {
		r[i] = v
	}
	return r
}

func repeatedFloats(v float64, n int) []float64 {
	r := make([]float64, n)
	for i := range r {
		r[i] = v
	}
	return r
}

// ── Cache key functions ─────────────────────────────────────────────────────────

func TestCacheKeyFromMicroDeg(t *testing.T) {
	tests := []struct {
		mlat, mlon int64
		want       string
	}{
		{0, 0, "0,0"},
		{1, 1, "1,1"},
		{-1, -1, "-1,-1"},
		{63943440, 19046277, "63943440,19046277"},
	}
	for _, tt := range tests {
		got := cacheKeyFromMicroDeg(tt.mlat, tt.mlon)
		if got != tt.want {
			t.Errorf("cacheKeyFromMicroDeg(%d,%d) = %q, want %q", tt.mlat, tt.mlon, got, tt.want)
		}
	}
}

func TestCacheKeyStable(t *testing.T) {
	lat, lon := 63.94344038093285, 19.046277812311036
	k1 := cacheKey(lat, lon)
	k2 := cacheKey(lat, lon)
	if k1 != k2 {
		t.Errorf("cacheKey not stable: %q != %q", k1, k2)
	}
}

func TestCacheKeyDifferentForDifferentCoords(t *testing.T) {
	k1 := cacheKey(63.943, 19.046)
	k2 := cacheKey(63.944, 19.046)
	if k1 == k2 {
		t.Errorf("Different coordinates should produce different keys: %q == %q", k1, k2)
	}
}

func TestSiteKey(t *testing.T) {
	s1 := makeTestSite("a")
	s2 := makeTestSite("b")
	// Name doesn't affect key — only coords
	if siteKey(s1) != siteKey(s2) {
		t.Error("same coords should produce same key regardless of name")
	}

	s3 := makeTestSiteWithCoords("c", 63.001, 19.0)
	if siteKey(s1) == siteKey(s3) {
		t.Error("different coords should produce different key")
	}
}

// makeTestSiteWithCoords creates a Site with explicit lat/lon for key-testing.
func makeTestSiteWithCoords(name string, lat, lon float64) config.Site {
	return config.Site{Name: name, Direction: [2]string{"N", "NE"}, Lat: lat, Lon: lon}
}

// ── Cache Get/Set ───────────────────────────────────────────────────────────────

func TestCacheGetEmpty(t *testing.T) {
	c := NewCache()
	_, ok := c.Get(makeTestSite("test"))
	if ok {
		t.Error("Get on empty cache should return false")
	}
}

func TestCacheSetThenGet(t *testing.T) {
	c := NewCache()
	data := makeTestResponse(24)

	c.Set(makeTestSite("test"), data)
	got, ok := c.Get(makeTestSite("test"))
	if !ok {
		t.Fatal("Get after Set should return true")
	}
	if got != data {
		t.Error("Get returned different data than Set")
	}
}

func TestCacheSetSameSiteTwice(t *testing.T) {
	c := NewCache()
	d1 := makeTestResponse(12)
	d2 := makeTestResponse(24)

	c.Set(makeTestSite("test"), d1)
	c.Set(makeTestSite("test"), d2) // overwrite

	got, ok := c.Get(makeTestSite("test"))
	if !ok {
		t.Fatal("Get should succeed after overwrite")
	}
	if got != d2 {
		t.Error("Get should return the last Set value")
	}
}

func TestCacheSeparateSites(t *testing.T) {
	c := NewCache()
	d1 := makeTestResponse(10)
	d2 := makeTestResponse(20)

	c.Set(makeTestSiteWithCoords("a", 63.0, 19.0), d1)
	c.Set(makeTestSiteWithCoords("b", 64.0, 20.0), d2)

	g1, ok1 := c.Get(makeTestSiteWithCoords("a", 63.0, 19.0))
	g2, ok2 := c.Get(makeTestSiteWithCoords("b", 64.0, 20.0))

	if !ok1 || !ok2 {
		t.Fatal("both sites should be retrievable")
	}
	if g1 != d1 || g2 != d2 {
		t.Error("sites returned wrong data")
	}
}

func TestCacheGetNonExistent(t *testing.T) {
	c := NewCache()
	c.Set(makeTestSiteWithCoords("a", 63.0, 19.0), makeTestResponse(10))

	_, ok := c.Get(makeTestSiteWithCoords("b", 64.0, 20.0)) // never Set
	if ok {
		t.Error("Get for never-set site should return false")
	}
}

func TestCacheTTLConstant(t *testing.T) {
	// Verify the TTL constant is set to 24 hours
	if cacheTTL != 24*time.Hour {
		t.Errorf("cacheTTL = %v, want 24h", cacheTTL)
	}
}

// ── Concurrent access ──────────────────────────────────────────────────────────

func TestCacheConcurrentGetSet(t *testing.T) {
	c := NewCache()
	data := makeTestResponse(24)
	site := makeTestSite("concurrent")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.Set(site, data)
		}()
		go func() {
			defer wg.Done()
			c.Get(site)
		}()
	}
	wg.Wait()
}

func TestCacheConcurrentSetMultipleSites(t *testing.T) {
	c := NewCache()
	var wg sync.WaitGroup

	// Create unique sites and store them so we can retrieve them below
	sites := make([]config.Site, 50)
	for i := 0; i < 50; i++ {
		site := makeTestSiteWithCoords("site", float64(i), float64(i))
		sites[i] = site
		wg.Add(1)
		go func(s config.Site) {
			defer wg.Done()
			c.Set(s, makeTestResponse(10))
		}(site)
	}
	wg.Wait()

	// Verify all 50 can be retrieved
	for i, site := range sites {
		_, ok := c.Get(site)
		if !ok {
			t.Errorf("site[%d] not retrievable after concurrent Set", i)
		}
	}
}
