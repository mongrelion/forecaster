# Forecaster

A personal weather companion for paragliding. Checks the forecast for all your flying sites at once and tells you where to go.

Instead of looking up 6 different forecasts and suffering decision paralysis, Forecaster fetches them all in parallel, evaluates flyability hour by hour, and surfaces the best window across all sites.

## Architecture

A Go backend serves both the frontend assets and a JSON API. The backend fetches ECMWF IFS HRES 9km forecasts from [Open-Meteo](https://open-meteo.com), caches responses in-memory (expiring at the next ECMWF model run completion), and returns raw hourly data to the frontend. The frontend computes all flyability flags client-side so threshold changes don't require a network call.

```
┌──────────┐    GET /api/forecast    ┌──────────┐    Open-Meteo API    ┌────────────┐
│  Browser  │ ────────────────────── │  Go      │ ─────────────────── │  open-meteo │
│  app.js   │ ◀───────────────────── │  server  │ ◀────────────────── │  .com       │
└──────────┘     JSON + static       └──────────┘    concurrent        └────────────┘
                                assets (public/)     fetch + cache
                                Weather model: ECMWF IFS HRES 9km
```

## Usage

### Running the server

```
# Build and run (default :8080)
make run
```

Or manually:

```
go run ./cmd/server
```

Then open **http://localhost:8080** in any modern browser.

### Controls

| Control | What it does |
|---|---|
| **Refresh ↻** | Re-fetch all sites with fresh data |
| **Sort** | Switch between Flyability (most windows first) and alphabetical (A–Z) ordering |
| **Thresholds ⚙** | Adjust the rain and cloud cover limits — re-evaluated client-side without a network call |

Hover any hour block on the strip to see the exact conditions for that hour.

## Flyability criteria

An hour is marked **flyable** (green) when all five conditions are met simultaneously:

| Criterion | Threshold |
|---|---|
| Wind direction | Within the site's defined compass range |
| Wind gusts | ≤ 25 km/h (fixed) |
| Cloud cover | ≤ configurable % (default 75%) |
| Precipitation probability | ≤ configurable % (default 30%) |
| Daylight | `is_day = 1` from the API — handles midnight sun correctly |

An hour is **marginal** (amber) when direction and gusts are OK but exactly one of cloud or rain is over the threshold.

## API

**`GET /api/forecast`** — returns forecast data for all configured sites using the ECMWF IFS HRES 9km model.

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

Errors are surfaced per-site in the `error` field (null on success). The `model` field indicates which weather model was used.

## Flying sites

For an up-to-date list of flying sites, see `internal/config/sites.go`

## Adding or editing sites

Edit the `Sites` slice in `internal/config/sites.go`:

```go
{Name: "My Site", Direction: [2]string{"SSW", "WSW"}, Lat: 63.123, Lon: 18.456}
```

`Direction` is a `[from, to]` pair of 16-point compass names defining the acceptable wind range (clockwise). For ranges that cross north, e.g. `["NW", "NE"]`, wrap-around is handled automatically.

Valid compass points: `N NNE NE ENE E ESE SE SSE S SSW SW WSW W WNW NW NNW`

See `docs/wind-direction.jpeg` for a visual degree reference.

## Development

### Prerequisites

- Go 1.26+

### Commands

| Command | What it does |
|---|---|
| `make build` | Build server binary to `./server` |
| `make run` | Run the server from source |
| `make test` | Run all Go tests |
| `make clean` | Remove built binary |
| `make tidy` | Format code and tidy modules |

### Project structure

```
forecaster/
  cmd/server/main.go          — Entry point, HTTP server setup
  internal/
    api/handler.go             — HTTP handlers
    api/handler_test.go
    config/sites.go            — Site list, constants
    config/sites_test.go
    forecast/client.go         — Open-Meteo API client + concurrent fetch
    forecast/cache.go          — In-memory cache with 24h TTL
    forecast/cache_test.go
    forecast/processor.go      — Raw → processed data mapping
    forecast/processor_test.go
    forecast/fixtures_test.go  — Test helpers
  public/
    index.html                 — Page structure
    app.css                    — All styling
    app.js                     — All frontend logic (rendering, flyability, thresholds)
  docs/
    wind-direction.jpeg        — Visual compass reference
```

## Tech

- **Frontend:** Vanilla HTML + CSS + JS — zero dependencies
- **Backend:** Go (stdlib) — no external dependencies
- **Data:** [Open-Meteo](https://open-meteo.com) free weather API (no key required, CORS enabled)
- **Weather model:** ECMWF IFS HRES 9km (via Open-Meteo)
- **Caching:** In-memory, per-coordinate key, expires at next ECMWF run completion (03/09/15/21 UTC)
- **Fonts:** Syne, Barlow, JetBrains Mono via Google Fonts
