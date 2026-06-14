---
gsd_state_version: 1.0
milestone: v1.1
milestone_name: Offline Navigation
status: planning
last_updated: "2026-06-14T00:00:00.000Z"
last_activity: 2026-06-14
progress:
  total_phases: 2
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-14)

**Core value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Current focus:** Phase 04 — Serialization Fix + Entity Schema

## Current Position

Phase: 4 — Serialization Fix + Entity Schema
Plan: —
Status: Ready to plan
Last activity: 2026-06-14 — v1.1 roadmap created (Phase 4 and Phase 5 defined)

## Performance Metrics

**Velocity (v1.0):**

- Total plans completed: 6
- Average duration: — min
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 1 | - | - |
| 02 | 3 | - | - |
| 03 | 2 | - | - |
| 04 | TBD | - | - |
| 05 | TBD | - | - |

**Recent Trend:**

- Last 5 plans: —
- Trend: —

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Extend SvelteKit Valhalla API (not direct Flutter→Valhalla) — keeps credentials server-side
- DraggableScrollableSheet for stats — matches MapScreen pattern
- TTS deferred to v2 — audio complexity not needed for navigation core
- Assume user is at trailhead — simplifies v1 scope
- Dio try-catch as sole offline gate — connectivity packages have false-positive/negative failure modes (captive portals, VPN); catch DioException is simpler and covers all failure modes
- `String? navCacheJson` on TrailEntity — follows `gpxData` precedent; List<List<double>> and List<NavigateManeuver> are unsupported as ObjectBox native types
- Best-effort cache write in downloadTrail — sequential try/catch, never in Future.wait, Valhalla outage cannot block tile download

### Pending Todos

- Fix NavigateResponse.toJson() serialization bug before any ObjectBox work (Phase 4)
- Commit objectbox-model.json immediately after build_runner to prevent UID conflicts

### Blockers/Concerns

- `NavigateResponse.toJson()` missing `@JsonSerializable(explicitToJson: true)` — BLOCKING for Phase 4; identified in generated navigate_response.g.dart

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Audio | TTS maneuver announcements (AUDIO-01, AUDIO-02) | v2 | Init |
| Routing | Navigate from user position to trailhead | Out of scope | Init |
| Routing | Re-routing when off-trail | Out of scope | Init |
| Offline | Stale-cache dialogs, "cached N days ago" UI | Out of scope | v1.1 research |
| Offline | User-initiated cache refresh | Out of scope | v1.1 research |
| Offline | Offline re-routing | Out of scope | v1.1 research |

## Session Continuity

Last session: 2026-06-14T00:00:00.000Z
Stopped at: v1.1 roadmap created — Phase 4 ready to plan
Resume file: None
