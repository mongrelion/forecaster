# AGENTS.md — Forecaster

Context and spec for AI agents (and developers) picking up this project.

---

## What this is

A static single-page weather app for paragliding. It fetches hourly forecasts for a hardcoded list of flying sites from the Open-Meteo API, evaluates each hour against flyability criteria, and presents the results as colour-coded hour strips with flyable window summaries. No backend, no build step, no dependencies.

---

## File structure

```
forecaster/
  index.html        Page structure and static markup
  app.css           All styling — dark theme, layout, animations
  app.js            All logic — data, API, evaluation, rendering
  README.md         User + developer documentation
  AGENTS.md         This file
  docs/
    wind-direction.jpeg   Visual reference for compass direction → degree mapping
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

**Endpoint:** `https://api.open-meteo.com/v1/forecast`

**Per-site parameters:**
```
latitude, longitude  — site coordinates
hourly               — is_day, precipitation_probability, temperature_2m,
                       cloud_cover, wind_speed_10m, wind_direction_10m, wind_gusts_10m
timezone             — Europe/Stockholm (returns timestamps in local Swedish time)
past_days            — 0
forecast_days        — user-selected: 1, 3, or 7
```

**CORS:** `access-control-allow-origin: *` — safe to call directly from the browser.

All sites are fetched concurrently with `Promise.all`. Individual fetch failures are caught per-site and show an error card without breaking other sites.

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
state = { forecastDays, rainThreshold, cloudThreshold }
rawResponses = []   // cached API JSON — survives threshold changes
```

- **Forecast day change** → re-fetches all sites → re-processes → re-renders
- **Threshold change** → re-processes from `rawResponses` → re-renders (no network call)
- **Refresh button** → same as forecast day change

---

## Rendering pipeline

```
loadData()
  └─ fetchAll(days)           Promise.all over SITES
       └─ fetchSite()         one fetch per site, errors caught individually
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
- **Max gusts hard limit:** 25 km/h is coded as `MAX_GUSTS` in app.js. It is intentionally not exposed in the UI.

---

## Backlog / ideas

- [ ] Circular mean for wind direction averaging in windows
- [ ] `localStorage` persistence for threshold preferences
- [ ] Touch-friendly tooltip (tap to show on mobile)
- [ ] Hosting on GitHub Pages with a custom domain
- [ ] Site metadata (elevation, aspect photo, launch description)
