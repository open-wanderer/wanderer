# Wanderer Trail Navigation

## What This Is

Turn-by-turn trail navigation and full settings management for the Wanderer Flutter mobile app. Users launch navigation from a trail's detail or map screen and get Valhalla-powered maneuver instructions, a live map centered on their position, and a stats sheet tracking distance, elevation, and speed. The app also includes a complete settings suite: language picker (14 locales), metric/imperial unit toggle, privacy controls, account/profile management, and per-type notification preferences.

## Core Value

A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## Requirements

### Validated

- [x] Navigate button on trail_detail_screen and trail_detail_map_screen (v1.0 — Phase 2)
- [x] Full-screen navigation screen with map centered on current GPS position (v1.0 — Phase 2)
- [x] Valhalla turn-by-turn maneuver instructions shown at the top of the screen (v1.0 — Phase 2)
- [x] North-up / heading-up map orientation toggle button (v1.0 — Phase 2)
- [x] DraggableScrollableSheet with live stats: distance, elevation gain/loss, speed (v1.0 — Phase 3)
- [x] New SvelteKit API endpoint POST /api/v1/valhalla/navigate (v1.0 — Phase 1)
- [x] Progress tracking: advance through maneuvers as user moves along the trail (v1.0 — Phase 2)
- [x] Offline navigation fallback: serve maneuvers from ObjectBox cache when network is unavailable (v1.1 — Phase 5)
- [x] Silent cache write at trail download time and re-cache after online sessions (v1.1 — Phase 5)
- [x] SettingsScreen: Privacy, Language, Notifications list entries with sub-routes (v1.2 — Phase 6)
- [x] SettingsLanguageScreen: 14-locale language picker + metric/imperial unit toggle, persists and auto-saves (v1.2 — Phase 6)
- [x] Unit toggle propagates to all format_util call sites app-wide (v1.2 — Phase 6)
- [x] SettingsPrivacyScreen: account, trails, and lists visibility radio groups, auto-save (v1.2 — Phase 7)
- [x] SettingsAccountScreen: avatar upload, bio editor, email change, password change, account deletion with confirmation (v1.2 — Phase 8)
- [x] SettingsNotificationsScreen: 9 notification types × web + email toggles, auto-save (v1.2 — Phase 9)

### Active

_(none — planning next milestone)_

### Out of Scope

- Text-to-speech maneuver announcements — deferred to v2 (audio infra adds complexity)
- Routing from user's current position to the trailhead — assume user is already at the trail start
- Re-routing if user goes off-trail
- API token management (ACCT-F01) — mobile clients don't need API tokens; web-only feature
- Favourite sport picker, Export, Integrations (Strava/Komoot), Maintenance, Map settings — out of scope for mobile settings v1

## Context

- Flutter mobile app with Riverpod state management, go_router navigation, flutter_map for maps, geolocator for GPS
- Shipped v1.0: Navigation screen (SvelteKit Valhalla endpoint + Flutter screen + stats sheet)
- Shipped v1.1: Offline navigation (ObjectBox cache, DioException fallback, offline indicator)
- Shipped v1.2: Full settings suite (Language/Units, Privacy, Account/Profile, Notifications)
- Settings infrastructure: `Settings` freezed model, `settingsProvider` with `saveToServer()` — reused by all four screens
- `localeProvider` and `unitProvider` derived from `settingsProvider` — live-switch locale and unit system app-wide
- 14 supported locales with ARB files; AppLocalizations regenerated in Phase 6
- Background navigation via tracelet package (from quick tasks)
- Along-track projection for waypoint advancement (from quick tasks)

## Constraints

- **Tech Stack**: Flutter + Riverpod (riverpod_annotation codegen) + go_router + flutter_map + freezed — must follow existing patterns
- **API**: SvelteKit app proxies Valhalla at POST /api/v1/valhalla/navigate; Flutter calls via Dio
- **Online-only navigation v1**: Navigation requires network; falls back to ObjectBox cache when offline
- **No breaking changes**: Existing trail detail screens, bottom nav, and routes must be unaffected

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Extend SvelteKit Valhalla API rather than calling Valhalla directly from Flutter | Keeps credentials server-side, consistent with existing route endpoint pattern | ✓ Good — used in v1.0 through v1.2 |
| DraggableScrollableSheet for stats, not fixed bottom bar | Matches MapScreen pattern; user can expand for more detail without blocking map | ✓ Good |
| Button-driven PageView for stats (not horizontal swipe) | Locked during Phase 3 context; horizontal swipe conflicts with map pan gesture | ✓ Good |
| TTS deferred to v2 | Adds Flutter TTS dependency and audio session handling — not needed for navigation core | — Pending |
| Assume user is at trailhead | Simplifies v1 scope; off-trail routing is a separate problem | — Pending |
| Dio try-catch as sole offline gate (not connectivity packages) | Connectivity packages have false-positive/negative failure modes (captive portals, VPN) | ✓ Good |
| `String? navCacheJson` on TrailEntity (not typed List) | ObjectBox doesn't support List<List<double>> or List<NavigateManeuver> natively; follows `gpxData` precedent | ✓ Good |
| freezed 3.x: @JsonSerializable(explicitToJson: true) on factory constructor | Class-level placement breaks json_serializable codegen in freezed 3.x | ✓ Good — critical for future freezed models |
| `Settings` freezed model + `settingsProvider.saveToServer()` shared across all v1.2 screens | No new persistence layer needed; single source of truth | ✓ Good |
| RadioGroup<Language> with hardcoded native-name map | Only approved exception to the no-hardcoded-strings rule; localized names require the locale itself to render | ✓ Good |
| Imperial = on-position for unit switch (UI-SPEC D-10) | Toggle polarity chosen to match natural reading: "imperial is the non-default" | ✓ Good |
| `State.mounted` in ConsumerState, `context.mounted` in ConsumerWidget helpers | `mounted` refers to State object — only available in ConsumerState; ConsumerWidget uses context.mounted | ✓ Good — important Flutter/Riverpod gotcha |
| Colors.red.shade400 for destructive foreground (not colorScheme.error) | colorScheme.error maps to #FEF2F2 (background token) — illegible as foreground text | ✓ Good |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-06-29 after v1.2 milestone — Settings Screens shipped (Language/Units, Privacy, Account/Profile, Notifications). All 15 cumulative requirements validated.*
