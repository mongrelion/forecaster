# Plan: Switch Forecast Provider to ECMWF IFS HRES

## Goal
Replace Open-Meteo's default "best match" weather model with ECMWF IFS HRES 9km for more accurate forecasts at northern Swedish paragliding sites. The switch is made through Open-Meteo's existing JSON API by adding `&models=ecmwf_ifs` — no GRIB parsing, no new accounts, no API keys. The frontend API contract remains identical (same JSON shape), with a minor addition of a `model` field for display.

---

## Stage 1 — Switch to ECMWF IFS Model in Go Backend

Add the `models=ecmwf_ifs` query parameter to the Open-Meteo API URL and expose a model name constant.

### Tasks
- [x] Add `ModelName` constant (e.g., `"ECMWF IFS HRES 9km"`) to `internal/config/sites.go`
- [x] Update `fetchFromAPI()` in `internal/forecast/client.go` to include `&models=ecmwf_ifs` in the URL parameters
- [x] Verify the response JSON shape is identical to the current one (same hourly keys, same types) by running the existing tests

---

## Stage 2 — Smart Cache Expiration Based on ECMWF Run Schedule

Replace the fixed 24h cache TTL with a model-run-aware expiration. ECMWF IFS produces new forecasts at 00, 06, 12, 18 UTC. Data becomes available approximately 3 hours after each run start (i.e., at 03, 09, 15, 21 UTC). Cache entries should expire at the next model run completion time, so fresh data is fetched as soon as a new run is available.

### Tasks
- [ ] Implement `nextRunCompletion(now time.Time) time.Time` in `internal/forecast/cache.go` — returns the next ECMWF data availability time after `now` (03, 09, 15, or 21 UTC, rolling over to the next day if past 21:00 UTC)
- [ ] Change `Cache.Set()` to set `expiresAt` to `nextRunCompletion(time.Now())` instead of `time.Now().Add(cacheTTL)`
- [ ] Remove the `cacheTTL` constant (no longer needed)
- [ ] Update `Cache.Get()` logic — expired entries already return `false`, no change needed
- [ ] Update cache tests: replace TTL-based tests with run-completion-based tests (verify that entries cached at various times of day expire at the correct next run completion)
- [ ] Add specific test cases for `nextRunCompletion()`:
  - At 04:00 UTC → next completion is 09:00 UTC same day
  - At 10:00 UTC → next completion is 15:00 UTC same day
  - At 22:00 UTC → next completion is 03:00 UTC next day
  - At 03:00 UTC (exact completion time) → next completion is 09:00 UTC (current data just became available, cache until next run)

---

## Stage 3 — Add Model Info to API Response

Add a `model` field to the forecast API response so the frontend can display which model is in use.

### Tasks
- [ ] Add `Model string` field to `ForecastResponse` struct in `internal/api/handler.go` with JSON tag `"model"`
- [ ] Set `Model` to `config.ModelName` when constructing the response in `ServeHTTP()`
- [ ] Update handler tests to verify the `model` field is present and has the expected value

---

## Stage 4 — Frontend Model Display

Update the frontend to show which forecast model is being used.

### Tasks
- [ ] Read `model` from the API response in `loadData()` and store it (e.g., `window._modelName = data.model`)
- [ ] Update the footer in `index.html` to show the model name (e.g., "ECMWF IFS HRES 9km via Open-Meteo" instead of just "Powered by Open-Meteo")
- [ ] Alternatively, render the model name dynamically from the API response into the footer or controls bar
- [ ] No changes to flyability logic, thresholds, rendering pipeline, or sorting — all untouched

---

## Stage 5 — Documentation Updates

Update project documentation to reflect the ECMWF switch.

### Tasks
- [ ] Update `README.md`:
  - Architecture diagram text: mention ECMWF IFS HRES model
  - API section: note the `model` field in the response
  - Usage section: mention ECMWF model if relevant
  - Tech section: add "Weather model: ECMWF IFS HRES 9km (via Open-Meteo)"
- [ ] Update `AGENTS.md`:
  - API section: add `models=ecmwf_ifs` parameter, mention ECMWF schedule
  - Note the smart cache strategy and model run times
  - Update any references to "default model" or "best match"

---

## Stage 6 — End-to-End Verification

Run the server and verify everything works with ECMWF data.

### Tasks
- [ ] Run `make test` — all Go tests pass
- [ ] Run `make run` — server starts without errors
- [ ] Open browser, verify all 8 sites load with ECMWF data
- [ ] Verify flyability evaluation still works correctly (green/amber/slate blocks)
- [ ] Verify threshold controls re-process without network call
- [ ] Verify refresh button fetches fresh data
- [ ] Verify cache behavior: first call hits API, second call within same run window serves from cache
- [ ] Verify model name is displayed in the frontend
- [ ] Compare a few hours of ECMWF data vs the old default to confirm meaningful differences exist

---

## File Summary

| File | Action |
|------|--------|
| `internal/config/sites.go` | Add `ModelName` constant |
| `internal/forecast/client.go` | Add `&models=ecmwf_ifs` to URL |
| `internal/forecast/cache.go` | Replace fixed TTL with `nextRunCompletion()`, remove `cacheTTL` |
| `internal/forecast/cache_test.go` | Rewrite TTL tests as run-completion tests, add `nextRunCompletion` tests |
| `internal/api/handler.go` | Add `Model` field to `ForecastResponse`, populate from config |
| `internal/api/handler_test.go` | Verify `model` field in response |
| `public/app.js` | Read `model` from API response, store for display |
| `public/index.html` | Update footer to show model name |
| `README.md` | Update architecture, API, tech sections |
| `AGENTS.md` | Update API section, cache strategy notes |
