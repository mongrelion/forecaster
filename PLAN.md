# Plan: Move API Logic to Go Backend

## Goal
Move Open-Meteo API interaction from the browser to a Go backend. The backend fetches forecasts, caches responses, and returns raw hourly data + site metadata (direction ranges). The frontend computes all flyability flags and handles user-configurable thresholds (cloud, rain) and all rendering.

---

## Stage 1 — Go Project Setup

Initialize the Go module and establish project structure.

### Tasks
- [x] Initialize Go module (`go mod init`)
- [x] Create `cmd/server/main.go` entry point with HTTP server
- [x] Set up `internal/` package structure:
  ```
  internal/
    config/    — sites list, constants
    forecast/  — Open-Meteo client, caching, processing
    api/       — HTTP handlers
  ```
- [x] Add `.gitignore` for Go binaries
- [x] Add `Makefile` with targets for building `make build`, testing `make test` and running `make run` the project

---

## Stage 2 — Sites Configuration

Move the sites list from `app.js` to a Go config package.

### Tasks
- [x] Create `internal/config/sites.go` with the 8 sites
- [x] Define `Site` struct: `Name`, `Direction [2]string`, `Lat`, `Lon`
- [x] Export `Sites` slice and `MaxGusts` constant (25 km/h)
- [x] Remove `SITES` array from `app.js`

---

## Stage 3 — Open-Meteo Client

Build the API client that fetches forecasts for all sites concurrently.

### Tasks
- [x] Create `internal/forecast/client.go`
- [x] Define response structs matching Open-Meteo JSON shape
- [x] Implement `FetchSite(site) (HourlyData, error)` — single site fetch
- [x] Implement `FetchAll(sites) []SiteResult` — concurrent fetch with `sync.WaitGroup` + error isolation
- [x] Always request 7 forecast days with `timezone=Europe/Stockholm`
- [x] Request fields: `is_day, precipitation_probability, temperature_2m, cloud_cover, wind_speed_10m, wind_direction_10m, wind_gusts_10m`

---

## Stage 4 — In-Process Cache

Cache Open-Meteo responses in memory with 24h TTL.

### Tasks
- [x] Create `internal/forecast/cache.go`
- [x] Implement thread-safe cache (sync.RWMutex + map)
- [x] Cache key: site lat+lon
- [x] TTL: 24 hours
- [x] `Get(site) (data, hit)`
- [x] `Set(site, data)`
- [x] Wire into `FetchSite`: check cache before API call, store on miss

---

## Stage 5 — Site Metadata

Return site direction ranges so the frontend can compute all flyability flags.

### Tasks
- [x] Create `internal/forecast/processor.go`
- [x] Define output structs for the API response (site name, direction ranges, hours)
- [x] No compass logic needed in Go — the frontend already has it
- [x] Direction ranges are passed through from config, not computed

---

## Stage 6 — API Handler

Expose a single endpoint that returns all sites with processed data.

### Tasks
- [x] Create `internal/api/handler.go`
- [x] `GET /api/forecast` — returns JSON:
  ```json
  {
    "sites": [
      {
        "name": "Balberget Ramp",
        "direction": ["SSW", "WSW"],
        "hours": [
          {
            "time": "2026-04-26T00:00",
            "is_day": 1,
            "wind_dir": 195,
            "wind_speed": 12,
            "gusts": 18,
            "cloud": 45,
            "rain": 10,
            "temp": 8.5
          }
        ],
        "error": null
      }
    ],
    "fetched_at": "2026-04-26T10:00:00Z"
  }
  ```
- [x] Per-site errors returned in `error` field (null on success)
- [x] Register handler in `cmd/server/main.go`

---

## Stage 7 — Static File Serving

Serve the frontend from the same Go binary.

### Tasks
- [x] Serve `public/` directory at `/` using `http.FileServer`
- [x] Ensure `/api/*` routes are handled before the file server catch-all
- [x] Verify `index.html`, `app.css`, `app.js` are served correctly

---

## Stage 8 — Frontend Refactor

Update `app.js` to consume the backend API instead of calling Open-Meteo directly.

### Tasks
- [x] Remove `API_BASE`, `MAX_FORECAST_DAYS`, `buildUrl()`, `fetchSite()`, `fetchAll()`
- [x] Remove all localStorage cache logic (`CACHE_PREFIX`, `siteCacheKey`, `getFromCache`, `setInCache`)
- [x] Remove `MAX_GUSTS` constant (keep it only in frontend)
- [x] Keep `SITES` direction ranges only for reference — actual site data comes from backend
- [x] Replace `loadData()`:
  - Fetch `GET /api/forecast`
  - Store response in `rawResponses`
- [x] Update `processResponse()` to read site direction from backend response and compute all flags client-side: `dir_ok`, `gusts_ok`, `cloud_ok`, `rain_ok`
- [x] Keep threshold logic computed client-side from raw values
- [x] Keep `findWindows()`, `findBestBet()`, `renderAll()` unchanged
- [x] Keep threshold controls working (re-process without network call)
- [x] Remove "Clear cache" button (no longer relevant)
- [x] Update footer or add note about backend

---

## Stage 9 — Wire Up & Verify

Connect everything and test end-to-end.

### Tasks
- [x] `cmd/server/main.go`: wire cache → client → processor → handler
- [x] Start server, open browser, verify...
- [x] Test error handling: stop backend, verify frontend shows error state
- [x] Test cache: first call hits API, second call serves from cache

---

## File Summary

| File | Action |
|------|--------|
| `go.mod` | Create |
| `cmd/server/main.go` | Create |
| `internal/config/sites.go` | Create |
| `internal/forecast/client.go` | Create |
| `internal/forecast/cache.go` | Create |
| `internal/forecast/processor.go` | Create |
| `internal/api/handler.go` | Create |
| `public/app.js` | Refactor (remove API/cache logic, consume backend) |
| `public/index.html` | Minor update (remove cache button) |
| `public/app.css` | No changes expected |
