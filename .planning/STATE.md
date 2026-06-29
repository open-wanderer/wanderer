---
gsd_state_version: 1.0
milestone: v1.2
milestone_name: milestone
status: Awaiting next milestone
stopped_at: Phase 9 context gathered
last_updated: "2026-06-29T11:47:50.049Z"
last_activity: 2026-06-29 — Milestone v1.2 completed and archived
progress:
  total_phases: 9
  completed_phases: 4
  total_plans: 9
  completed_plans: 9
  percent: 44
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-29)

**Core value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Current focus:** Planning next milestone

## Current Position

Phase: Milestone v1.2 complete
Plan: —
Status: Awaiting next milestone
Last activity: 2026-06-29 — Milestone v1.2 completed and archived

## Performance Metrics

**Velocity (v1.0 + v1.1):**

- Total plans completed: 21
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
| 06 | 4 | - | - |
| 07 | TBD | - | - |
| 08 | 3 | - | - |
| 09 | 1 | - | - |
| 7 | 1 | - | - |

**Recent Trend:**

- Last 5 plans: —
- Trend: —

*Updated after each plan completion*
| Phase 04 P01 | 17 | 2 tasks | 4 files |
| Phase 04 P02 | 13 | 2 tasks | 3 files |
| Phase 06 P01 | 6 | 3 tasks | 5 files |
| Phase 06 P02 | 14 | 3 tasks | 12 files |
| Phase 06 P03 | 22 | 2 tasks | 15 files |
| Phase 06 P04 | 10 | 1 tasks | 25 files |
| Phase 07 P01 | 15 | 2 tasks | 2 files |

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
- [Phase ?]: localeProvider returns null when language is null so Flutter falls back to device locale (Phase 6 live-switch infra)
- [Phase ?]: unitProvider falls back to 'metric' on null settings or null unit, matching format_util.dart default
- [Phase ?]: [Phase 6 P02]: Units switch polarity — imperial = on-position, metric = off (UI-SPEC D-10)
- [Phase ?]: [Phase 6 P02]: RadioGroup<Language> with hardcoded native-name map (the single approved hardcoded-string exception)
- [Phase ?]: [Phase 6 P03]: Single ref.watch(unitProvider) per build path, threaded to private stat widgets/helpers via constructor or method param
- [Phase ?]: [Phase 6 P03]: elevation_profile stores unit in a _unit instance field set at top of build so fl_chart axis closures read the live value during the same build
- [Phase ?]: [Phase 7 P01]: Plural ARB getters use positional args (l10n.trail(2)) not named — plan/UI-SPEC syntax was wrong
- [Phase ?]: [Phase 7 P01]: Widget tests set a tall (1080x4000) viewport so all six long-subtitle tiles mount in the lazy ListView
- [Phase 08 P02]: PasswordChangeSheet payload must include oldPassword — PocketBase's native password-change rule enforces it; {password, passwordConfirm}-only payload returns 400
- [Phase 08 P02]: State.mounted used (not context.mounted) for async BuildContext guards in ConsumerState — IDE linter use_build_context_synchronously requires mounted check on the State object
- [Phase 08 P02]: PasswordChangeSheet uses generic l10n.error_updating_password (no server internals) per T-08-07; EmailChangeSheet surfaces ApiError.message only
- [Phase 08 P03]: _BioSection extracted as ConsumerStatefulWidget so outer ConsumerWidget can hold controller state without converting the full screen to StatefulWidget
- [Phase 08 P03]: hintText reuse means 'Add Bio' appears twice in widget tree (header + hint); test uses findsWidgets not findsOneWidget
- [Phase 08 P03]: Colors.red.shade400 for destructive foreground — colorScheme.error maps to #FEF2F2 (background token) which is illegible as foreground text
- [Phase 08 P03]: context.mounted (not mounted) used in ConsumerWidget helper methods — mounted refers to State object only in ConsumerState subclasses

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

Items acknowledged and deferred at milestone close on 2026-06-29 (12 total):

| Category | Item | Status |
|----------|------|--------|
| uat_gap | Phase 08: 08-UAT.md — 4 pending test scenarios | acknowledged |
| verification | Phase 06: 06-VERIFICATION.md — human_needed items | acknowledged |
| verification | Phase 07: 07-VERIFICATION.md — human_needed items | acknowledged |
| verification | Phase 08: 08-VERIFICATION.md — human_needed items | acknowledged |
| quick_task | 260610-kdc-fix-trail-pmtiles-download-add-missing-g | acknowledged |
| quick_task | 260611-whq-support-multiple-pmtiles-sources-in-offl | acknowledged |
| quick_task | 260612-gmg-add-proper-dark-mode-to-the-flutter-app | acknowledged |
| quick_task | 260615-k0w-implement-along-track-projection-for-way | acknowledged |
| quick_task | 260615-ktn-research-flutter-background-geolocation | acknowledged |
| quick_task | 260615-mxk-implement-background-navigation-so-locat | acknowledged |
| quick_task | 260616-h99-create-wandereractorsearch-component-for | acknowledged |
| quick_task | 260616-j2n-implement-the-like-feature-from-the-web | acknowledged |

## Session Continuity

Last session: 2026-06-21T09:28:39.566Z
Stopped at: Phase 9 context gathered
Resume file: .planning/phases/09-notifications/09-CONTEXT.md

## Operator Next Steps

- Start the next milestone with /gsd-new-milestone
