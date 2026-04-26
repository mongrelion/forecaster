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

func TestNextRunCompletion(t *testing.T) {
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "at 04:00 UTC → next is 09:00 UTC same day",
			now:  time.Date(2026, 4, 26, 4, 0, 0, 0, time.UTC),
			want: time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC),
		},
		{
			name: "at 10:00 UTC → next is 15:00 UTC same day",
			now:  time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
			want: time.Date(2026, 4, 26, 15, 0, 0, 0, time.UTC),
		},
		{
			name: "at 22:00 UTC → next is 03:00 UTC next day",
			now:  time.Date(2026, 4, 26, 22, 0, 0, 0, time.UTC),
			want: time.Date(2026, 4, 27, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "at 03:00 UTC (exact completion) → next is 09:00 UTC same day",
			now:  time.Date(2026, 4, 26, 3, 0, 0, 0, time.UTC),
			want: time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC),
		},
		{
			name: "at 00:00 UTC → next is 03:00 UTC same day",
			now:  time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 4, 26, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "at 02:59 UTC → next is 03:00 UTC same day",
			now:  time.Date(2026, 4, 26, 2, 59, 59, 0, time.UTC),
			want: time.Date(2026, 4, 26, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "at 03:01 UTC → next is 09:00 UTC same day",
			now:  time.Date(2026, 4, 26, 3, 1, 0, 0, time.UTC),
			want: time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC),
		},
		{
			name: "at 20:59 UTC → next is 21:00 UTC same day",
			now:  time.Date(2026, 4, 26, 20, 59, 0, 0, time.UTC),
			want: time.Date(2026, 4, 26, 21, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextRunCompletion(tt.now)
			if !got.Equal(tt.want) {
				t.Errorf("nextRunCompletion(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}

func TestNextRunCompletionNonUTCTimezone(t *testing.T) {
	// Input in non-UTC timezone should be converted to UTC internally.
	// Stockholm is UTC+2 in April (CEST).
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}

	// 04:00 CEST = 02:00 UTC → next completion is 03:00 UTC
	now := time.Date(2026, 4, 26, 4, 0, 0, 0, loc)
	want := time.Date(2026, 4, 26, 3, 0, 0, 0, time.UTC)

	got := nextRunCompletion(now)
	if !got.Equal(want) {
		t.Errorf("nextRunCompletion(%v) = %v, want %v", now, got, want)
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
