package forecast

import (
	"sync"
	"time"

	"forecaster/internal/config"
)

const cacheTTL = 24 * time.Hour

type entry struct {
	data      *OpenMeteoResponse
	expiresAt time.Time
}

// Cache is a thread-safe in-memory cache for Open-Meteo responses keyed by site.
type Cache struct {
	mu    sync.RWMutex
	items map[string]*entry
}

// NewCache creates an empty cache.
func NewCache() *Cache {
	return &Cache{items: make(map[string]*entry)}
}

// siteKey returns a deterministic cache key from site coordinates.
func siteKey(site config.Site) string {
	return cacheKey(site.Lat, site.Lon)
}

// cacheKey returns a deterministic cache key from lat/lon.
func cacheKey(lat, lon float64) string {
	// Round to 6 decimal places (~11cm precision) for a stable key.
	return cacheKeyFromMicroDeg(int64(lat*1e6), int64(lon*1e6))
}

// cacheKeyFromMicroDeg creates a stable key from microdegree integers.
func cacheKeyFromMicroDeg(mlat, mlon int64) string {
	return formatInt(mlat) + "," + formatInt(mlon)
}

// formatInt converts an int64 to a string.
func formatInt(n int64) string {
	if n < 0 {
		return "-" + uformat(uint64(-n))
	}
	return uformat(uint64(n))
}

// uformat converts a uint64 to a decimal string.
func uformat(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// Get returns cached data for the site and whether it was found and not expired.
func (c *Cache) Get(site config.Site) (*OpenMeteoResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.items[siteKey(site)]
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.data, true
}

// Set stores data in the cache for the site.
func (c *Cache) Set(site config.Site, data *OpenMeteoResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[siteKey(site)] = &entry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}
