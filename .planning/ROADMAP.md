# Roadmap: Wanderer Trail Navigation

## Overview

**v1.0 (Complete):** Three phases deliver turn-by-turn trail navigation from nothing to a complete in-app experience. Phase 1 built the SvelteKit API endpoint that fetches Valhalla maneuvers. Phase 2 wired the Flutter navigation screen — entry points, full-screen map, GPS centering, maneuver display, orientation toggle, automatic advancement, and exit. Phase 3 added the DraggableScrollableSheet stats panel with live distance/elevation/speed stats and a reused elevation-profile page.

**v1.1 (Complete):** Two phases deliver offline navigation. Phase 4 fixed the serialization bug that blocked ObjectBox caching and added the `navCacheJson` field to `TrailEntity`. Phase 5 wired the cache write into the download service, added the DioException fallback in `launchNavigation`, fires a silent re-cache after successful online sessions, and shows an offline indicator in the NavigationScreen AppBar.

**v1.2 (Active):** Four phases port the web client's settings screens into the Flutter app. Phase 6 wires the settings navigation (Privacy, Language, Notifications list entries + routes) and ships the Language & Units screen as the foundational first screen. Phase 7 adds the Privacy screen with account/trails/lists visibility controls. Phase 8 builds the combined Account & Profile screen (avatar, bio, email, password, delete account). Phase 9 adds the Notifications screen with web + email toggles for all nine notification types.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Backend API** - SvelteKit POST /api/v1/valhalla/navigate endpoint returns structured maneuver list (completed 2026-06-12)
- [x] **Phase 2: Navigation Screen** - Full-screen Flutter navigation screen with map, maneuvers, GPS, and orientation toggle (completed 2026-06-13)
- [x] **Phase 3: Stats Sheet** - DraggableScrollableSheet with live distance/elevation/speed stats and a reused elevation-profile page (completed 2026-06-13)
- [x] **Phase 4: Serialization Fix + Entity Schema** - Fix NavigateResponse.toJson() serialization bug, add navCacheJson to TrailEntity (completed 2026-06-14)
- [x] **Phase 5: Cache Write + Fallback + UI** - Cache navigation instructions at download time, fall back to cache when offline, re-cache after online sessions, show offline indicator (completed 2026-06-14)
- [x] **Phase 6: Settings Navigation + Language & Units** - Add Privacy/Language/Notifications list entries and routes, ship the Language & Units screen (completed 2026-06-19)
- [x] **Phase 7: Privacy** - Privacy screen with account, trails, and lists visibility controls (completed 2026-06-20)
- [x] **Phase 8: Account & Profile** - Combined account screen: avatar, bio, change email, change password, delete account (completed 2026-06-20)
- [ ] **Phase 9: Notifications** - Notifications screen with web + email toggles for all nine notification types

## Phase Details

### Phase 1: Backend API

**Goal**: The SvelteKit navigate endpoint is live and returns structured Valhalla maneuvers for any trail
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: API-01, API-02
**Success Criteria** (what must be TRUE):

  1. POST /api/v1/valhalla/navigate accepts a GPX string or waypoint array and returns HTTP 200 with a maneuver list
  2. Each maneuver object in the response includes instruction text, distance to next turn, and bearing
  3. Invalid or missing input returns a descriptive error response (not a 500)

**Plans**: 1 planPlans:

- [x] 01-01-PLAN.md — Authenticated POST /api/v1/valhalla/navigate endpoint: Zod-validated waypoints, Valhalla transform to maneuver + shape contract, error/auth handling (TDD)

### Phase 2: Navigation Screen

**Goal**: Users can launch navigation from a trail screen and follow the trail with a live map and maneuver instructions
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: NAV-01, NAV-02, NAV-03, NAV-04, NAV-05, NAV-06, NAV-07, NAV-08
**Success Criteria** (what must be TRUE):

  1. User can tap a Navigate button on the trail detail screen or the trail detail map screen to open the navigation screen
  2. Navigation screen shows a full-screen map that centers and follows the user's current GPS position
  3. The current Valhalla maneuver instruction is displayed at the top of the screen
  4. A button on the map toggles between north-up and heading-up orientation
  5. Maneuvers advance automatically as the user moves along the trail without any manual interaction
  6. User can exit navigation and return to the screen they launched from
  7. A red breadcrumb polyline traces the user's actual traveled path during the session

**Plans**: 3 plansPlans:
**Wave 1**

- [x] 02-01-PLAN.md — NavigateResponse freezed model + Navigation notifier (maneuver auto-advancement + breadcrumb logic, unit-tested)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-02-PLAN.md — Full-screen NavigationScreen (follow map, maneuver banner, compass toggle, breadcrumb, exit, recenter) + navigate sub-route + i18n

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 02-03-PLAN.md — Navigate-button entry points on trail detail + map screens (Dio POST, costing, loading, toast, push-with-extra)

**UI hint**: yes

### Phase 3: Stats Sheet

**Goal**: Users can see live navigation statistics — distance, elevation, and speed — in a draggable bottom sheet during navigation
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: STATS-01, STATS-02, STATS-03, STATS-04, STATS-05
**Success Criteria** (what must be TRUE):

