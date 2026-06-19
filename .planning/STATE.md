---
gsd_state_version: 1.0
milestone: v1.2
milestone_name: milestone
status: planning
stopped_at: Phase 6 UI-SPEC approved
last_updated: "2026-06-19T19:08:32.592Z"
last_activity: 2026-06-19 — v1.2 roadmap created (Phases 6-9), 20/20 requirements mapped
progress:
  total_phases: 9
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-19)

**Core value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Current focus:** Phase 6 — Settings Navigation + Language & Units (v1.2 roadmap created)

## Current Position

Phase: Not started (roadmap complete — ready to plan Phase 6)
Plan: —
Status: Roadmap complete; awaiting Phase 6 planning
Last activity: 2026-06-19 — v1.2 roadmap created (Phases 6-9), 20/20 requirements mapped

## Performance Metrics

**Velocity (v1.0 + v1.1):**

- Total plans completed: 12
- Average duration: — min
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 1 | - | - |
| 02 | 3 | - | - |
| 03 | 2 | - | - |
| 04 | 2 | - | - |
| 05 | 4 | - | - |
| 06 | TBD | - | - |
| 07 | TBD | - | - |
| 08 | TBD | - | - |
| 09 | TBD | - | - |

**Recent Trend:**

- Last 5 plans: —
- Trend: —

*Updated after each plan completion*
| Phase 04 P01 | 17 | 2 tasks | 4 files |
| Phase 04 P02 | 13 | 2 tasks | 3 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [v1.2 roadmap] Phase 6 combines settings navigation wiring (SETNAV-01) with the Language & Units screen (LANG-01, LANG-02) — avoids a thin standalone navigation phase; coarse granularity favors folding small wiring work into the first concrete screen
- [v1.2 roadmap] Phases 7 (Privacy), 8 (Account), 9 (Notifications) each depend only on Phase 6 and are independent of one another — they can be planned/executed in any order after the navigation wiring lands
- [v1.2 roadmap] ACCT-01..05 ship as one combined Account & Profile screen (avatar, bio, email, password, delete) per PROJECT.md target features — matches the existing blank SettingsAccountScreen scaffold
- Extend SvelteKit Valhalla API (not direct Flutter→Valhalla) — keeps credentials server-side
- DraggableScrollableSheet for stats — matches MapScreen pattern
- TTS deferred to v2 — audio complexity not needed for navigation core
- Assume user is at trailhead — simplifies v1 scope
- Dio try-catch as sole offline gate — connectivity packages have false-positive/negative failure modes (captive portals, VPN); catch DioException is simpler and covers all failure modes
- `String? navCacheJson` on TrailEntity — follows `gpxData` precedent; List<List<double>> and List<NavigateManeuver> are unsupported as ObjectBox native types
- Best-effort cache write in downloadTrail — sequential try/catch, never in Future.wait, Valhalla outage cannot block tile download
- freezed 3.x: @JsonSerializable(explicitToJson: true) must be placed on the factory constructor, not above @freezed; class-level placement breaks json_serializable codegen
- NavigateResponse.toJson() serialization fixed (nested maneuvers now serialized) — Phase 4 blocker resolved; unblocked Phase 5 ObjectBox navCacheJson caching
- navCacheJson added as entity-only nullable String on TrailEntity (follows gpxData precedent); objectbox-model.json regenerated and committed together to avoid UID conflicts

### Pending Todos

- Plan Phase 6 first — it wires the routes (/settings/privacy, /settings/language, /settings/notifications) and list entries that Phases 7-9 depend on
- Reuse the existing `Settings` freezed model (bio, language, privacy, notifications map) and `settingsProvider.saveToServer()` for all four screens — no new persistence layer needed

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260615-k0w | Implement along-track projection for waypoint advancement in navigation | 2026-06-15 | 84dac0dd | [260615-k0w-implement-along-track-projection-for-way](.planning/quick/260615-k0w-implement-along-track-projection-for-way/) |
| 260615-ktn | Research flutter_background_geolocation for background navigation support when phone is locked | 2026-06-15 | 6efe690c | [260615-ktn-research-flutter-background-geolocation-](.planning/quick/260615-ktn-research-flutter-background-geolocation-/) |
| 260615-mxk | Implement background navigation so location tracking continues when the phone screen locks | 2026-06-15 | 61591bc7 | [260615-mxk-implement-background-navigation-so-locat](.planning/quick/260615-mxk-implement-background-navigation-so-locat/) |
| 260616-h99 | Create WandererActorSearch component for author filter with actor search dropdown | 2026-06-16 | 30ea7b5f | [260616-h99-create-wandereractorsearch-component-for](.planning/quick/260616-h99-create-wandereractorsearch-component-for/) |
| 260616-j2n | Implement the like feature from the web version | 2026-06-16 | 69d2dd0a | [260616-j2n-implement-the-like-feature-from-the-web-](.planning/quick/260616-j2n-implement-the-like-feature-from-the-web-/) |
| 260618-o0r | Add quick filter bar to library and profile trail screens with horizontal action chips opening bottom sheets for Sort, Category, Difficulty, Elevation, Date, and Completion Status filters using family trail filter provider | 2026-06-18 | fdfd75e1 | [260618-o0r-add-quick-filter-bar-to-library-and-prof](.planning/quick/260618-o0r-add-quick-filter-bar-to-library-and-prof/) |
| 260618-ola | Move profile trail filtering server-side: add filter/sort params to API route and update Flutter provider to send filter state instead of filtering client-side | 2026-06-18 | d84134c2 | [260618-ola-move-profile-trail-filtering-server-side](.planning/quick/260618-ola-move-profile-trail-filtering-server-side/) |
| 260619-okw | Use tracelet to facilitate background location tracking in the navigation screen | 2026-06-19 | 4d70bd85 | [260619-okw-use-tracelet-to-facilitate-background-lo](.planning/quick/260619-okw-use-tracelet-to-facilitate-background-lo/) |

### Blockers/Concerns

- None for v1.2 — the `Settings` freezed model, `settingsProvider`, and `/settings/*` route scaffold already exist; v1.2 is primarily UI + wiring on top of existing infrastructure.

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Audio | TTS maneuver announcements (AUDIO-01, AUDIO-02) | v2 | Init |
| Routing | Navigate from user position to trailhead | Out of scope | Init |
| Routing | Re-routing when off-trail | Out of scope | Init |
| Offline | Stale-cache dialogs, "cached N days ago" UI | Out of scope | v1.1 research |
| Offline | User-initiated cache refresh | Out of scope | v1.1 research |
| Offline | Offline re-routing | Out of scope | v1.1 research |
| Account | API token management (ACCT-F01) | Future | v1.2 requirements |
| Settings | Favourite sport picker, Export, Integrations, Maintenance, Map settings | Out of scope | v1.2 requirements |

## Session Continuity

Last session: 2026-06-19T18:47:13.092Z
Stopped at: Phase 6 UI-SPEC approved
Resume file: .planning/phases/06-settings-navigation-language-units/06-UI-SPEC.md
