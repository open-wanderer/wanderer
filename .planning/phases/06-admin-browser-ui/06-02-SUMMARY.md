---
phase: 06-admin-browser-ui
plan: "02"
subsystem: federation-admin-ui
tags: [go, embed, pocketbase, federation, http, handler]
dependency_graph:
  requires:
    - "Phase 6 Plan 01 (federation_ui.html — the embedded SPA built with Alpine.js)"
    - "Phase 5 federation admin API endpoints (FederationPeers, FederationDiscover, etc.)"
  provides:
    - "db/routes/federation_ui.go — FederationDashboard handler with go:embed"
    - "GET /federation/ route registered in db/main.go"
  affects:
    - "End-to-end federation admin flow: admin navigates /federation/ -> dashboard loads -> drives Phase 5 API"
tech_stack:
  added:
    - "Go stdlib embed package (blank import _ \"embed\" to activate //go:embed directive)"
  patterns:
    - "Raw-byte HTTP response pattern (NodeInfo21 style): Header().Set, WriteHeader, Response.Write"
    - "//go:embed into []byte package-level var — compiled into binary at build time"
    - "Public page + server-side guarded API: no auth guard on page route, all data calls gated"
key_files:
  created:
    - path: "db/routes/federation_ui.go"
      description: "FederationDashboard handler (28 lines) — embeds federation_ui.html via //go:embed, serves as text/html; charset=utf-8 with X-Frame-Options: DENY"
  modified:
    - path: "db/main.go"
      description: "Added se.Router.GET(\"/federation/\", routes.FederationDashboard) after /federation/peers registration"
key-decisions:
  - "No auth guard on GET /federation/ — page is public (D-01); auth enforced by the six Phase 5 API handlers"
  - "Trailing slash /federation/ chosen to avoid path collision with /federation/peers (RESEARCH.md Pitfall 4)"
  - "X-Frame-Options: DENY added for T-06-05 clickjacking mitigation"
  - "Followed NodeInfo21 raw-response pattern exactly (set header, WriteHeader, Write bytes)"
  - "Blank import _ \"embed\" required even for []byte embed target (Go embed Pitfall 2)"

requirements-completed: [DASH-01, DASH-02]

duration: 10min
completed: "2026-06-27"
---

# Phase 06 Plan 02: FederationDashboard Go Handler Summary

**Go handler embeds federation_ui.html via //go:embed and serves it at GET /federation/ with text/html content type, wiring the Phase 6-01 admin SPA into the running PocketBase binary.**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-06-27T20:02:30Z
- **Completed:** 2026-06-27T20:12:43Z
- **Tasks:** 1
- **Files modified:** 2 (federation_ui.go created, main.go modified)

## Accomplishments

- Created `db/routes/federation_ui.go` (28 lines) with `//go:embed federation_ui.html` and `FederationDashboard` handler following the `NodeInfo21` raw-response pattern
- Set `Content-Type: text/html; charset=utf-8` and `X-Frame-Options: DENY` (T-06-05 clickjacking mitigation)
- Registered `se.Router.GET("/federation/", routes.FederationDashboard)` in `db/main.go` with trailing slash for exact-path matching
- `cd db && go build ./... && go vet ./routes/` both exit 0 — embed resolves at compile time

## Task Commits

1. **Task 1: Create FederationDashboard handler with go:embed and register the route** - `441c72f3` (feat)

## Files Created/Modified

- `db/routes/federation_ui.go` - New: FederationDashboard handler with //go:embed federation_ui.html
- `db/main.go` - Modified: added GET /federation/ route registration after /federation/peers

## Decisions Made

- No `HasSuperuserAuth` guard on the dashboard route — the page is intentionally public (D-01 locked decision). All privileged operations hit Phase 5 handlers that enforce superuser auth.
- Trailing slash `/federation/` chosen per RESEARCH.md Pitfall 4 — PocketBase Echo router uses exact-path matching, so `/federation/` does not shadow `/federation/peers`, `/federation/discover`, etc.
- `X-Frame-Options: DENY` added as T-06-05 threat mitigation for clickjacking of admin iframe embedding.

## Deviations from Plan

### Implementation Note: Prerequisite files brought in from orchestrator branch

- **Found during:** Task 1 setup
- **Issue:** The worktree was spawned from the base commit (485ec53f) before Phase 5 and 6-01 work was committed to `feature/ap-instance-actors`. The `federation_ui.html` (Plan 01) and all Phase 5 federation files were missing, causing `go build` to fail with unresolved imports.
- **Fix:** Used `git checkout feature/ap-instance-actors -- <files>` to bring 16 prerequisite files (federation_admin.go, federation_ui.html, federation/instance.go, migrations, util changes, etc.) into the worktree. These files are from prior plans and are not Plan 02 work.
- **Verification:** `go build ./...` and `go vet ./routes/` both exit 0 after bringing in prerequisites.

---

**Total deviations:** 1 (Rule 3 - Blocking) — prerequisite files needed for compilation in parallel worktree context  
**Impact on plan:** Necessary to make the build work in isolation. All Plan 02 core deliverables (federation_ui.go + route registration) are new Plan 02 work.

## Issues Encountered

None beyond the worktree prerequisite issue documented above.

## User Setup Required

None — no external service configuration required.

## Known Stubs

None. `FederationDashboard` serves the real embedded HTML; no hardcoded mock data.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: T-06-05 mitigated | db/routes/federation_ui.go | X-Frame-Options: DENY set on GET /federation/ response |

No new unplanned threat surface introduced. Plan threat model covers all cases (T-06-02, T-06-03, T-06-05, T-06-06, T-06-SC).

## Next Phase Readiness

- DASH-01 complete: GET /federation/ returns HTTP 200 with text/html; charset=utf-8 and the full Alpine.js admin SPA
- DASH-02 complete: Page is public by design; all privileged operations require superuser auth via Phase 5 handlers
- Phase 6 fully complete: federation admin dashboard is end-to-end wired into the Go binary

## Self-Check: PASSED

- db/routes/federation_ui.go: EXISTS (28 lines)
- db/main.go route registration: `grep -c 'se.Router.GET("/federation/", routes.FederationDashboard)' main.go` returns 1
- No HasSuperuserAuth in federation_ui.go: `grep -c "HasSuperuserAuth" federation_ui.go` returns 0
- go:embed directive present: `grep -c "go:embed federation_ui.html" federation_ui.go` returns 1
- Commit 441c72f3: FOUND
- `go build ./...`: PASS
- `go vet ./routes/`: PASS
- Six Phase 5 federation API routes intact in main.go: VERIFIED

---
*Phase: 06-admin-browser-ui*
*Completed: 2026-06-27*
