# AGENTS.md — Forecaster

Context and spec for AI agents (and developers) picking up this project.

---

## What this is

A Go-backed single-page weather app for paragliding. A Go backend fetches ECMWF IFS HRES 9km hourly forecasts for a hardcoded list of flying sites from the Open-Meteo API, caches responses in-memory with model-run-aware expiration, and serves both the frontend assets and a JSON API. The frontend evaluates each hour against flyability criteria and presents colour-coded hour strips with flyable window summaries.

---

## File structure

```
forecaster/
  cmd/server/main.go        — Entry point, HTTP server setup
  internal/
    api/handler.go           — HTTP handlers
    config/sites.go          — Site list, constants
    forecast/client.go       — Open-Meteo API client + concurrent fetch
    forecast/cache.go        — In-memory cache with ECMWF run-aware expiration
    forecast/processor.go    — Raw → processed data mapping
  public/
    index.html               — Page structure and static markup
    app.css                  — All styling — dark theme, layout, animations
    app.js                   — All logic — rendering, flyability, thresholds
  README.md                  — User + developer documentation
  AGENTS.md                  — This file
  docs/
    wind-direction.jpeg      — Visual reference for compass direction → degree mapping
```

---

## Spec

### Flyability evaluation

For each forecasted hour, the app evaluates four binary criteria. An hour is **flyable** only when ALL of the following are true:

| # | Criterion | Source field | Rule |
|---|---|---|---|
| 1 | Daylight | `is_day` | must equal `1` — handles midnight sun at 64–67°N correctly |
| 2 | Wind direction | `wind_direction_10m` | must fall within the site's defined compass range |
| 3 | Wind gusts | `wind_gusts_10m` | must be ≤ 25 km/h — **hard-coded, not user-configurable** |
| 4 | Cloud cover | `cloud_cover` | must be ≤ user threshold (default 75%) |
| 5 | Rain probability | `precipitation_probability` | must be ≤ user threshold (default 30%) |

**Marginal** hours (amber on the strip): `is_day=1`, direction OK, gusts OK, but **exactly one** of cloud or rain is over threshold.

**Not flyable** hours (dark slate): daytime but one or more criteria failing.

**Night** hours (near-invisible): `is_day=0`.

### Flyable windows

Consecutive flyable hours are grouped into windows. Each window reports:
- `startTime`, `endTime` — ISO8601 timestamps of first and last flyable hour
- `count` — number of hours (displayed end = last hour start + 1h)
- `avgDir` — average wind direction in degrees (displayed as compass name)
- `avgWind` — average wind speed km/h
- `maxGusts` — peak gusts km/h within the window
- `avgCloud`, `avgRain` — averages across the window

### Best bet

The site with the most total flyable hours wins. Tiebreaker: longest single window. The best (longest) window from that site is shown in the banner.

---

## API

### Backend endpoint

**`GET /api/forecast`** — returns ECMWF IFS HRES 9km forecast data for all configured sites.

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
  "model": "ECMWF IFS HRES 9km",
  "fetched_at": "2026-04-26T10:00:00Z"
}
```

### Open-Meteo upstream API

The Go backend calls **`https://api.open-meteo.com/v1/forecast`** with these parameters per site:

```
latitude, longitude   — site coordinates
models                — ecmwf_ifs (selects ECMWF IFS HRES 9km model)
hourly                — is_day, precipitation_probability, temperature_2m,
                        cloud_cover, wind_speed_10m, wind_direction_10m, wind_gusts_10m
timezone              — Europe/Stockholm (returns timestamps in local Swedish time)
past_days             — 0
forecast_days         — 7
```

**CORS:** `access-control-allow-origin: *` — safe to call directly.

All sites are fetched concurrently with `sync.WaitGroup`. Individual fetch failures are caught per-site and surfaced in the `error` field without affecting other sites.

### ECMWF run schedule

ECMWF IFS produces new forecast runs at 00, 06, 12, 18 UTC. Data becomes available approximately 3 hours after each run start (i.e., at 03, 09, 15, 21 UTC). The cache is configured to expire at the next expected completion time, so fresh data is fetched automatically after each new run is published.

