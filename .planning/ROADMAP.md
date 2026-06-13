# Roadmap: Wanderer Trail Navigation

## Overview

Three phases deliver turn-by-turn trail navigation from nothing to a complete in-app experience. Phase 1 builds the SvelteKit API endpoint that fetches Valhalla maneuvers. Phase 2 wires the Flutter navigation screen — entry points, full-screen map, GPS centering, maneuver display, orientation toggle, automatic advancement, and exit. Phase 3 adds the DraggableScrollableSheet stats panel with live distance/elevation/speed stats and a reused elevation-profile page.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Backend API** - SvelteKit POST /api/v1/valhalla/navigate endpoint returns structured maneuver list (completed 2026-06-12)
- [x] **Phase 2: Navigation Screen** - Full-screen Flutter navigation screen with map, maneuvers, GPS, and orientation toggle (completed 2026-06-13)
- [x] **Phase 3: Stats Sheet** - DraggableScrollableSheet with live distance/elevation/speed stats and a reused elevation-profile page (completed 2026-06-13)

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

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Backend API | 1/1 | Complete    | 2026-06-12 |
| 2. Navigation Screen | 3/3 | Complete   | 2026-06-13 |
| 3. Stats Sheet | 2/2 | Complete    | 2026-06-13 |
