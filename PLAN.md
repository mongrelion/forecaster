# Sites Database as JSON — Execution Plan

> **Spec**: `docs/specs/sites-json/SPEC.md`
> **Architecture**: Modular monolith — same pattern as current. Site list moves from a package-level `var Sites` in `internal/config/sites.go` to a JSON file at the project root. The handler receives sites via constructor injection (no global state). JSON is loaded once at startup with fail-fast validation. Configurable via `-sites` flag / `SITES_PATH` env var (12-Factor). Cache continues to key by microdegree lat/lon, so same-coordinate sites share cache entries — harmless and documented.

---

## Stage 1 — Create `sites.json` and `LoadSites` function

Add the JSON file and a loading function with validation. Existing `config.Sites` global and its tests are untouched.

### Tasks
- [x] Create `sites.json` at project root with all 8 existing sites in JSON format
- [x] Add `LoadSites(path string) ([]Site, error)` to `internal/config/sites.go`
- [x] Add `validateSites([]Site) error` with all validation rules
- [x] Add new tests to `internal/config/sites_test.go`: `TestLoadSites_ValidFile`, `TestLoadSites_FileNotFound`, `TestLoadSites_InvalidJSON`, `TestLoadSites_EmptyArray`, `TestLoadSites_EmptyName`, `TestLoadSites_InvalidDirection`, `TestLoadSites_InvalidCoords`, `TestLoadSites_DuplicateName`, `TestLoadSites_Success`
- [x] Run `go test ./internal/config/...` — both old and new tests pass

---

## Stage 2 — Update handler to accept sites as a parameter

Handler stores its own `[]Site` slice. `main.go` still passes `config.Sites` for now — transitional state.

### Tasks
- [x] Add `sites []config.Site` field to `Handler` struct in `internal/api/handler.go`
- [x] Change `NewHandler(cache)` to `NewHandler(sites []config.Site, cache *forecast.Cache)`
- [x] Replace `config.Sites` with `h.sites` in `ServeHTTP`
- [x] Update `cmd/server/main.go`: change `api.NewHandler(cache)` to `api.NewHandler(config.Sites, cache)`
- [x] Run `make test` — all tests pass
- [x] Run `make run` — all 8 sites appear in the frontend

---

## Stage 3 — Wire up JSON loading in main.go, remove global

`main.go` loads sites from the JSON file. The `config.Sites` global and its tests are removed.

### Tasks
- [x] Add `-sites` flag (default `sites.json`) to `cmd/server/main.go`
- [x] Add `SITES_PATH` env var support (wins over flag default if set)
- [x] Call `config.LoadSites(path)` at startup; `log.Fatalf` on error
- [x] Replace `config.Sites` with loaded slice in `NewHandler` call
- [x] Remove `var Sites = []Site{...}` from `internal/config/sites.go`
- [x] Remove old global-dependent tests from `internal/config/sites_test.go`: `TestSitesNotEmpty`, `TestSiteCount`, `TestAllSitesHaveName`, `TestAllSitesHaveTwoDirections`, `TestAllSitesHaveValidCoords`, `TestSiteNamesUnique`, `TestSitesExpectedSiteNames`
- [x] Run `make test` — only `LoadSites`-based tests remain in config package
- [x] Verify `rg "config\.Sites"` returns no results anywhere
- [x] Test: `SITES_PATH=/nonexistent go run ./cmd/server` fails with a clear error
- [x] Test: `go run ./cmd/server -sites /nonexistent` fails with a clear error
- [x] Test: `go run ./cmd/server` loads `sites.json` and serves all 8 sites

---

## Stage 4 — Update handler tests

Handler tests load sites from `sites.json` instead of the removed global. Tests are resilient to site list changes.

### Tasks
- [x] Add `loadTestSites(t)` helper to `internal/api/handler_test.go` using `config.LoadSites("../../sites.json")`
- [x] Replace all `NewHandler(config.Sites, nil)` with `NewHandler(loadTestSites(t), nil)`
- [x] Replace hardcoded `8` with `len(testSites)` in all assertions
- [x] Rename `TestHandlerReturnsAll8SitesWithNames` to `TestHandlerReturnsExpectedSitesWithNames`
- [x] Run `make test` — zero failures

---

## Stage 5 — Documentation

Update README and AGENTS to reflect the JSON file as the source of truth.

### Tasks
- [ ] Update `README.md`: replace Go snippet with JSON example in "Adding or editing sites"
- [ ] Update `README.md`: point "Flying sites" section to `sites.json`
- [ ] Update `README.md`: add `sites.json` to project structure
- [ ] Update `AGENTS.md`: add `sites.json` to file structure
- [ ] Update `AGENTS.md`: note sites are loaded from JSON at startup
- [ ] Remove stale `config.Sites` references from both docs
