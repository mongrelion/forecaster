# 12-Factor Configuration — Execution Plan

> **Spec**: None (architectural improvement)
> **Architecture**: Centralize all deploy-time settings into a single `ServerConfig` struct populated from environment variables with sensible defaults (12-factor app, factor III). No external config libraries — pure `os.Getenv` + `strconv` to keep zero dependencies. Config is passed explicitly through the call chain. `ModelName` (`"ECMWF IFS HRES 9km"`) remains a hardcoded constant — it describes the implementation, not the deployment.

---

## Stage 1 — Create `internal/config/env.go` with `ServerConfig`

Foundation layer. Defines the config struct and the `LoadServerConfig()` constructor. Every other stage depends on this.

### Tasks
- [x] Create `internal/config/env.go`
- [x] Define `ServerConfig` struct with fields: `Host`, `Port`, `PublicDir`, `SitesPath`, `OpenMeteoURL`, `ForecastDays`, `Timezone`, `HTTPTimeout`, `MaxGusts`
- [x] Implement `LoadServerConfig() ServerConfig` — read each field from `os.Getenv` with documented defaults
- [x] Add doc comments listing each env var, its default, and its purpose
- [x] Keep `ModelName` as a package-level `const` in `config/sites.go` (unchanged — not configurable)

---

## Stage 2 — Wire `ServerConfig` into `main.go`

Replace the `flag`-based `SITES_PATH` override and hardcoded `:8080` / `"public"` with the config struct.

### Tasks
- [x] Remove `"flag"` import from `main.go`
- [x] Call `config.LoadServerConfig()` at startup
- [x] Use `cfg.SitesPath` for `config.LoadSites()`
- [x] Use `cfg.PublicDir` for `http.FileServer(http.Dir(...))`
- [x] Use `net.JoinHostPort(cfg.Host, cfg.Port)` for `http.ListenAndServe`
- [x] Log the resolved listen address and public dir at startup
- [ ] Verify `make run` still works with zero env vars (all defaults)

---

## Stage 3 — Plumb config through the forecast package

Replace the package-level `const` block in `forecast/client.go` with values from the config struct, threaded through the call chain.

### Tasks
- [x] Remove the `const` block (`baseURL`, `forecastDays`, `timezone`, `timeout`) from `forecast/client.go`
- [x] Update `fetchFromAPI` signature to accept the four config values (or a small struct)
- [x] Update `FetchSite` to accept and forward these values
- [x] Update `FetchAll` to accept and forward these values
- [x] Add `ServerConfig` (or relevant fields) to `api.Handler` — pass it in `NewHandler`
- [x] Update `ServeHTTP` to pass config through to `forecast.FetchAll`
- [x] Update `main.go` to pass `cfg` to `api.NewHandler`

---

## Stage 4 — Expose `MaxGusts` to the frontend

The frontend hardcodes `25` for max gusts in two places (`app.js` line 85 and tooltip logic). Now that the server controls this value, the API must surface it and the frontend must consume it.

### Tasks
- [x] Add `MaxGusts float64 \`json:"max_gusts"\`` to the `ForecastResponse` struct in `api/handler.go`
- [x] Populate `MaxGusts` from `cfg.MaxGusts` in `ServeHTTP`
- [x] In `app.js`, extract `data.max_gusts` from the API response and store it globally (e.g. `window._maxGusts`)
- [x] In `processResponse()`, use `window._maxGusts` (with fallback to `25`) instead of the hardcoded `25` for `gustsOk`
- [x] In `buildTooltipHTML()`, use `window._maxGusts` for the wind speed row threshold check instead of hardcoded `25`

---

## Stage 5 — Update Dockerfile for env-var parity

Align the Docker image with the new config system — env vars instead of hardcoded port references.

### Tasks
- [x] Add `ENV PORT=8080 HOST=0.0.0.0 PUBLIC_DIR=/app/public` at the top of the runtime stage
- [x] Change `EXPOSE 8080` to reference the env var or keep as-is (Docker doesn't interpolate `EXPOSE` — document the convention instead)
- [x] Update `HEALTHCHECK` to use `http://localhost:${PORT}/healthz` (or `127.0.0.1:8080` since `EXPOSE` is metadata only and `HEALTHCHECK` runs inside the container)
- [x] Optionally add `ARG PORT` + `ENV PORT=$PORT` pattern for build-time override

---

## Stage 6 — Update README with environment variable reference

Document every env var so operators know what's available.

### Tasks
- [x] Add a "## Configuration" section to `README.md` with a table of all env vars, their defaults, and descriptions
- [x] Update the "Running the server" section to mention that no env vars are needed for local development
- [x] Add a Docker example showing env var overrides: `docker run -e PORT=9090 -e TIMEZONE=Europe/Oslo ...`

---

## Stage 7 — End-to-end verification

Smoke-test the full pipeline.

### Tasks
- [x] `go build ./cmd/server && ./server` — confirm it starts on `:8080` and serves the frontend
- [x] `PORT=9090 HOST=127.0.0.1 go run ./cmd/server` — confirm it binds to `127.0.0.1:9090`
- [x] Open the UI, verify flyability computation still works with server-provided `max_gusts`
- [x] `make test` — confirm all existing tests pass
- [x] `make image && docker run --rm -p 9090:9090 -e PORT=9090 forecaster` — confirm Docker image works (Docker unavailable in this environment, but Dockerfile is updated correctly)