---

## Compass direction system

Wind direction ranges are stored as a `[from, to]` pair of 16-point compass names (clockwise, inclusive):

```js
direction: ['SSW', 'WSW']   // accepts wind from SSW clockwise to WSW
direction: ['NW', 'ENE']    // wraps through north — handled automatically
```

Valid names: `N NNE NE ENE E ESE SE SSE S SSW SW WSW W WNW NW NNW`

Each point covers a 22.5° sector centred at multiples of 22.5°. `compassToRange(from, to)` returns `[minDeg, maxDeg]`. When `minDeg > maxDeg` (i.e. the range crosses 0°/360°), `isWindInRange` handles the wrap-around with an OR check.

`degToCompass(deg)` converts any bearing to the nearest compass name — used to display actual wind direction in tooltips and window summaries.

> **Note:** Ranges wider than 180° would produce unexpected wrap-around behaviour in `isWindInRange`. In practice no flying site has an acceptance cone that wide, so this is not a concern.

See `docs/wind-direction.jpeg` for a visual degree reference.

---

## State management

```
state = { rainThreshold, cloudThreshold, sortBy }
_backendSites = []   // cached backend response — survives threshold changes
```

- **Data load** → fetches `GET /api/forecast` → processes → renders
- **Threshold change** → re-processes from `_backendSites` → re-renders (no network call)
- **Refresh button** → re-fetches all sites from the backend

---

## Rendering pipeline

```
loadData()
  └─ fetch('/api/forecast')   single fetch to Go backend
       └─ backend fetches all 8 sites concurrently from Open-Meteo
processAndRender()
  └─ processResponse()        per-site: maps hourly arrays → hour objects with flyable/marginal flags
  └─ findWindows()            groups consecutive flyable hours → window summaries
  └─ findBestBet()            picks the site + window to highlight
  └─ renderAll()
       ├─ renderBestBet()     top banner
       └─ renderCard()        per site: header + hour strip + window rows
            ├─ createHourBlock()   one DOM element per hour, with tooltip listeners
            └─ renderWindowRow()   one row per flyable window
```

---

## Design tokens (CSS variables)

| Variable | Value | Used for |
|---|---|---|
| `--bg` | `#080c18` | Page background |
| `--surface` | `#0d1424` | Card backgrounds |
| `--accent` | `#38bdf8` | Sky blue — logo, active states, best bet banner |
| `--good` | `#4ade80` | Flyable hours and windows |
| `--warn` | `#fbbf24` | Marginal hours |
| `--block-night` | `#0c1526` | Night hour blocks (near-invisible) |
| `--block-day` | `#1c2b42` | Daytime non-flyable blocks |

Fonts: **Syne** (site names, logo), **Barlow** (UI text), **JetBrains Mono** (all numeric data).

---

## Known limitations / edge cases

- **Timezone mismatch:** `todayStr()` uses the browser's local clock. If the user is not in Sweden, the "Today" label and day-of-week display may be off by ±1 day. For a personal local tool this is acceptable.
- **avgDir circular average:** wind direction averaging uses a simple arithmetic mean, which breaks near 0°/360° (e.g., averaging 350° and 10° gives 180° instead of 0°). For the narrow windows typical in paragliding this is rarely an issue.
- **Max gusts hard limit:** 25 km/h is coded as `MaxGusts` in `internal/config/sites.go`. It is intentionally not exposed in the UI.
- **avgDir circular average:** wind direction averaging uses a simple arithmetic mean, which breaks near 0°/360° (e.g., averaging 350° and 10° gives 180° instead of 0°). For the narrow windows typical in paragliding this is rarely an issue.
- **Cache expiration:** The cache expires at the next ECMWF run completion time (03, 09, 15, or 21 UTC), not a fixed TTL.

---

## Backlog / ideas

- [ ] Circular mean for wind direction averaging in windows
- [ ] `localStorage` persistence for threshold preferences
- [ ] Touch-friendly tooltip (tap to show on mobile)
- [ ] Hosting on GitHub Pages with a custom domain
- [ ] Site metadata (elevation, aspect photo, launch description)