> Design note: 03-CONTEXT.md (locked) overrides the original "4 horizontal-swipe pages" wording below. The realized design is a single DraggableScrollableSheet with a collapsed/expanded state plus a button-driven 2-page PageView (live stats + reused elevation profile). Horizontal swipe is disabled; the requirement intent (show distance, elevation, speed) is fully met.

  1. A DraggableScrollableSheet is visible at the bottom of the navigation screen and can be dragged between a collapsed and expanded snap point (STATS-01)
  2. Collapsed sheet shows Time elapsed, Distance covered, and Elevation gain (STATS-02, STATS-03, STATS-04)
  3. Expanded sheet adds Elevation loss, Current speed, and Average speed (STATS-04, STATS-05)
  4. A button-driven PageView switches to the reused trail elevation-profile chart and back (STATS-01)
  5. A button row provides Elevation profile, Pause/Resume, and Exit; the old top-left exit overlay is removed

**Plans**: 2 plansPlans:
**Wave 1**

- [x] 03-01-PLAN.md — navigationStatsProvider (@riverpod + freezed NavigationStats; distance/elevation/speed accumulation, pause/resume, elapsed timer) + format_util formatSpeed/formatElapsed, unit-tested

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 03-02-PLAN.md — Stats sheet UI in NavigationScreen (DraggableScrollableSheet + button-driven PageView + Pause/Exit button row, GPS-fed stats, reused ElevationProfile) + i18n; old exit overlay removed

**UI hint**: yes

### Phase 4: Serialization Fix + Entity Schema

**Goal**: The infrastructure required for ObjectBox caching is in place — NavigateResponse serializes correctly and TrailEntity has the navCacheJson cache field (the shape helper is NOT extracted per locked decision D-05; the existing `shapeAsLatLng` extension already serves both online and cache paths)
**Depends on**: Phase 3
**Requirements**: (prerequisite phase — all OFFLINE-xx requirements deliver in Phase 5)
**Success Criteria** (what must be TRUE):

  1. A roundtrip unit test passes: `jsonEncode(response.toJson())` followed by `NavigateResponse.fromJson(jsonDecode(...))` reconstructs all maneuver fields without throwing
  2. `objectbox-model.json` contains a `navCacheJson` property entry under `TrailEntity` after build_runner runs
  3. Existing navigation flows (online path) are unaffected — all Phase 2 and Phase 3 behaviors still work

**Plans**: 2 plans
**Wave 1**

- [x] 04-01-PLAN.md — Fix NavigateResponse serialization: add @JsonSerializable(explicitToJson: true), regenerate .g.dart, add roundtrip test group (full/empty/minimal cases)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 04-02-PLAN.md — Add String? navCacheJson to TrailEntity (entity-only, gpxData precedent), regenerate + commit objectbox-model.json, verify Trail model / fromModel / toModel untouched + full analyze/test pass

### Phase 5: Cache Write + Fallback + UI

**Goal**: Hikers can follow downloaded trails step-by-step without a network connection, and the app silently keeps the cache current after each online session
**Depends on**: Phase 4
**Requirements**: OFFLINE-01, OFFLINE-02, OFFLINE-03, OFFLINE-04
**Success Criteria** (what must be TRUE):

  1. After a trail is downloaded, navigation can be launched without a network connection and the maneuver list is served from ObjectBox (OFFLINE-01, OFFLINE-02)
  2. When the network call succeeds, navigation launches normally with no user-visible change; the local cache is silently updated for future offline use (OFFLINE-03)
  3. When navigation falls back to the cache, a distinct icon appears in the NavigationScreen AppBar indicating offline mode (OFFLINE-04)
  4. A Valhalla outage during trail download does not block or error the download — the cache step is best-effort and silent (OFFLINE-01)

**Plans**: 4 plans
**Wave 1**

- [x] 05-01-PLAN.md — Extract shared buildNavShape downsampling helper into gpx_util.dart + unit tests (foundational contract for OFFLINE-01/02)
- [x] 05-02-PLAN.md — NavigationScreen isOffline param + wifi-off banner icon + navigate route (NavigateResponse, bool) record unpacking (OFFLINE-04)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 05-03-PLAN.md — launchNavigation DioException cache fallback + isOffline propagation + unawaited silent re-cache (OFFLINE-02, OFFLINE-03)
- [x] 05-04-PLAN.md — Best-effort silent Valhalla cache write in TrailDownloadService.downloadTrail (OFFLINE-01)

### Phase 6: Settings Navigation + Language & Units

**Goal**: Users can reach every settings sub-screen from the Settings list and can set their preferred language and unit system
**Depends on**: Nothing new (extends existing settings infrastructure; first phase of v1.2)
**Requirements**: SETNAV-01, LANG-01, LANG-02
**Success Criteria** (what must be TRUE):

  1. The Settings screen lists entries for Account, Privacy, Language, and Notifications alongside the existing Appearance entry, and each entry navigates to its screen
  2. Tapping Language opens a screen where the user can pick from the 14 supported locales, and the selection persists after leaving and reopening the screen
  3. The Language screen lets the user toggle between metric and imperial units, and the choice persists across app restarts
  4. Changing language or units saves to the server and the existing Settings, Account, and Appearance screens remain reachable and unbroken

