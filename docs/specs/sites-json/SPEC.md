# Sites Database as JSON File

## Overview

Move the flying site list out of compiled Go code (`internal/config/sites.go`) and into a standalone JSON file (`sites.json`) at the project root. The server loads and validates the file at startup, failing fast on errors. The file becomes the single source of truth for sites — editable without recompilation.

## Problem Statement

Currently, adding or modifying a flying site requires editing Go source, recompiling, and redeploying the binary. For a personal tool where site lists evolve, this is unnecessary friction. A JSON file decouples data from code, enables simple editing, and follows 12-Factor App configuration principles.

## Requirements

### Functional Requirements
- [ ] Site data lives in `sites.json` at the project root
- [ ] Server loads sites at startup via a configurable path (`-sites` flag, `SITES_PATH` env var, default `sites.json`)
- [ ] Malformed or missing JSON → clear error message + exit(1) (fail fast)
- [ ] Invalid site data (empty name, bad coords, unknown compass points, duplicate names) → fail fast with a descriptive error
- [ ] The `Site` Go struct, `ModelName`, and `MaxGusts` constants remain in `internal/config/sites.go`
- [ ] The handler receives sites as a constructor parameter (no package-level global)
- [ ] All existing tests pass after the refactor
- [ ] JSON file is committed to version control

### Non-Functional Requirements
- **Performance:** JSON parsing happens once at startup — no runtime overhead
- **Maintainability:** Adding a site = editing one JSON object, no Go changes
- **12-Factor:** Configurable via env var or CLI flag, with a sensible default

## Architecture (High-Level)

```
sites.json ──▶ config.LoadSites(path) ──▶ []Site ──▶ main.go ──▶ api.NewHandler(sites, cache)
                    │                                       │
                    └─ validate (fail fast)                  └─ h.sites (replaces config.Sites global)
```

- `config.LoadSites(path)` reads, parses, and validates the JSON file
- `main.go` resolves the path from `-sites` flag or `SITES_PATH` env var (default `sites.json`)
- `api.NewHandler` signature changes from `(cache)` to `(sites, cache)`
- `config.Sites` package-level variable is **removed**

## JSON Format

```json
[
  {
    "name": "Balberget Ramp",
    "direction": ["SSW", "WSW"],
    "lat": 63.94344038093285,
    "lon": 19.046277812311036
  }
]
```

| Field | Type | Constraints |
|---|---|---|
| `name` | string | Non-empty, unique across all sites |
| `direction` | [string, string] | Both must be valid 16-point compass names (N, NNE, NE, ENE, E, ESE, SE, SSE, S, SSW, SW, WSW, W, WNW, NW, NNW) |
| `lat` | float64 | -90 ≤ lat ≤ 90 |
| `lon` | float64 | -180 ≤ lon ≤ 180 |

## Implementation Stages

### Stage 1: Create `sites.json` and `LoadSites` function

**Goal:** The JSON file exists and a Go function can load + validate it. Existing `config.Sites` global and its tests remain untouched.

**Files to create/modify:**
- **Create** `sites.json` — the 8 existing sites in JSON format
- **Modify** `internal/config/sites.go`:
  - Add `LoadSites(path string) ([]Site, error)` — reads file, unmarshals, validates
  - Add validation helpers: `validateSites([]Site) error`
  - Keep `Site` struct, `ModelName`, `MaxGusts` — unchanged
  - Keep `var Sites` and its hardcoded data (removed in Stage 3)

**Validation rules (fail fast, descriptive errors):**
- File must exist and be readable
- JSON must parse as `[]Site`
- At least one site must be present (empty array is an error)
- Every site must have a non-empty `Name`
- `Direction` must have exactly 2 elements, both valid compass names (N, NNE, NE, ENE, E, ESE, SE, SSE, S, SSW, SW, WSW, W, WNW, NW, NNW)
- `Lat` must be in [-90, 90], `Lon` in [-180, 180]
- No duplicate site names

> **Note:** Duplicate lat/lon pairs are intentionally **not** rejected. The cache keys by microdegree coordinates, so two sites at the same location would share a cache entry — this is harmless (same forecast data) and arguably useful (two launches at the same mountain with different wind acceptance cones).

**New tests to add (in `internal/config/sites_test.go`):**
- `TestLoadSites_ValidFile` — loads a temp JSON file, verifies correct sites returned
- `TestLoadSites_FileNotFound` — error on missing file
- `TestLoadSites_InvalidJSON` — error on malformed JSON
- `TestLoadSites_EmptyArray` — error on `[]` (zero sites)
- `TestLoadSites_EmptyName` — error on site with empty name
- `TestLoadSites_InvalidDirection` — error on bad compass name
- `TestLoadSites_InvalidCoords` — error on out-of-range lat/lon
- `TestLoadSites_DuplicateName` — error on duplicate name
- `TestLoadSites_Success` — loads the real `../../sites.json` (relative from test cwd, since test files live in `internal/config/`) and verifies all 8 sites

**Existing tests preserved (removed in Stage 3):**
- `TestSitesNotEmpty`, `TestSiteCount`, `TestAllSitesHaveName`, `TestAllSitesHaveTwoDirections`, `TestAllSitesHaveValidCoords`, `TestSiteNamesUnique`, `TestSitesExpectedSiteNames` — still pass against `var Sites`
- `TestMaxGustsConstant`, `TestSiteJSON` — permanent, don't depend on the global

