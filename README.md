# Forecaster

A personal weather companion for paragliding. Checks the forecast for all your flying sites at once and tells you where to go.

Instead of looking up 6 different forecasts and suffering decision paralysis, Forecaster fetches them all in parallel, evaluates flyability hour by hour, and surfaces the best window across all sites.

## Usage

Open `index.html` in any modern browser — no server, no install, no build step.

| Control | What it does |
|---|---|
| **1 day / 3 days / 7 days** | How far ahead to look |
| **Thresholds ⚙** | Adjust the rain and cloud cover limits |
| **↻** | Re-fetch all sites with fresh data |

Hover any hour block on the strip to see the exact conditions for that hour.

## Flyability criteria

An hour is marked **flyable** (green) when all four conditions are met simultaneously:

| Criterion | Threshold |
|---|---|
| Wind direction | Within the site's defined range |
| Wind gusts | ≤ 25 km/h (fixed) |
| Cloud cover | ≤ configurable % (default 75%) |
| Precipitation probability | ≤ configurable % (default 30%) |
| Daylight | `is_day = 1` from the API — handles midnight sun correctly |

An hour is **marginal** (amber) when direction and gusts are OK but exactly one of cloud or rain is over the threshold.

## Flying sites

For an up-to-date list of flying sites, see `app.js`

## Adding or editing sites

Edit the `SITES` array at the top of `app.js`:

```js
{ name: 'My Site', direction: ['SSW', 'SW'], lat: 63.123, lon: 18.456 }
```

`direction` is a `[from, to]` pair of 16-point compass names defining the acceptable wind range (clockwise). For ranges that cross north, e.g. `['NW', 'NE']`, wrap-around is handled automatically.

Valid compass points: `N NNE NE ENE E ESE SE SSE S SSW SW WSW W WNW NW NNW`

See `docs/wind-direction.jpeg` for a visual degree reference.

## Tech

- Vanilla HTML + CSS + JS — zero dependencies, zero build tooling
- Data: [Open-Meteo](https://open-meteo.com) free weather API (no key required, CORS enabled)
- Fonts: Syne, Barlow, JetBrains Mono via Google Fonts