**Plans**: 4 plans
Plans:
**Wave 1**

- [x] 06-01-PLAN.md — localeProvider + unitProvider derived providers, MaterialApp.router live-locale wiring + 14 supportedLocales, Wave 0 test scaffolds (LANG-01, LANG-02)

**Wave 2** *(blocked on Wave 1 completion; the three Wave 2 plans run in parallel — no file overlap)*

- [x] 06-02-PLAN.md — Settings nav rows + 3 routes + Privacy/Notifications stubs + Language & Units screen (SETNAV-01, LANG-01, LANG-02)
- [x] 06-03-PLAN.md — Wire unitProvider into all ~14 format_util call sites (incl. 3 non-Consumer conversions) + imperial format tests (LANG-02)
- [x] 06-04-PLAN.md — Port 12 missing locale ARB files from web JSON + regenerate AppLocalizations (LANG-01)

**UI hint**: yes

### Phase 7: Privacy

**Goal**: Users can control the default visibility of their account, trails, and lists from a dedicated Privacy screen
**Depends on**: Phase 6
**Requirements**: PRIV-01, PRIV-02, PRIV-03
**Success Criteria** (what must be TRUE):

  1. Tapping Privacy from the Settings screen opens the Privacy screen
  2. User can set account visibility to public or private, and the selection is saved and reflected when the screen is reopened
  3. User can set the default trails visibility to public or private, and the selection persists
  4. User can set the default lists visibility to public or private, and the selection persists

**Plans**: 1 plan
**Wave 1**

- [x] 07-01-PLAN.md — Fill SettingsPrivacyScreen stub: three RadioGroup<String> visibility sections (account/trails/lists) with subtitle descriptions + auto-save via settingsProvider; mirrors SettingsLanguageScreen (PRIV-01, PRIV-02, PRIV-03) + widget test

**UI hint**: yes

### Phase 8: Account & Profile

**Goal**: Users can manage their profile and account credentials — avatar, bio, email, password, and account deletion — from a single Account screen
**Depends on**: Phase 6
**Requirements**: ACCT-01, ACCT-02, ACCT-03, ACCT-04, ACCT-05
**Success Criteria** (what must be TRUE):

  1. User can upload or replace their avatar and the new image is shown after saving
  2. User can edit and save their bio, and the updated text persists after reopening the screen
  3. User can change their email address, with the change taking effect after saving
  4. User can change their password by providing the required credentials, and a clear error is shown if the change is rejected
  5. User can delete their account, but only after passing an explicit confirmation step

**Plans**: 3 plans
Plans:
**Wave 1**

- [x] 08-01-PLAN.md — Foundation: add image_picker + iOS photo-library permission, add `account` l10n key, add public `Auth.refresh()` (ACCT-01, ACCT-03)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 08-02-PLAN.md — EmailChangeSheet + PasswordChangeSheet bottom-sheet forms (FormBuilder + WandererTextField; email re-fetch, password oldPassword fix) (ACCT-03, ACCT-04)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 08-03-PLAN.md — Fill SettingsAccountScreen: avatar upload, change-aware bio Save, email/password sheet rows, confirm-gated delete + widget test (ACCT-01..05)

**UI hint**: yes

### Phase 9: Notifications

**Goal**: Users can independently toggle web and email delivery for each of the nine notification types from a Notifications screen
**Depends on**: Phase 6
**Requirements**: NOTIF-01, NOTIF-02, NOTIF-03, NOTIF-04, NOTIF-05, NOTIF-06, NOTIF-07, NOTIF-08, NOTIF-09
**Success Criteria** (what must be TRUE):

  1. Tapping Notifications from the Settings screen opens a screen listing all nine notification types (trail comments, new followers, trail shares, trail likes, list shares, summit log creates, trail mentions, comment mentions, summit log mentions)
  2. Each notification type exposes an independent web toggle and email toggle
  3. Toggling any switch saves the change to the server and the new state persists after the screen is reopened

**Plans**: TBD
**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9

Phases 7, 8, and 9 each depend only on Phase 6 and are otherwise independent of one another.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Backend API | 1/1 | Complete | 2026-06-12 |
| 2. Navigation Screen | 3/3 | Complete | 2026-06-13 |
| 3. Stats Sheet | 2/2 | Complete | 2026-06-13 |
| 4. Serialization Fix + Entity Schema | 2/2 | Complete | 2026-06-14 |
| 5. Cache Write + Fallback + UI | 4/4 | Complete | 2026-06-14 |
| 6. Settings Navigation + Language & Units | 4/4 | Complete    | 2026-06-20 |
| 7. Privacy | 1/1 | Complete    | 2026-06-20 |
| 8. Account & Profile | 3/3 | Complete    | 2026-06-20 |
| 9. Notifications | 0/0 | Not started | - |