**Acceptance Criteria:**
- `go test ./internal/config/...` passes — both old and new tests
- Loading a malformed JSON file returns a descriptive error
- Loading a file with invalid compass direction returns an error mentioning the bad value
- Loading an empty array returns an error ("at least one site required")
- Loading the real `sites.json` returns all 8 sites

**Dependencies:** None

---

### Stage 2: Update handler to accept sites as a parameter

**Goal:** The handler stores its own sites slice — but `main.go` still passes the old `config.Sites` global for now. This is a transitional state: the handler is ready for injected sites, but the source hasn't changed yet.

**Files to modify:**
- `internal/api/handler.go`:
  - `Handler` struct gains a `sites []config.Site` field
  - `NewHandler(cache *forecast.Cache)` → `NewHandler(sites []config.Site, cache *forecast.Cache)`
  - `ServeHTTP`: replace `config.Sites` with `h.sites`
- `cmd/server/main.go`:
  - Update call: `api.NewHandler(cache)` → `api.NewHandler(config.Sites, cache)`

**Acceptance Criteria:**
- `make test` passes — handler tests still use `NewHandler(config.Sites, nil)` (or equivalent)
- `make run` starts and serves all 8 sites correctly (still using the global as source)
- `rg "config\.Sites"` in `internal/api/` returns no results

**Dependencies:** Stage 1

---

### Stage 3: Wire up JSON loading in main.go, remove global

**Goal:** `main.go` loads sites from the JSON file. The `config.Sites` global and its hardcoded data are removed. Old config tests that depended on the global are removed.

**Files to modify:**
- `cmd/server/main.go`:
  - Add `-sites` flag (default `sites.json`)
  - Check `SITES_PATH` env var as override (if set and non-empty, it wins over the flag default)
  - Call `config.LoadSites(path)` at startup
  - If error: `log.Fatalf("loading sites: %v", err)`
  - Replace `config.Sites` with the loaded slice in the `NewHandler` call
- `internal/config/sites.go`:
  - **Remove** `var Sites = []Site{...}` and its hardcoded data
  - Keep `Site` struct, `LoadSites`, `validateSites`, `ModelName`, `MaxGusts`
- `internal/config/sites_test.go`:
  - Remove tests that referenced the `Sites` global: `TestSitesNotEmpty`, `TestSiteCount`, `TestAllSitesHaveName`, `TestAllSitesHaveTwoDirections`, `TestAllSitesHaveValidCoords`, `TestSiteNamesUnique`, `TestSitesExpectedSiteNames`
  - Keep `TestMaxGustsConstant`, `TestSiteJSON`, and all new `TestLoadSites_*` tests from Stage 1

**Path resolution priority:**
1. `SITES_PATH` environment variable (if set and non-empty)
2. `-sites` CLI flag value (defaults to `sites.json`)

**Acceptance Criteria:**
- `rg "config\.Sites"` returns no results anywhere in the project
- `make test` passes — only `LoadSites`-based tests remain in config package
- `go run ./cmd/server` loads `sites.json` and starts normally
- `SITES_PATH=/nonexistent go run ./cmd/server` fails with a clear error
- `go run ./cmd/server -sites /nonexistent` fails with a clear error
- All 8 sites appear in the frontend

**Dependencies:** Stage 2

---

### Stage 4: Update handler tests

**Goal:** Handler tests use sites loaded from `sites.json` (or a minimal fixture) instead of the old `config.Sites` global, and don't break when sites are added or removed.

**Files to modify:**
- `internal/api/handler_test.go`:
  - Add a `loadTestSites(t)` helper that calls `config.LoadSites("../../sites.json")` (relative from `internal/api/`)
  - Update all `NewHandler(config.Sites, nil)` calls to `NewHandler(loadTestSites(t), nil)`
  - Tests that assert "8 sites": replace hardcoded count with `len(testSites)`
  - `TestHandlerResponseHasSitesAndFetchedAt`: assert `len(resp.Sites) == len(testSites)` instead of `== 8`
  - `TestHandlerReturnsAll8SitesWithNames`: rename to `TestHandlerReturnsExpectedSitesWithNames` and compare against loaded sites
  - `TestHandlerCaches`: use loaded site count

**Acceptance Criteria:**
- `make test` passes with zero failures
- No code in the project references `config.Sites`
- Handler tests don't break when a site is added to `sites.json`

**Dependencies:** Stage 3

---

### Stage 5: Documentation

**Goal:** Reflect the change in project docs.

**Files to modify:**
- `README.md`:
  - "Adding or editing sites" section: replace Go snippet with JSON example
  - "Flying sites" section: point to `sites.json` instead of `internal/config/sites.go`
  - "Project structure": add `sites.json` entry at root
- `AGENTS.md`:
  - File structure: add `sites.json` entry
  - Spec / API sections: mention sites are loaded from JSON at startup
  - Remove or update references to `config.Sites` global

**Acceptance Criteria:**
- Docs are accurate and reflect the current state
- A new developer can add a site by reading the README

**Dependencies:** Stage 3

---

## Testing Strategy

- **Unit tests** for `LoadSites` with temp files (valid, invalid, edge cases) — Stage 1
- **Integration tests** for handler with loaded sites — Stage 4
- **Manual smoke test:** `make run`, open browser, verify all sites load — Stage 5

## Open Questions

- None at this stage — all decisions confirmed.
