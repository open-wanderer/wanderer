# Wanderer Trail Navigation

## What This Is

Turn-by-turn trail navigation for the Wanderer Flutter mobile app. Users launch navigation from a trail's detail or map screen and get Valhalla-powered maneuver instructions, a live map centered on their position, and a stats sheet tracking distance, elevation, and speed — all in one focused screen.

## Core Value

A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## Current Milestone: v1.2 Settings Screens

**Goal:** Port the settings screens from the web client into the Flutter app so users can manage profile, account credentials, privacy, language/units, and notification preferences on mobile.

**Target features:**
- Combined Account + Profile screen: avatar upload, bio editor, change email, change password, delete account
- Privacy screen: account/trails/lists visibility radio groups
- Language screen: 14-language picker + metric/imperial unit toggle
- Notifications screen: 9 notification types × web + email toggles
- Settings screen: add Privacy, Language, Notifications list entries

## Requirements

### Validated

- [x] Navigate button on trail_detail_screen and trail_detail_map_screen (v1.0 — Phase 2)
- [x] Full-screen navigation screen with map centered on current GPS position (v1.0 — Phase 2)
- [x] Valhalla turn-by-turn maneuver instructions shown at the top of the screen (v1.0 — Phase 2)
- [x] North-up / heading-up map orientation toggle button (v1.0 — Phase 2)
- [x] DraggableScrollableSheet with live stats: distance, elevation gain/loss, speed (v1.0 — Phase 3)
- [x] New SvelteKit API endpoint POST /api/v1/valhalla/navigate (v1.0 — Phase 1)
- [x] Progress tracking: advance through maneuvers as user moves along the trail (v1.0 — Phase 2)

### Active

- [ ] SettingsAccountScreen: avatar upload, bio editor, change email, change password, delete account
- [ ] SettingsPrivacyScreen: account, trails, and lists visibility toggles
- [ ] SettingsLanguageScreen: language picker and metric/imperial unit toggle
- [ ] SettingsNotificationsScreen: per-type web and email notification toggles
- [ ] SettingsScreen: add Privacy, Language, Notifications list entries

### Out of Scope

- Text-to-speech maneuver announcements — deferred to v2 (audio infra adds complexity)
- Routing from user's current position to the trailhead — assume user is already at the trail start
- Re-routing if user goes off-trail

## Context

- Flutter mobile app with Riverpod state management, go_router navigation, flutter_map for maps, geolocator for GPS
- SvelteKit web app already proxies Valhalla routing via POST /api/v1/valhalla/route — thin proxy, passes through raw Valhalla JSON
- Trail model has `gpxData` (raw string) and `gpx` (parsed Gpx object) plus `waypointsViaTrail`
- Valhalla returns structured maneuver objects with type, instruction text, length, and bearing — these are the turn-by-turn instructions
- MapScreen already uses AnimatedMapController, CurrentLocationLayer, vector tiles — navigation screen will reuse the same map stack
- flutter_map_location_marker already integrated for live position display
- Existing stats (elevation profile, distance, duration) computed from GPX on trail detail screens — navigation screen will track live equivalents

## Constraints

- **Tech Stack**: Flutter + Riverpod (riverpod_annotation codegen) + go_router + flutter_map + freezed — must follow existing patterns
- **API**: New endpoint at POST /api/v1/valhalla/navigate in the SvelteKit app; Flutter calls it via existing dio HTTP client
- **Online-only v1**: Navigation requires a network connection to fetch Valhalla instructions
- **No breaking changes**: Existing trail detail screens, bottom nav, and routes must be unaffected

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Extend SvelteKit Valhalla API rather than calling Valhalla directly from Flutter | Keeps credentials server-side, consistent with existing route endpoint pattern | — Pending |
| DraggableScrollableSheet for stats, not fixed bottom bar | Matches MapScreen pattern; user can expand for more detail without blocking map | — Pending |
| Horizontal swipe pagination for stats | Keeps the sheet compact by default while showing all metrics | — Pending |
| TTS deferred to v2 | Adds dependency on Flutter TTS package and audio session handling — not needed for navigation core | — Pending |
| Assume user is at trailhead | Simplifies v1 scope; off-trail routing is a separate problem | — Pending |

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
*Last updated: 2026-06-21 — Phase 9 complete: Notifications settings sub-screen shipped (9 notification types, independent Web/Email toggles, auto-save, "web" ARB key). v1.2 Settings milestone complete.*
